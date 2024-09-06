import os
import tempfile
import time
from typing import Any, Dict, List

import pytest

from determined.common.api import bindings
from determined.experimental import client
from tests import api_utils
from tests import experiment as exp
from tests.experiment import noop


def do_test_pause_and_nan_validations(time_scale: int) -> None:
    """
    Walk through starting, pausing, and resuming a single no-op experiment.

    Simultaneously, ensure that NaN validation metrics don't explode an experiment, which used to
    be a separate test.
    """
    sess = api_utils.user_session()
    actions: List[noop.Action] = []
    for _ in range(20):
        actions.append(noop.Report({"loss": "nan"}, group="training"))
        actions.append(noop.Report({"loss": "nan"}, group="validation"))
        actions.append(noop.Sleep(1 * time_scale))
    exp_ref = noop.create_experiment(sess, actions)
    # Wait for the experiment to become RUNNING, which can take especially long for the slurm tests.
    exp.wait_for_experiment_state(sess, exp_ref.id, bindings.experimentv1State.RUNNING)
    # Wait for the experiment to actually show progress, so we don't risk pausing before starting.
    exp.wait_for_experiment_workload_progress(sess, exp_ref.id)

    # Pause the experiment. Note that Determined does not currently differentiate
    # between a "stopping paused" and a "paused" state, so we follow this check
    # up by ensuring the experiment cleared all scheduled workloads.
    exp_ref.pause()
    exp.wait_for_experiment_state(sess, exp_ref.id, bindings.experimentv1State.PAUSED)

    # Wait at most 20 * time_scale seconds for the experiment to clear all workloads.
    for _ in range(200):
        workload_active = exp.experiment_has_active_workload(sess, exp_ref.id)
        if not workload_active:
            break
        time.sleep(0.1 * time_scale)
    else:
        raise ValueError(f"The experiment cannot be paused within {20 * time_scale} seconds.")

    # Resume the experiment and ensure it is still running.
    trial = exp_ref.get_trials()[0]

    def count_metrics() -> int:
        return sum(1 for _ in trial.iter_metrics("training"))

    metrics_before = count_metrics()
    exp_ref.activate()
    for _ in range(200):
        if count_metrics() > metrics_before:
            break
        time.sleep(0.1 * time_scale)
    else:
        raise ValueError(f"The experiment did not reactivate within {20 * time_scale} seconds.")

    exp_ref.kill()


@pytest.mark.e2e_cpu
def test_pause_and_nan_validations() -> None:
    do_test_pause_and_nan_validations(time_scale=1)


@pytest.mark.e2e_slurm
@pytest.mark.e2e_pbs
@pytest.mark.timeout(20 * 60)
def test_pause_and_nan_validations_hpc() -> None:
    """
    Just like the e2e_cpu verison, but much slower.
    """
    do_test_pause_and_nan_validations(time_scale=10)


@pytest.mark.e2e_cpu
def test_generic_checkpoint_associated_with_trial() -> None:
    sess = api_utils.user_session()
    exp_ref = noop.create_experiment(sess, [noop.Checkpoint()])
    assert exp_ref.wait(interval=0.01) == client.ExperimentState.COMPLETED
    trials = exp.experiment_trials(sess, exp_ref.id)
    checkpoint = (
        client.Determined._from_session(sess).get_trial(trials[0].trial.id).top_checkpoint()
    )
    assert checkpoint.task_id == trials[0].trial.taskId


@pytest.mark.e2e_cpu
def test_pause_of_experiment_without_trials() -> None:
    """
    Walk through starting, pausing, and resuming a single no-op experiment
    which will never schedule a trial.
    """
    sess = api_utils.user_session()
    exp_ref = noop.create_experiment(sess, config={"resources": {"slots_per_trial": 1000}})
    exp_ref.pause()
    exp.wait_for_experiment_state(sess, exp_ref.id, bindings.experimentv1State.PAUSED)

    exp_ref.activate()
    exp.wait_for_experiment_state(sess, exp_ref.id, bindings.experimentv1State.QUEUED)

    exp_ref.kill()


@pytest.mark.e2e_cpu
def test_pause_with_multiexperiment() -> None:
    """
    Start, pause, and resume a single no-op experiment
    using the bulk action endpoints and ExperimentIds param.
    """
    sess = api_utils.user_session()
    exp_ref = noop.create_experiment(sess, config={"resources": {"slots_per_trial": 1000}})
    exp.pause_experiments(sess, [exp_ref.id], -1)
    exp.wait_for_experiment_state(sess, exp_ref.id, bindings.experimentv1State.PAUSED)

    exp.activate_experiments(sess, [exp_ref.id], -1)
    exp.wait_for_experiment_state(sess, exp_ref.id, bindings.experimentv1State.QUEUED)
    exp.kill_experiments(sess, [exp_ref.id], -1)
    exp_ref.reload()
    assert exp_ref.wait(interval=0.01) == client.ExperimentState.CANCELED


@pytest.mark.e2e_cpu
def test_pause_with_multiexperiment_filter() -> None:
    """
    Pause a single no-op experiment
    using the bulk action endpoint and Filters param.
    """
    sess = api_utils.user_session()
    name = api_utils.get_random_string()
    config = {"name": name, "resources": {"slots_per_trial": 1000}}
    exp_ref = noop.create_experiment(sess, config=config)
    exp.pause_experiments(sess, [], -1, name=name)
    exp.wait_for_experiment_state(sess, exp_ref.id, bindings.experimentv1State.PAUSED)
    # test state=nonTerminalExperimentStates() filter in cancel/kill
    exp.kill_experiments(sess, [], -1, name=name)
    exp.wait_for_experiment_state(sess, exp_ref.id, bindings.experimentv1State.CANCELED)
    # test state=terminalExperimentStates() filter in archive
    exp.archive_experiments(sess, [], -1, name=name)
    exp_ref.reload()
    assert exp_ref.archived, exp_ref.archived


@pytest.mark.e2e_cpu
def test_warm_start() -> None:
    sess = api_utils.user_session()
    exp_1 = noop.create_experiment(
        sess,
        [
            # Two checkpoints to ensure source_trial_id takes the second one.
            noop.Checkpoint(),
            noop.Report({"x": 1}),
            noop.Checkpoint(),
        ],
    )
    assert exp_1.wait(interval=0.01) == client.ExperimentState.COMPLETED

    trial_1 = exp.experiment_trials(sess, exp_1.id)[0]
    ckpt_1 = exp.workloads_with_checkpoint(trial_1.workloads)[1].uuid

    def wait_for_trial(exp_id: int) -> exp.TrialPlusWorkload:
        # Wait for a trial to appear.
        deadline = time.time() + 20
        while time.time() < deadline:
            trials = exp.experiment_trials(sess, exp_id)
            if trials:
                return trials[0]
        raise ValueError("no trial created before deadline")

    # Test source_trial_id.
    config: Dict[str, Any] = {"searcher": {"source_trial_id": trial_1.trial.id}}
    exp_2 = noop.create_experiment(sess, config=config)
    trial_2 = wait_for_trial(exp_2.id)
    exp_2.kill()

    # Second trial should have a warm start checkpoint id.
    assert trial_2.trial.warmStartCheckpointUuid == ckpt_1

    # Now test source_checkpoint_uuid.
    config = {"searcher": {"source_checkpoint_uuid": ckpt_1}}
    exp_3 = noop.create_experiment(sess, config=config)
    trial_3 = wait_for_trial(exp_3.id)
    exp_3.kill()

    assert trial_3.trial.warmStartCheckpointUuid == ckpt_1


@pytest.mark.e2e_cpu
def test_cancel_experiment() -> None:
    sess = api_utils.user_session()
    exp_ref = noop.create_experiment(sess)
    exp_ref.cancel()
    assert exp_ref.wait(interval=0.01) == client.ExperimentState.CANCELED


@pytest.mark.e2e_cpu
def test_cancel_active_experiment_unready() -> None:
    sess = api_utils.user_session()
    actions: List[noop.Action] = []
    for _ in range(20):
        actions.append(noop.Report({"loss": "1"}, group="training"))
        actions.append(noop.Report({"loss": "1"}, group="validation"))
        actions.append(noop.Sleep(1))
    exp_ref = noop.create_experiment(sess, actions)

    for _ in range(150):
        if exp.experiment_has_active_workload(sess, exp_ref.id):
            break
        time.sleep(0.1)
    else:
        raise AssertionError("no workload active after 15 seconds")

    exp_ref.cancel()
    assert exp_ref.wait(interval=0.01) == client.ExperimentState.CANCELED


@pytest.mark.e2e_cpu
@pytest.mark.timeout(3 * 60)
def test_cancel_active_experiment_ready() -> None:
    sess = api_utils.user_session()
    actions: List[noop.Action] = []
    for _ in range(20):
        actions.append(noop.Report({"loss": "1"}, group="training"))
        actions.append(noop.Report({"loss": "1"}, group="validation"))
        actions.append(noop.Sleep(1))
    exp_ref = noop.create_experiment(sess, actions)
    exp.wait_for_experiment_workload_progress(sess, exp_ref.id)
    exp_ref.cancel()
    assert exp_ref.wait(interval=0.01) == client.ExperimentState.CANCELED


@pytest.mark.e2e_cpu
def test_cancel_paused_experiment() -> None:
    sess = api_utils.user_session()
    exp_ref = noop.create_paused_experiment(sess)
    exp_ref.cancel()
    assert exp_ref.wait(interval=0.01) == client.ExperimentState.CANCELED


@pytest.mark.e2e_cpu
def test_large_model_def_experiment() -> None:
    sess = api_utils.user_session()
    with tempfile.TemporaryDirectory() as td:
        # Write a 94MB file into the directory.  Use random data because it is not compressible.
        junk_path = os.path.join(td, "junk.txt")
        with open(junk_path, "wb") as f:
            f.write(os.urandom(94 * 1024 * 1024))

        # Actually run the test to make sure that not only does the master accept the model, but the
        # resource manager can start the experiment.
        exp_ref = noop.create_experiment(sess, includes=[junk_path])
        assert exp_ref.wait(interval=0.01) == client.ExperimentState.COMPLETED


@pytest.mark.e2e_cpu
def test_experiment_config_override() -> None:
    sess = api_utils.user_session()
    with tempfile.NamedTemporaryFile() as tf:
        with open(tf.name, "w") as f:
            f.write(
                """
                name: test_experiment_config_override
                searcher:
                    name: single
                    metric: x
                entrypoint: echo yo dawg
            """
            )
        experiment_id = exp.create_experiment(
            sess,
            tf.name,
            None,
            ["--config=name=xyz", "--paused"],
        )
        exp_config = exp.experiment_config_json(sess, experiment_id)
        assert exp_config["name"] == "xyz"
        exp.kill_single(sess, experiment_id)
