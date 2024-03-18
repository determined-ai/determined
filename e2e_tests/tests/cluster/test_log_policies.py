import tempfile

import pytest
import yaml

from determined.common.api import bindings
from tests import api_utils
from tests import config as conf
from tests import experiment as exp


@pytest.mark.e2e_cpu
@pytest.mark.parametrize("should_match", [True, False])
def test_log_policy_cancel_retries(should_match: bool) -> None:
    sess = api_utils.user_session()
    regex = r"assert 0 <= self\.metrics_sigma"
    if not should_match:
        regex = r"(.*) this should not match (.*)"

    config = conf.load_config(conf.fixtures_path("no_op/single-medium-train-step.yaml"))
    config["log_policies"] = [
        {
            "pattern": regex,
            "action": {
                "type": "cancel_retries",
            },
        },
    ]
    config["hyperparameters"]["metrics_sigma"] = -1
    config["max_restarts"] = 1

    with tempfile.NamedTemporaryFile() as tf:
        with open(tf.name, "w") as f:
            yaml.dump(config, f)
        exp_id = exp.create_experiment(sess, tf.name, conf.fixtures_path("no_op"))

    exp.wait_for_experiment_state(sess, exp_id, bindings.experimentv1State.ERROR)

    experiment_trials = exp.experiment_trials(sess, exp_id)
    assert len(experiment_trials) == 1
    trial_logs = "\n".join(exp.trial_logs(sess, experiment_trials[0].trial.id))

    if should_match:
        assert experiment_trials[0].trial.restarts == 0
        assert "trial failed and matched logs to a don't retry policy" in trial_logs
    else:
        assert experiment_trials[0].trial.restarts == 1
        assert "trial failed and matched logs to a don't retry policy" not in trial_logs


@pytest.mark.e2e_k8s
@pytest.mark.parametrize("should_match", [True, False])
def test_log_policy_exclude_node_k8s(should_match: bool) -> None:
    sess = api_utils.user_session()
    regex = r"assert 0 <= self\.metrics_sigma"
    if not should_match:
        regex = r"\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b"

    config = conf.load_config(conf.fixtures_path("no_op/single-medium-train-step.yaml"))
    config["log_policies"] = [
        {
            "pattern": regex,
            "action": {
                "type": "exclude_node",
            },
        },
    ]
    config["hyperparameters"]["metrics_sigma"] = -1
    config["max_restarts"] = 1

    agents = bindings.get_GetAgents(sess).agents
    assert len(agents) == 1
    assert agents[0].slots is not None
    config["resources"] = {"slots_per_trial": len(agents[0].slots)}

    with tempfile.NamedTemporaryFile() as tf:
        with open(tf.name, "w") as f:
            yaml.dump(config, f)
        exp_id = exp.create_experiment(sess, tf.name, conf.fixtures_path("no_op"))

    exp.wait_for_experiment_state(sess, exp_id, bindings.experimentv1State.RUNNING)

    if should_match:
        second_exp_id = exp.create_experiment(
            sess,
            conf.fixtures_path("no_op/single-one-short-step.yaml"),
            conf.fixtures_path("no_op"),
        )
        exp.wait_for_experiment_state(sess, second_exp_id, bindings.experimentv1State.COMPLETED)

        exp.wait_for_experiment_state(sess, exp_id, bindings.experimentv1State.QUEUED)

        experiment_trials = exp.experiment_trials(sess, exp_id)
        assert len(experiment_trials) == 1
        assert experiment_trials[0].trial.restarts == 1
        trial_logs = "\n".join(exp.trial_logs(sess, experiment_trials[0].trial.id))
        assert "therefore will not schedule on" in trial_logs

        exp.kill_experiments(sess, [exp_id])
    else:
        exp.wait_for_experiment_state(sess, exp_id, bindings.experimentv1State.ERROR)

        experiment_trials = exp.experiment_trials(sess, exp_id)
        assert len(experiment_trials) == 1
        assert experiment_trials[0].trial.restarts == 1
        trial_logs = "\n".join(exp.trial_logs(sess, experiment_trials[0].trial.id))
        assert "therefore will not schedule on" not in trial_logs


@pytest.mark.e2e_cpu
@pytest.mark.parametrize("should_match", [True, False])
def test_log_policy_exclude_node_single_agent(should_match: bool) -> None:
    sess = api_utils.user_session()
    regex = r"assert 0 <= self\.metrics_sigma"
    if not should_match:
        regex = r"\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b"

    config = conf.load_config(conf.fixtures_path("no_op/single-medium-train-step.yaml"))
    config["log_policies"] = [
        {
            "pattern": regex,
            "action": {
                "type": "exclude_node",
            },
        },
    ]
    config["hyperparameters"]["metrics_sigma"] = -1
    config["max_restarts"] = 1

    agents = bindings.get_GetAgents(sess).agents
    assert len(agents) == 1
    assert agents[0].slots is not None
    config["resources"] = {"slots_per_trial": len(agents[0].slots)}

    with tempfile.NamedTemporaryFile() as tf:
        with open(tf.name, "w") as f:
            yaml.dump(config, f)
        exp_id = exp.create_experiment(sess, tf.name, conf.fixtures_path("no_op"))

    exp.wait_for_experiment_state(sess, exp_id, bindings.experimentv1State.RUNNING)

    master_config = bindings.get_GetMasterConfig(api_utils.admin_session()).config
    if master_config.get("launch_error"):
        exp.wait_for_experiment_state(sess, exp_id, bindings.experimentv1State.ERROR)
    else:
        if should_match:
            exp.wait_for_experiment_state(sess, exp_id, bindings.experimentv1State.QUEUED)
            exp.kill_experiments(sess, [exp_id])
        else:
            exp.wait_for_experiment_state(sess, exp_id, bindings.experimentv1State.ERROR)

    experiment_trials = exp.experiment_trials(sess, exp_id)
    assert len(experiment_trials) == 1
    assert experiment_trials[0].trial.restarts == 1
    trial_logs = "\n".join(exp.trial_logs(sess, experiment_trials[0].trial.id))

    if should_match:
        assert "therefore will not schedule on" in trial_logs
    else:
        assert "therefore will not schedule on" not in trial_logs


# Slurm behaviour is different than agent's currently. Slurm fails
# job if it can't be scheduled due to excluding while agents / k8s remain in queued.
@pytest.mark.e2e_slurm
@pytest.mark.parametrize("should_match", [True, False])
def test_log_policy_exclude_slurm(should_match: bool) -> None:
    sess = api_utils.user_session()
    agents = bindings.get_GetAgents(sess).agents
    if len(agents) != 1:
        pytest.skip("can only be run on a single agent cluster")

    regex = r"assert 0 <= self\.metrics_sigma"
    if not should_match:
        regex = r"\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b"

    config = conf.load_config(conf.fixtures_path("no_op/single-medium-train-step.yaml"))
    config["log_policies"] = [
        {
            "pattern": regex,
            "action": {
                "type": "exclude_node",
            },
        },
    ]
    config["hyperparameters"]["metrics_sigma"] = -1
    config["max_restarts"] = 1

    with tempfile.NamedTemporaryFile() as tf:
        with open(tf.name, "w") as f:
            yaml.dump(config, f)
        exp_id = exp.create_experiment(sess, tf.name, conf.fixtures_path("no_op"))
    exp.wait_for_experiment_state(sess, exp_id, bindings.experimentv1State.ERROR)

    trials = exp.experiment_trials(sess, exp_id)
    assert len(trials) == 1
    assert trials[0].trial.restarts == 1

    times_ran = "\n".join(exp.trial_logs(sess, trials[0].trial.id)).count(
        "Validating checkpoint storage"
    )
    if should_match:
        assert (
            times_ran == 1
        )  # Job fails to start up the second restart since all nodes are excluded.
    else:
        assert times_ran == 2
