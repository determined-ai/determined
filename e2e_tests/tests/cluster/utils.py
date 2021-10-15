from typing import Any, Dict

import requests

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
