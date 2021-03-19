import subprocess
import time
from typing import Any, Dict, List

import requests

import determined.common.api.authentication as auth
from determined.common import api
from tests import config as conf


def det_version() -> str:
    output = subprocess.check_output(["det", "--version"], universal_newlines=True)  # type: str
    return output.split()[1]


def cluster_slots() -> Dict[str, Any]:
    """
    cluster_slots returns a dict of slots that each agent has.
    :return:  Dict[AgentID, List[Slot]]
    """
    auth.initialize_session(conf.make_master_url(), try_reauth=True)
    r = api.get(conf.make_master_url(), "agents")
    assert r.status_code == requests.codes.ok, r.text
    json = r.json()  # type: Dict[str, Any]
    return {agent["id"]: agent["slots"].values() for agent in json.values()}


def num_slots() -> int:
    return sum(len(agent_slots) for agent_slots in cluster_slots().values())


def max_slots_per_agent() -> int:
    return max(map(len, cluster_slots().values()))


def gpu_slots_per_agent() -> List[int]:
    return [
        sum(1 if slot["type"] == "gpu" else 0 for slot in slot_list)
        for slot_list in cluster_slots().values()
    ]


def num_free_slots() -> int:
    return sum(
        0 if slot["container"] else 1
        for agent_slots in cluster_slots().values()
        for slot in agent_slots
    )


def running_on_gpu() -> bool:
    return any(
        slot["device"]["type"] == "gpu"
        for slot_list in cluster_slots().values()
        for slot in slot_list
    )


def wait_for_agents(min_agent_count: int) -> None:
    while True:
        if num_agents() >= min_agent_count:
            return

        print("Waiting for {} agents to register...".format(min_agent_count))
        time.sleep(1)


def num_agents() -> int:
    auth.initialize_session(conf.make_master_url(), try_reauth=True)
    r = api.get(conf.make_master_url(), "agents")
    assert r.status_code == requests.codes.ok, r.text

    return len(r.json())
