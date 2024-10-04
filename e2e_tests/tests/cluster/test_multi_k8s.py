from typing import Any, Dict, Optional

import pytest

from determined.common import api
from determined.common.api import bindings
from determined.experimental import client
from tests import api_utils, detproc
from tests import experiment as exp
from tests.cluster import utils
from tests.experiment import noop

MAX_WAIT_TIME = 500  # Really long since minikube will need to pull images.


@pytest.mark.e2e_multi_k8s
@pytest.mark.parametrize(
    "resource_pool, expected_node", [(None, "defaultrm"), ("additional_pool", "additionalrm")]
)
def test_run_experiment_multi_k8s(resource_pool: Optional[str], expected_node: str) -> None:
    config: Dict[str, Any] = {"entrypoint": "echo RunningOnNode=$DET_AGENT_ID"}
    if resource_pool:
        config["resources"] = {"resource_pool": resource_pool}

    sess = api_utils.user_session()
    exp_ref = noop.create_experiment(sess, config=config)
    assert exp_ref.wait(interval=0.01) == client.ExperimentState.COMPLETED
    exp.assert_patterns_in_trial_logs(
        sess, exp.experiment_first_trial(sess, exp_ref.id), [f"RunningOnNode={expected_node}"]
    )


@pytest.mark.e2e_multi_k8s
@pytest.mark.parametrize(
    "resource_pool, expected_node", [(None, "defaultrm"), ("additional_pool", "additionalrm")]
)
def test_run_command_multi_k8s(resource_pool: Optional[str], expected_node: str) -> None:
    sess = api_utils.user_session()
    args = (
        None if resource_pool is None else ["--config", f"resources.resource_pool={resource_pool}"]
    )
    command_id = utils.run_command_args(sess, "echo RunningOnNode=$DET_AGENT_ID", args)
    utils.wait_for_command_state(sess, command_id, "TERMINATED", MAX_WAIT_TIME)
    utils.assert_command_succeeded(sess, command_id)

    logs = api.task_logs(sess, command_id)
    str_logs = "".join(log.log for log in logs)
    assert f"RunningOnNode={expected_node}" in str_logs, str_logs


@pytest.mark.e2e_multi_k8s
def test_not_found_pool_multi_k8s() -> None:
    sess = api_utils.user_session()
    detproc.check_error(
        sess,
        ["det", "cmd", "run", "--config", "resources.resource_pool=missing", "echo"],
        "could not find resource pool missing",
    )


@pytest.mark.e2e_multi_k8s
def test_get_agents_multi_k8s() -> None:
    sess = api_utils.user_session()
    resp = bindings.get_GetAgents(sess)
    assert {agent.id for agent in resp.agents} == {"defaultrm", "additionalrm"}
