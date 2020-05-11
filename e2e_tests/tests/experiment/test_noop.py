import copy
import tempfile
import time

import pytest
from ruamel import yaml

from determined_common import check
from tests import config as conf
from tests import experiment as exp


@pytest.mark.e2e_cpu  # type: ignore
def test_noop_long_train_step() -> None:
    exp.run_basic_test(
        conf.fixtures_path("no_op/single-long-train-step.yaml"), conf.fixtures_path("no_op"), 1,
    )


@pytest.mark.e2e_cpu  # type: ignore
def test_noop_pause() -> None:
    """
    Walk through starting, pausing, and resuming a single no-op experiment.
    """
    experiment_id = exp.create_experiment(
        conf.fixtures_path("no_op/single-medium-train-step.yaml"),
        conf.fixtures_path("no_op"),
        None,
    )
    exp.wait_for_experiment_state(experiment_id, "ACTIVE")

    # Wait for the only trial to get scheduled.
    workload_active = False
    for _ in range(conf.MAX_TASK_SCHEDULED_SECS):
        workload_active = exp.experiment_has_active_workload(experiment_id)
        if workload_active:
            break
        else:
            time.sleep(1)
    check.true(
        workload_active,
        f"The only trial cannot be scheduled within {conf.MAX_TASK_SCHEDULED_SECS} seconds.",
    )

    # Wait for the only trial to show progress, indicating the image is built and running.
    num_steps = 0
    for _ in range(conf.MAX_TRIAL_BUILD_SECS):
        trials = exp.experiment_trials(experiment_id)
        if len(trials) > 0:
            only_trial = trials[0]
            num_steps = len(only_trial["steps"])
            if num_steps > 1:
                break
        time.sleep(1)
    check.true(
        num_steps > 1,
        f"The only trial cannot start training within {conf.MAX_TRIAL_BUILD_SECS} seconds.",
    )

    # Pause the experiment. Note that Determined does not currently differentiate
    # between a "stopping paused" and a "paused" state, so we follow this check
    # up by ensuring the experiment cleared all scheduled workloads.
    exp.pause_experiment(experiment_id)
    exp.wait_for_experiment_state(experiment_id, "PAUSED")

    # Wait at most 20 seconds for the experiment to clear all workloads (each
    # train step should take 5 seconds).
    for _ in range(20):
        workload_active = exp.experiment_has_active_workload(experiment_id)
        if not workload_active:
            break
        else:
            time.sleep(1)
    check.true(
        not workload_active, "The experiment cannot be paused within 20 seconds.",
    )

    # Resume the experiment and wait for completion.
    exp.activate_experiment(experiment_id)
    exp.wait_for_experiment_state(experiment_id, "COMPLETED")


@pytest.mark.e2e_cpu  # type: ignore
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
    exp.wait_for_experiment_state(experiment_id, "PAUSED")

    exp.activate_experiment(experiment_id)
    exp.wait_for_experiment_state(experiment_id, "ACTIVE")

    for _ in range(5):
        assert exp.experiment_state(experiment_id) == "ACTIVE"
        time.sleep(1)

    exp.cancel_single(experiment_id)


@pytest.mark.e2e_cpu  # type: ignore
def test_noop_single_warm_start() -> None:
    experiment_id1 = exp.run_basic_test(
        conf.fixtures_path("no_op/single.yaml"), conf.fixtures_path("no_op"), 1
    )

    trials = exp.experiment_trials(experiment_id1)
    assert len(trials) == 1

    first_trial = trials[0]
    first_trial_id = first_trial["id"]

    assert len(first_trial["steps"]) == 30
    first_step = first_trial["steps"][0]
    first_checkpoint_id = first_step["checkpoint"]["id"]
    last_step = first_trial["steps"][29]
    last_checkpoint_id = last_step["checkpoint"]["id"]
    assert last_step["validation"]["metrics"]["validation_metrics"][
        "validation_error"
    ] == pytest.approx(0.9 ** 30)

    config_base = conf.load_config(conf.fixtures_path("no_op/single.yaml"))

    # Test source_trial_id.
    config_obj = copy.deepcopy(config_base)
    # Add a source trial ID to warm start from.
    config_obj["searcher"]["source_trial_id"] = first_trial_id

    experiment_id2 = exp.run_basic_test_with_temp_config(config_obj, conf.fixtures_path("no_op"), 1)

    trials = exp.experiment_trials(experiment_id2)
    assert len(trials) == 1

    second_trial = trials[0]
    assert len(second_trial["steps"]) == 30

    # Second trial should have a warm start checkpoint id.
    assert second_trial["warm_start_checkpoint_id"] == last_checkpoint_id

    assert second_trial["steps"][29]["validation"]["metrics"]["validation_metrics"][
        "validation_error"
    ] == pytest.approx(0.9 ** 60)

    # Now test source_checkpoint_uuid.
    config_obj = copy.deepcopy(config_base)
    # Add a source trial ID to warm start from.
    config_obj["searcher"]["source_checkpoint_uuid"] = first_step["checkpoint"]["uuid"]

    with tempfile.NamedTemporaryFile() as tf:
        with open(tf.name, "w") as f:
            yaml.dump(config_obj, f)

        experiment_id3 = exp.run_basic_test(tf.name, conf.fixtures_path("no_op"), 1)

    trials = exp.experiment_trials(experiment_id3)
    assert len(trials) == 1

    third_trial = trials[0]
    assert len(third_trial["steps"]) == 30

    assert third_trial["warm_start_checkpoint_id"] == first_checkpoint_id

    assert third_trial["steps"][1]["validation"]["metrics"]["validation_metrics"][
        "validation_error"
    ] == pytest.approx(0.9 ** 3)


@pytest.mark.e2e_cpu  # type: ignore
def test_cancel_one_experiment() -> None:
    experiment_id = exp.create_experiment(
        conf.fixtures_path("no_op/single-many-long-steps.yaml"), conf.fixtures_path("no_op"),
    )

    exp.cancel_single(experiment_id)


@pytest.mark.e2e_cpu  # type: ignore
def test_cancel_one_active_experiment() -> None:
    experiment_id = exp.create_experiment(
        conf.fixtures_path("no_op/single-many-long-steps.yaml"), conf.fixtures_path("no_op"),
    )

    for _ in range(15):
        if exp.experiment_has_active_workload(experiment_id):
            break
        time.sleep(1)
    else:
        raise AssertionError("no workload active after 15 seconds")

    exp.cancel_single(experiment_id, should_have_trial=True)


@pytest.mark.e2e_cpu  # type: ignore
def test_cancel_one_paused_experiment() -> None:
    experiment_id = exp.create_experiment(
        conf.fixtures_path("no_op/single-many-long-steps.yaml"),
        conf.fixtures_path("no_op"),
        ["--paused"],
    )

    exp.cancel_single(experiment_id)


@pytest.mark.e2e_cpu  # type: ignore
def test_cancel_ten_experiments() -> None:
    experiment_ids = [
        exp.create_experiment(
            conf.fixtures_path("no_op/single-many-long-steps.yaml"), conf.fixtures_path("no_op"),
        )
        for _ in range(10)
    ]

    for experiment_id in experiment_ids:
        exp.cancel_single(experiment_id)


@pytest.mark.e2e_cpu  # type: ignore
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


@pytest.mark.e2e_cpu  # type: ignore
def test_startup_hook() -> None:
    exp.run_basic_test(
        conf.fixtures_path("no_op/startup-hook.yaml"), conf.fixtures_path("no_op"), 1,
    )
