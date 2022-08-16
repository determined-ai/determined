import json
import subprocess
import time
from typing import Any, Dict

import pytest
import requests
from typing_extensions import Literal

from determined.common import api
from determined.common.api import authentication, certs
from tests import config as conf


def cluster_slots() -> Dict[str, Any]:
    """
    cluster_slots returns a dict of slots that each agent has.
    :return:  Dict[AgentID, List[Slot]]
    """
    # TODO: refactor tests to not use cli singleton auth.
    certs.cli_cert = certs.default_load(conf.make_master_url())
    authentication.cli_auth = authentication.Authentication(conf.make_master_url(), try_reauth=True)
    r = api.get(conf.make_master_url(), "api/v1/agents")
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


def num_slots() -> int:
    return sum(len(agent_slots) for agent_slots in cluster_slots().values())


def num_free_slots() -> int:
    return sum(
        0 if slot["container"] else 1
        for agent_slots in cluster_slots().values()
        for slot in agent_slots
    )


def run_command_set_priority(sleep: int = 30, slots: int = 1, priority: int = 0) -> str:
    command = [
        "det",
        "-m",
        conf.make_master_url(),
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
    return subprocess.check_output(command).decode().strip()


def run_command(sleep: int = 30, slots: int = 1) -> str:
    command = [
        "det",
        "-m",
        conf.make_master_url(),
        "command",
        "run",
        "-d",
        "--config",
        f"resources.slots={slots}",
        "sleep",
        str(sleep),
    ]
    return subprocess.check_output(command).decode().strip()


def run_zero_slot_command(sleep: int = 30) -> str:
    return run_command(sleep=sleep, slots=0)


TaskType = Literal["command", "notebook", "tensorboard", "shell"]


def get_task_info(task_type: TaskType, task_id: str) -> Dict[str, Any]:
    task = ["det", "-m", conf.make_master_url(), task_type, "list", "--json"]
    task_data = json.loads(subprocess.check_output(task).decode())
    return next((d for d in task_data if d["id"] == task_id), {})


def get_command_info(command_id: str) -> Dict[str, Any]:
    return get_task_info("command", command_id)


def command_succeeded(command_id: str) -> bool:
    print(get_command_info(command_id))

    return "success" in get_command_info(command_id)["exitStatus"]


def wait_for_task_state(task_type: TaskType, task_id: str, state: str, ticks: int = 60) -> None:
    for _ in range(ticks):
        info = get_task_info(task_type, task_id)
        gotten_state = info.get("state")
        if gotten_state == state:
            return
        time.sleep(1)

    print(subprocess.check_output(["det", "-m", conf.make_master_url(), "task", "logs", task_id]))
    pytest.fail(f"{task_type} expected {state} state got {gotten_state} instead after {ticks} secs")


def wait_for_command_state(command_id: str, state: str, ticks: int = 60) -> None:
    return wait_for_task_state("command", command_id, state, ticks)
