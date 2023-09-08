import multiprocessing
from typing import Callable

import pytest

from determined import keras
from determined.experimental import client
from tests import config as conf
from tests import experiment as exp


def _export_and_load_model(experiment_id: int, master_url: str) -> None:
    # Normally verifying that we can load a model would be a good unit test, but making this an e2e
    # test ensures that our model saving and loading works with all the versions of tf that we test.
    ckpt = client.Determined(master_url).get_experiment(experiment_id).top_checkpoint()
    _ = keras.load_model_from_checkpoint_path(ckpt.download())


def export_and_load_model(experiment_id: int) -> None:
    # We run this in a subprocess to avoid module name collisions
    # when performing checkpoint export of different models.
    ctx = multiprocessing.get_context("spawn")
    p = ctx.Process(
        target=_export_and_load_model,
        args=(
            experiment_id,
            conf.make_master_url(),
        ),
    )
    p.start()
    p.join()
    assert p.exitcode == 0, p.exitcode


@pytest.mark.parallel
@pytest.mark.parametrize("aggregation_frequency", [1, 4])
def test_tf_keras_parallel(
    aggregation_frequency: int, collect_trial_profiles: Callable[[int], None]
) -> None:
    config = conf.load_config(conf.cv_examples_path("cifar10_tf_keras/const.yaml"))
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_aggregation_frequency(config, aggregation_frequency)
    config = conf.set_tf2_image(config)
    config = conf.set_profiling_enabled(config)

    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.cv_examples_path("cifar10_tf_keras"), 1
    )
    trials = exp.experiment_trials(experiment_id)
    assert len(trials) == 1

    # Test exporting a checkpoint.
    export_and_load_model(experiment_id)
    collect_trial_profiles(trials[0].trial.id)

    # Check on record/batch counts we emitted in logs.
    validation_size = 10000
    global_batch_size = config["hyperparameters"]["global_batch_size"]
    num_workers = config.get("resources", {}).get("slots_per_trial", 1)
    global_batch_size = config["hyperparameters"]["global_batch_size"]
    scheduling_unit = config.get("scheduling_unit", 100)
    per_slot_batch_size = global_batch_size // num_workers
    exp_val_batches = (validation_size + (per_slot_batch_size - 1)) // per_slot_batch_size
    patterns = [
        # Expect two copies of matching training reports.
        f"trained: {scheduling_unit * global_batch_size} records.*in {scheduling_unit} batches",
        f"trained: {scheduling_unit * global_batch_size} records.*in {scheduling_unit} batches",
        f"validated: {validation_size} records.*in {exp_val_batches} batches",
    ]
    exp.assert_patterns_in_trial_logs(trials[0].trial.id, patterns)


@pytest.mark.parallel
@pytest.mark.tensorflow2
def run_tf_keras_dcgan_example(collect_trial_profiles: Callable[[int], None]) -> None:
    config = conf.load_config(conf.gan_examples_path("dcgan_tf_keras/const.yaml"))
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_min_validation_period(config, {"batches": 200})
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_tf2_image(config)
    config = conf.set_profiling_enabled(config)

    exp_id = exp.run_basic_test_with_temp_config(
        config, conf.gan_examples_path("dcgan_tf_keras"), 1
    )
    trial_id = exp.experiment_trials(exp_id)[0].trial.id
    collect_trial_profiles(trial_id)
