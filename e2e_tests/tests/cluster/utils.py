import json
import subprocess
import time
from typing import Any, Dict

import pytest
import requests

from determined.common import api, constants
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
    r = api.get(conf.make_master_url(), "agents")
    assert r.status_code == requests.codes.ok, r.text
    json = r.json()  # type: Dict[str, Any]
    return {agent["id"]: agent["slots"].values() for agent in json.values()}


def num_slots() -> int:
    return sum(len(agent_slots) for agent_slots in cluster_slots().values())


def num_free_slots() -> int:
    return sum(
        0 if slot["container"] else 1
        for agent_slots in cluster_slots().values()
        for slot in agent_slots
    )


def run_command(sleep: int = 30, slots: int = 1) -> str:
    command = [
        "det",
        "-m",
        conf.make_master_url(),
        "-u",
        constants.DEFAULT_DETERMINED_USER,
        "command",
        "run",
        "-d",
        "--config",
        "resources.slots=0",
        "sleep",
        str(sleep),
    ]
    return subprocess.check_output(command).decode().strip()


def run_zero_slot_command(sleep: int = 30) -> str:
    return run_command(sleep=sleep, slots=0)


def get_command_info(command_id: str) -> Dict[str, Any]:
    command = [
        "det",
        "-m",
        conf.make_master_url(),
        "-u",
        constants.DEFAULT_DETERMINED_USER,
        "command",
        "list",
        "--json",
    ]
    command_data = json.loads(subprocess.check_output(command).decode())
    return next((d for d in command_data if d["id"] == command_id), {})


def wait_for_command_state(command_id: str, state: str, ticks: int = 60) -> None:
    for _ in range(ticks):
        info = get_command_info(command_id)
        if info.get("state") == state:
            return
        time.sleep(1)

    pytest.fail(f"Command did't reach {state} state after {ticks} secs")
