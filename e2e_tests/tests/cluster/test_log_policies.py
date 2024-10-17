import pytest

from determined.common.api import bindings
from determined.experimental import client
from tests import api_utils, detproc
from tests import experiment as exp
from tests.cluster import utils
from tests.experiment import noop


@pytest.mark.e2e_cpu
@pytest.mark.parametrize("should_match", [True, False])
def test_log_policy_cancel_retries(should_match: bool) -> None:
    sess = api_utils.user_session()
    regex = r"executing.*action.*exit.*code.*7"
    if not should_match:
        regex = r"(.*) this should not match (.*)"

    config = {
        "log_policies": [{"pattern": regex, "actions": [{"type": "cancel_retries"}]}],
        "max_restarts": 1,
    }
    exp_ref = noop.create_experiment(sess, [noop.Exit(7)], config=config)
    assert exp_ref.wait(interval=0.01) == client.ExperimentState.ERROR

    experiment_trials = exp.experiment_trials(sess, exp_ref.id)
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
    regex = r"executing.*action.*exit.*code.*7"
    if not should_match:
        regex = r"(.*) this should not match (.*)"

    agents = bindings.get_GetAgents(sess).agents
    assert len(agents) == 1
    assert agents[0].slots is not None

    config = {
        "log_policies": [{"pattern": regex, "actions": [{"type": "exclude_node"}]}],
        "resources": {"slots_per_trial": len(agents[0].slots)},
        "max_restarts": 1,
    }
    exp_ref = noop.create_experiment(sess, [noop.Exit(7)], config=config)
    exp.wait_for_experiment_state(sess, exp_ref.id, bindings.experimentv1State.RUNNING)

    if should_match:
        exp_ref_2 = noop.create_experiment(sess)
        assert exp_ref_2.wait(interval=0.01) == client.ExperimentState.COMPLETED

        exp.wait_for_experiment_state(sess, exp_ref.id, bindings.experimentv1State.QUEUED)

        experiment_trials = exp.experiment_trials(sess, exp_ref.id)
        assert len(experiment_trials) == 1
        assert experiment_trials[0].trial.restarts == 1
        trial_logs = "\n".join(exp.trial_logs(sess, experiment_trials[0].trial.id))
        assert "therefore will not schedule on" in trial_logs

        exp.kill_experiments(sess, [exp_ref.id], -1)
    else:
        assert exp_ref.wait(interval=0.01) == client.ExperimentState.ERROR

        experiment_trials = exp.experiment_trials(sess, exp_ref.id)
        assert len(experiment_trials) == 1
        assert experiment_trials[0].trial.restarts == 1
        trial_logs = "\n".join(exp.trial_logs(sess, experiment_trials[0].trial.id))
        assert "therefore will not schedule on" not in trial_logs


@pytest.mark.e2e_cpu
@pytest.mark.parametrize("should_match", [True, False])
def test_log_policy_exclude_node_single_agent(should_match: bool) -> None:
    sess = api_utils.user_session()
    regex = r"executing.*action.*exit.*code.*7"
    if not should_match:
        regex = r"(.*) this should not match (.*)"

    agents = bindings.get_GetAgents(sess).agents
    assert len(agents) == 1
    assert agents[0].slots is not None

    config = {
        "log_policies": [{"pattern": regex, "actions": [{"type": "exclude_node"}]}],
        "resources": {"slots_per_trial": len(agents[0].slots)},
        "max_restarts": 1,
    }
    exp_ref = noop.create_experiment(sess, [noop.Exit(7)], config=config)
    exp.wait_for_experiment_state(sess, exp_ref.id, bindings.experimentv1State.RUNNING)

    master_config = bindings.get_GetMasterConfig(api_utils.admin_session()).config
    if master_config.get("launch_error"):
        assert exp_ref.wait(interval=0.01) == client.ExperimentState.ERROR
    else:
        if should_match:
            exp.wait_for_experiment_state(sess, exp_ref.id, bindings.experimentv1State.QUEUED)
            exp.kill_experiments(sess, [exp_ref.id], -1)
        else:
            assert exp_ref.wait(interval=0.01) == client.ExperimentState.ERROR

    experiment_trials = exp.experiment_trials(sess, exp_ref.id)
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

    regex = r"executing.*action.*exit.*code.*7"
    if not should_match:
        regex = r"(.*) this should not match (.*)"

    config = {
        "log_policies": [{"pattern": regex, "actions": [{"type": "exclude_node"}]}],
        "max_restarts": 1,
    }
    exp_ref = noop.create_experiment(sess, [noop.Exit(7)], config=config)
    assert exp_ref.wait(interval=0.01) == client.ExperimentState.ERROR

    trials = exp.experiment_trials(sess, exp_ref.id)
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


@pytest.mark.e2e_cpu
@pytest.mark.parametrize("should_match", [True, False])
def test_log_signal(should_match: bool) -> None:
    sess = api_utils.user_session()
    regex = r"executing.*action.*exit.*code.*7"
    if not should_match:
        regex = r"(.*) this should not match (.*)"

    expected_signal = "Test Signal"
    config = {
        "log_policies": [{"pattern": regex, "signal": expected_signal}],
        "max_restarts": 1,
    }

    exp_ref = noop.create_experiment(sess, [noop.Exit(7)], config=config)
    assert exp_ref.wait(interval=0.01) == client.ExperimentState.ERROR

    searchRes = utils.get_run_by_exp_id(sess, exp_ref.id)
    runSignal = searchRes.runs[0].logSignal

    trialRes = bindings.get_GetTrial(sess, trialId=searchRes.runs[0].id)
    trialSignal = trialRes.trial.logSignal

    if should_match:
        assert runSignal == expected_signal
        assert trialSignal == expected_signal
    else:
        assert runSignal is None
        assert trialSignal is None


@pytest.mark.e2e_cpu
def test_signal_clear_after_exp_continue() -> None:
    sess = api_utils.user_session()
    regex = r"executing.*action.*exit.*code.*7"

    expected_signal = "Test Signal"
    config = {
        "log_policies": [{"pattern": regex, "signal": expected_signal}],
        "max_restarts": 0,
    }

    exp_ref = noop.create_experiment(sess, [noop.Exit(7)], config=config)
    assert exp_ref.wait(interval=0.01) == client.ExperimentState.ERROR

    searchRes = utils.get_run_by_exp_id(sess, exp_ref.id)
    runSignal = searchRes.runs[0].logSignal

    trialRes = bindings.get_GetTrial(sess, trialId=searchRes.runs[0].id)
    trialSignal = trialRes.trial.logSignal

    assert runSignal == expected_signal
    assert trialSignal == expected_signal

    detproc.check_call(
        sess,
        [
            "det",
            "e",
            "continue",
            str(exp_ref.id),
            "--config",
            "hyperparameters.crash_on_startup=false",
        ],
    )
    exp.wait_for_experiment_state(sess, exp_ref.id, bindings.experimentv1State.COMPLETED)

    searchRes = utils.get_run_by_exp_id(sess, exp_ref.id)
    runSignal = searchRes.runs[0].logSignal

    trialRes = bindings.get_GetTrial(sess, trialId=searchRes.runs[0].id)
    trialSignal = trialRes.trial.logSignal

    assert runSignal is None
    assert trialSignal is None
