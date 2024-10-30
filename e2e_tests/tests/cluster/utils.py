import copy
import datetime
import http.server
import sys
import threading
import time
from typing import Any, Dict, List, Optional, Tuple, Type

import pytest
import requests
from typing_extensions import Literal  # noqa:I2041

from determined.common import api
from determined.common.api import bindings
from tests import command
from tests import config as conf
from tests import detproc

KUBERNETES_EXPERIMENT_TIMEOUT = 600


class _HTTPServerWithRequest(http.server.HTTPServer):
    def __init__(
        self,
        server_address: Tuple[str, int],
        RequestHandlerClass: Type[http.server.SimpleHTTPRequestHandler],
        allow_dupes: bool,
    ):
        super().__init__(server_address, RequestHandlerClass)
        self.url_to_request_body: Dict[str, str] = {}
        self.url_to_request_body_lock = threading.Lock()
        self.allow_dupes = allow_dupes


class _WebhookRequestHandler(http.server.SimpleHTTPRequestHandler):
    def do_POST(self) -> None:
        assert isinstance(self.server, _HTTPServerWithRequest)
        with self.server.url_to_request_body_lock:
            if self.path in self.server.url_to_request_body and not self.server.allow_dupes:
                print(self.server.url_to_request_body, self.path)
                pytest.fail(f"got two webhooks sent to path {self.path}")

            content_length = int(self.headers.get("content-length"))
            request_body = self.rfile.read(content_length)

            self.server.url_to_request_body[self.path] = request_body.decode("utf-8")

            self.send_response(200, "Success")
            self.end_headers()
            self.wfile.write("".encode("utf-8"))


class WebhookServer:
    def __init__(self, port: int, allow_dupes: bool = False):
        self.server = _HTTPServerWithRequest(("", port), _WebhookRequestHandler, allow_dupes)

        self.server_thread = threading.Thread(target=self.server.serve_forever)
        self.server_thread.start()

    def return_responses(self) -> Dict[str, str]:
        with self.server.url_to_request_body_lock:
            return copy.deepcopy(self.server.url_to_request_body)

    def close_and_return_responses(self) -> Dict[str, str]:
        self.server.shutdown()
        self.server.server_close()
        self.server_thread.join()
        return self.server.url_to_request_body


def cluster_slots(sess: api.Session) -> Dict[str, Any]:
    """
    cluster_slots returns a dict of slots that each agent has.
    :return:  Dict[AgentID, List[Slot]]
    """
    r = sess.get("api/v1/agents")
    assert r.status_code == requests.codes.ok, r.text
    jvals = r.json()  # type: Dict[str, Any]
    return {agent["id"]: agent["slots"].values() for agent in jvals["agents"]}


def get_master_port(loaded_config: dict) -> str:
    for d in loaded_config["stages"]:
        for k in d.keys():
            if k == "master":
                if "port" in d["master"]["config_file"]:
                    return str(d["master"]["config_file"]["port"])

    return "8080"  # default value if not explicit in config file


def num_slots(sess: api.Session) -> int:
    return sum(len(agent_slots) for agent_slots in cluster_slots(sess).values())


def num_free_slots(sess: api.Session) -> int:
    return sum(
        0 if slot["container"] else 1
        for agent_slots in cluster_slots(sess).values()
        for slot in agent_slots
    )


def run_command_set_priority(
    sess: api.Session, sleep: int, slots: int = 1, priority: int = 0
) -> str:
    cmd = [
        "det",
        "command",
        "run",
        "-d",
        "--config",
        f"resources.slots={slots}",
        "--config",
        f"resources.priority={priority}",
        "sleep",
        str(sleep),
    ]
    return detproc.check_output(sess, cmd).strip()


def run_command(sess: api.Session, sleep: int, slots: int = 1) -> str:
    cmd = [
        "det",
        "command",
        "run",
        "-d",
        "--config",
        f"resources.slots={slots}",
        "sleep",
        str(sleep),
    ]
    return detproc.check_output(sess, cmd).strip()


def run_command_args(sess: api.Session, entrypoint: str, args: Optional[List[str]]) -> str:
    cmd = [
        "det",
        "command",
        "run",
        "-d",
    ]
    if args:
        cmd += args
    return detproc.check_output(sess, cmd + [entrypoint]).strip()


def run_zero_slot_command(sess: api.Session, sleep: int) -> str:
    return run_command(sess, sleep=sleep, slots=0)


TaskType = Literal["command", "notebook", "tensorboard", "shell"]


def get_task_info(sess: api.Session, task_type: TaskType, task_id: str) -> Dict[str, Any]:
    cmd = ["det", task_type, "list", "--json"]
    task_data = detproc.check_json(sess, cmd)
    return next((d for d in task_data if d["id"] == task_id), {})


def get_command_info(sess: api.Session, command_id: str) -> Dict[str, Any]:
    return get_task_info(sess, "command", command_id)


# assert_command_succeded checks if a command succeeded or not. It prints the command logs if the
# command failed.
def assert_command_succeeded(sess: api.Session, command_id: str) -> None:
    command_info = get_command_info(sess, command_id)
    succeeded = "success" in command_info["exitStatus"]
    assert succeeded, command.print_command_logs(sess, command_id)


def wait_for_task_state(
    sess: api.Session, task_type: TaskType, task_id: str, state: str, ticks: int = 60
) -> None:
    for _ in range(ticks):
        info = get_task_info(sess, task_type, task_id)
        gotten_state = info.get("state")
        if gotten_state == state:
            return
        time.sleep(1)

    print("== begin task logs ==", file=sys.stderr)
    print(detproc.check_output(sess, ["det", "task", "logs", task_id]), file=sys.stderr)
    print("== end task logs ==", file=sys.stderr)
    pytest.fail(f"{task_type} expected {state} state got {gotten_state} instead after {ticks} secs")


def wait_for_command_state(sess: api.Session, command_id: str, state: str, ticks: int = 60) -> None:
    return wait_for_task_state(sess, "command", command_id, state, ticks)


def now_ts() -> str:
    return datetime.datetime.now(datetime.timezone.utc).astimezone().isoformat()


def set_master_port(config: str) -> None:
    lc = conf.load_config(config_path=config)
    port = get_master_port(lc)
    conf.MASTER_PORT = port


def get_run_by_exp_id(sess: api.Session, exp_id: int) -> bindings.v1SearchRunsResponse:
    return bindings.post_SearchRuns(
        sess,
        body=bindings.v1SearchRunsRequest(
            limit=1,
            filter="""{
            "filterGroup": {
                "children": [
                {
                    "columnName": "experimentId",
                    "kind": "field",
                    "location": "LOCATION_TYPE_RUN",
                    "operator": "=",
                    "type": "COLUMN_TYPE_NUMBER",
                    "value": %s
                }
                ],
                "conjunction": "and",
                "kind": "group"
            },
            "showArchived": false
            }"""
            % exp_id,
        ),
    )
