from typing import Optional

import pytest

from determined.common import api
from determined.common.api import bindings
from tests import api_utils
from tests import config as conf
from tests import experiment as exp
from tests.cluster import utils

MAX_WAIT_TIME = 500  # Really long since minikube will need to pull images.


@pytest.mark.e2e_multi_k8s
@pytest.mark.parametrize(
    "resource_pool, expected_node", [(None, "defaultrm"), ("additional_pool", "additionalrm")]
)
def test_run_experiment_multi_k8s(resource_pool: Optional[str], expected_node: str) -> None:
    args = ["--config", "entrypoint=echo RunningOnNode=$DET_AGENT_ID"]
    if resource_pool:
        args += ["--config", f"resources.resource_pool={resource_pool}"]

    sess = api_utils.user_session()
    exp_id = exp.create_experiment(
        sess,
        conf.fixtures_path("no_op/single-one-short-step.yaml"),
        conf.fixtures_path("no_op"),
        args,
    )
    exp.wait_for_experiment_state(
        sess,
        exp_id,
        bindings.experimentv1State.COMPLETED,
        max_wait_secs=MAX_WAIT_TIME,
    )
    exp.assert_patterns_in_trial_logs(
        sess, exp.experiment_first_trial(sess, exp_id), [f"RunningOnNode={expected_node}"]
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
