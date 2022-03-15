import copy
import os
import shutil
import tempfile
import time
from typing import Union

import pytest

from determined.common import check, yaml
from determined.common.api import bindings
from tests import config as conf
from tests import experiment as exp


@pytest.mark.e2e_cpu
def test_noop_long_train_step() -> None:
    exp.run_basic_test(
        conf.fixtures_path("no_op/single-long-train-step.yaml"),
        conf.fixtures_path("no_op"),
        1,
    )


@pytest.mark.e2e_cpu
def test_noop_pause() -> None:
    """
    Walk through starting, pausing, and resuming a single no-op experiment.
    """
    experiment_id = exp.create_experiment(
        conf.fixtures_path("no_op/single-medium-train-step.yaml"),
        conf.fixtures_path("no_op"),
        None,
    )
    exp.wait_for_experiment_state(experiment_id, bindings.determinedexperimentv1State.STATE_ACTIVE)

    # Wait for the only trial to get scheduled.
    exp.wait_for_experiment_active_workload(experiment_id)

    # Wait for the only trial to show progress, indicating the image is built and running.
    exp.wait_for_experiment_workload_progress(experiment_id)

    # Pause the experiment. Note that Determined does not currently differentiate
    # between a "stopping paused" and a "paused" state, so we follow this check
    # up by ensuring the experiment cleared all scheduled workloads.
    exp.pause_experiment(experiment_id)
    exp.wait_for_experiment_state(experiment_id, bindings.determinedexperimentv1State.STATE_PAUSED)

    # Wait at most 20 seconds for the experiment to clear all workloads (each
    # train step should take 5 seconds).
    for _ in range(20):
        workload_active = exp.experiment_has_active_workload(experiment_id)
        if not workload_active:
            break
        else:
            time.sleep(1)
    check.true(
        not workload_active,
        "The experiment cannot be paused within 20 seconds.",
    )

    # Resume the experiment and wait for completion.
    exp.activate_experiment(experiment_id)
    exp.wait_for_experiment_state(
        experiment_id, bindings.determinedexperimentv1State.STATE_COMPLETED
    )


@pytest.mark.e2e_cpu
def test_noop_pause_of_experiment_without_trials() -> None:
    """
    Walk through starting, pausing, and resuming a single no-op experiment
    which will never schedule a trial.
    """
    config_obj = conf.load_config(conf.fixtures_path("no_op/single-one-short-step.yaml"))
    impossibly_large = 100
    config_obj["max_restarts"] = 0
    config_obj["resources"] = {"slots_per_trial": impossibly_large}
    with tempfile.NamedTemporaryFile() as tf:
        with open(tf.name, "w") as f:
            yaml.dump(config_obj, f)
        experiment_id = exp.create_experiment(tf.name, conf.fixtures_path("no_op"), None)
    exp.pause_experiment(experiment_id)
    exp.wait_for_experiment_state(experiment_id, bindings.determinedexperimentv1State.STATE_PAUSED)

    exp.activate_experiment(experiment_id)
    exp.wait_for_experiment_state(experiment_id, bindings.determinedexperimentv1State.STATE_ACTIVE)

    for _ in range(5):
        assert (
            exp.experiment_state(experiment_id) == bindings.determinedexperimentv1State.STATE_ACTIVE
        )
        time.sleep(1)

    exp.cancel_single(experiment_id)


@pytest.mark.e2e_cpu
def test_noop_single_warm_start() -> None:
    experiment_id1 = exp.run_basic_test(
        conf.fixtures_path("no_op/single.yaml"), conf.fixtures_path("no_op"), 1
    )

    trials = exp.experiment_trials(experiment_id1)
    assert len(trials) == 1

    first_trial = trials[0].trial
    first_trial_id = first_trial.id

    first_workloads = trials[0].workloads
    assert len(first_workloads or []) == 90
    checkpoints = exp.workloads_for_mode(first_workloads, "checkpoint")
    assert len(checkpoints or []) == 30
    assert checkpoints[0] and checkpoints[0].checkpoint
    first_checkpoint_uuid = checkpoints[0].checkpoint.uuid
    assert checkpoints[-1] and checkpoints[-1].checkpoint
    last_checkpoint_uuid = checkpoints[-1].checkpoint.uuid
    last_validation = exp.workloads_for_mode(first_workloads, "validation")[-1]
    assert last_validation and last_validation.validation and last_validation.validation.metrics
    assert last_validation.validation.metrics["validation_error"] == pytest.approx(0.9 ** 30)

    config_base = conf.load_config(conf.fixtures_path("no_op/single.yaml"))

    # Test source_trial_id.
    config_obj = copy.deepcopy(config_base)
    # Add a source trial ID to warm start from.
    config_obj["searcher"]["source_trial_id"] = first_trial_id

    experiment_id2 = exp.run_basic_test_with_temp_config(config_obj, conf.fixtures_path("no_op"), 1)

    trials = exp.experiment_trials(experiment_id2)
    assert len(trials) == 1

    second_trial = trials[0]
    assert len(second_trial.workloads or []) == 90

    # Second trial should have a warm start checkpoint id.
    assert second_trial.trial
    assert second_trial.trial.warmStartCheckpointUuid == last_checkpoint_uuid

    val_workloads = exp.workloads_for_mode(second_trial.workloads, "validation")
    assert (
        val_workloads[-1] and val_workloads[-1].validation and val_workloads[-1].validation.metrics
    )
    assert val_workloads[-1].validation.metrics["validation_error"] == pytest.approx(0.9 ** 60)

    # Now test source_checkpoint_uuid.
    config_obj = copy.deepcopy(config_base)
    # Add a source trial ID to warm start from.
    config_obj["searcher"]["source_checkpoint_uuid"] = checkpoints[0].checkpoint.uuid

    with tempfile.NamedTemporaryFile() as tf:
        with open(tf.name, "w") as f:
            yaml.dump(config_obj, f)

        experiment_id3 = exp.run_basic_test(tf.name, conf.fixtures_path("no_op"), 1)

    trials = exp.experiment_trials(experiment_id3)
    assert len(trials) == 1

    third_trial = trials[0]
    assert len(third_trial.workloads or []) == 90

    assert third_trial.trial
    assert third_trial.trial.warmStartCheckpointUuid == first_checkpoint_uuid
    validations = exp.workloads_for_mode(third_trial.workloads, "validation")
    assert validations[1] and validations[1].validation and validations[1].validation.metrics
    assert validations[1].validation.metrics["validation_error"] == pytest.approx(0.9 ** 3)


@pytest.mark.e2e_cpu
def test_cancel_one_experiment() -> None:
    experiment_id = exp.create_experiment(
        conf.fixtures_path("no_op/single-many-long-steps.yaml"),
        conf.fixtures_path("no_op"),
    )

    exp.cancel_single(experiment_id)


@pytest.mark.e2e_cpu
def test_cancel_one_active_experiment_unready() -> None:
    experiment_id = exp.create_experiment(
        conf.fixtures_path("no_op/single-many-long-steps.yaml"),
        conf.fixtures_path("no_op"),
    )

    for _ in range(15):
        if exp.experiment_has_active_workload(experiment_id):
            break
        time.sleep(1)
    else:
        raise AssertionError("no workload active after 15 seconds")

    exp.cancel_single(experiment_id, should_have_trial=True)


@pytest.mark.e2e_cpu
@pytest.mark.timeout(3 * 60)
def test_cancel_one_active_experiment_ready() -> None:
    experiment_id = exp.create_experiment(
        conf.tutorials_path("mnist_pytorch/const.yaml"),
        conf.tutorials_path("mnist_pytorch"),
    )

    while 1:
        if exp.experiment_has_completed_workload(experiment_id):
            break
        time.sleep(1)

    exp.cancel_single(experiment_id, should_have_trial=True)
    exp.assert_performed_final_checkpoint(experiment_id)


@pytest.mark.e2e_cpu
def test_cancel_one_paused_experiment() -> None:
    experiment_id = exp.create_experiment(
        conf.fixtures_path("no_op/single-many-long-steps.yaml"),
        conf.fixtures_path("no_op"),
        ["--paused"],
    )
    exp.cancel_single(experiment_id)


@pytest.mark.e2e_cpu
def test_cancel_ten_experiments() -> None:
    experiment_ids = [
        exp.create_experiment(
            conf.fixtures_path("no_op/single-many-long-steps.yaml"),
            conf.fixtures_path("no_op"),
        )
        for _ in range(10)
    ]

    for experiment_id in experiment_ids:
        exp.cancel_single(experiment_id)


@pytest.mark.e2e_cpu
def test_cancel_ten_paused_experiments() -> None:
    experiment_ids = [
        exp.create_experiment(
            conf.fixtures_path("no_op/single-many-long-steps.yaml"),
            conf.fixtures_path("no_op"),
            ["--paused"],
        )
        for _ in range(10)
    ]

    for experiment_id in experiment_ids:
        exp.cancel_single(experiment_id)


@pytest.mark.e2e_cpu
def test_startup_hook() -> None:
    exp.run_basic_test(
        conf.fixtures_path("no_op/startup-hook.yaml"),
        conf.fixtures_path("no_op"),
        1,
    )


@pytest.mark.e2e_cpu
def test_large_model_def_experiment() -> None:
    with tempfile.TemporaryDirectory() as td:
        shutil.copy(conf.fixtures_path("no_op/model_def.py"), td)
        # Write a 94MB file into the directory.  Use random data because it is not compressible.
        with open(os.path.join(td, "junk.txt"), "wb") as f:
            f.write(os.urandom(94 * 1024 * 1024))

        exp.run_basic_test(conf.fixtures_path("no_op/single-one-short-step.yaml"), td, 1)


def _test_rng_restore(fixture: str, metrics: list, tf2: Union[None, bool] = None) -> None:
    """
    This test confirms that an experiment can be restarted from a checkpoint
    with the same RNG state. It requires a test fixture that will emit
    random numbers from all of the RNGs used in the relevant framework as
    metrics. The experiment must have a const.yaml, run for at least 3 steps,
    checkpoint every step, and keep the first checkpoint (either by having
    metrics get worse over time, or by configuring the experiment to keep all
    checkpoints).
    """
    config_base = conf.load_config(conf.fixtures_path(fixture + "/const.yaml"))
    config = copy.deepcopy(config_base)
    if tf2 is not None:
        config = conf.set_tf2_image(config) if tf2 else conf.set_tf1_image(config)

    experiment = exp.run_basic_test_with_temp_config(
        config,
        conf.fixtures_path(fixture),
        1,
    )

    first_trial = exp.experiment_trials(experiment)[0]

    assert len(first_trial.workloads or []) >= 4

    first_checkpoint = exp.workloads_for_mode(first_trial.workloads, "checkpoint")[0]
    assert first_checkpoint and first_checkpoint.checkpoint
    first_checkpoint_uuid = first_checkpoint.checkpoint.uuid

    config = copy.deepcopy(config_base)
    if tf2 is not None:
        config = conf.set_tf2_image(config) if tf2 else conf.set_tf1_image(config)
    config["searcher"]["source_checkpoint_uuid"] = first_checkpoint.checkpoint.uuid

    experiment2 = exp.run_basic_test_with_temp_config(config, conf.fixtures_path(fixture), 1)

    second_trial = exp.experiment_trials(experiment2)[0]

    assert len(second_trial.workloads or []) >= 4
    assert second_trial.trial.warmStartCheckpointUuid == first_checkpoint_uuid
    first_trial_validations = exp.workloads_for_mode(first_trial.workloads, "validation")
    second_trial_validations = exp.workloads_for_mode(second_trial.workloads, "validation")

    for wl in range(0, 2):
        for metric in metrics:
            first_trial_val = first_trial_validations[wl + 1]
            assert (
                first_trial_val
                and first_trial_val.validation
                and first_trial_val.validation.metrics
            )
            first_metric = first_trial_val.validation.metrics[metric]
            second_trial_val = second_trial_validations[wl]
            assert (
                second_trial_val
                and second_trial_val.validation
                and second_trial_val.validation.metrics
            )
            second_metric = second_trial_val.validation.metrics[metric]
            assert (
                first_metric == second_metric
            ), f"failures on iteration: {wl} with metric: {metric}"


@pytest.mark.e2e_cpu
@pytest.mark.parametrize(
    "tf2",
    [
        pytest.param(True, marks=pytest.mark.tensorflow2_cpu),
        pytest.param(False, marks=pytest.mark.tensorflow1_cpu),
    ],
)
def test_keras_rng_restore(tf2: bool) -> None:
    _test_rng_restore("keras_no_op", ["val_rand_rand", "val_np_rand", "val_tf_rand"], tf2=tf2)


@pytest.mark.e2e_cpu
@pytest.mark.tensorflow1_cpu
@pytest.mark.tensorflow2_cpu
def test_estimator_rng_restore() -> None:
    _test_rng_restore("estimator_no_op", ["rand_rand", "np_rand"])


@pytest.mark.e2e_cpu
def test_pytorch_cpu_rng_restore() -> None:
    _test_rng_restore("pytorch_no_op", ["np_rand", "rand_rand", "torch_rand"])


@pytest.mark.e2e_gpu
def test_pytorch_gpu_rng_restore() -> None:
    _test_rng_restore("pytorch_no_op", ["np_rand", "rand_rand", "torch_rand", "gpu_rand"])


@pytest.mark.e2e_cpu
def test_noop_experiment_config_override() -> None:
    config_obj = conf.load_config(conf.fixtures_path("no_op/single-one-short-step.yaml"))
    with tempfile.NamedTemporaryFile() as tf:
        with open(tf.name, "w") as f:
            yaml.dump(config_obj, f)
        experiment_id = exp.create_experiment(
            tf.name,
            conf.fixtures_path("no_op"),
            ["--config", "reproducibility.experiment_seed=8200"],
        )
        exp_config = exp.experiment_config_json(experiment_id)
        assert exp_config["reproducibility"]["experiment_seed"] == 8200
        exp.cancel_single(experiment_id)
