import os

import pytest

import determined
from determined.common import api
from tests import config as conf


# TODO: This should be marked as a cross-version test, but it can't actually be at the time of
# writing, since older agent versions don't report their versions.
@pytest.mark.e2e_cpu
def test_agent_version() -> None:
    # DET_AGENT_VERSION is available and specifies the agent version in cross-version tests; for
    # other tests, this evaluates to the current version.
    target_version = os.environ.get("DET_AGENT_VERSION") or determined.__version__
    agents = api.get(conf.make_master_url(), "agents").json()

    assert all(agent["version"] == target_version for agent in agents.values())
