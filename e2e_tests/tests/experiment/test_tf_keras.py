from typing import Callable, Dict

import pytest

from tests import config as conf
from tests import experiment as exp


@pytest.mark.tensorflow2  # type: ignore
@pytest.mark.parametrize(  # type: ignore
    "tf2",
    [
        pytest.param(True, marks=pytest.mark.tensorflow2_cpu),
        pytest.param(False, marks=pytest.mark.tensorflow1_cpu),
    ],
)
def test_tf_keras_const_warm_start(
    tf2: bool, collect_trial_profiles: Callable[[int], None]
) -> None:
    config = conf.load_config(conf.cv_examples_path("cifar10_tf_keras/const.yaml"))
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_min_validation_period(config, {"batches": 1000})
    config = conf.set_tf2_image(config) if tf2 else conf.set_tf1_image(config)
    config = conf.set_profiling_enabled(config)

    experiment_id1 = exp.run_basic_test_with_temp_config(
        config, conf.cv_examples_path("cifar10_tf_keras"), 1
    )
    trials = exp.experiment_trials(experiment_id1)
    assert len(trials) == 1

    first_trial = trials[0]
    first_trial_id = first_trial["id"]

    assert len(first_trial["steps"]) == 2
    first_checkpoint_id = first_trial["steps"][1]["checkpoint"]["id"]

    # Add a source trial ID to warm start from.
    config["searcher"]["source_trial_id"] = first_trial_id

    experiment_id2 = exp.run_basic_test_with_temp_config(
        config, conf.cv_examples_path("cifar10_tf_keras"), 1
    )

    # The new  trials should have a warm start checkpoint ID.
    trials = exp.experiment_trials(experiment_id2)
    assert len(trials) == 1
    for trial in trials:
        assert trial["warm_start_checkpoint_id"] == first_checkpoint_id
    trial_id = trials[0]["id"]
    collect_trial_profiles(trial_id)


@pytest.mark.parallel  # type: ignore
@pytest.mark.tensorflow2  # type: ignore
@pytest.mark.parametrize("aggregation_frequency", [1, 4])  # type: ignore
@pytest.mark.parametrize("tf2", [False, True])  # type: ignore
def test_tf_keras_parallel(
    aggregation_frequency: int, tf2: bool, collect_trial_profiles: Callable[[int], None]
) -> None:
    config = conf.load_config(conf.cv_examples_path("cifar10_tf_keras/const.yaml"))
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_aggregation_frequency(config, aggregation_frequency)
    config = conf.set_tf2_image(config) if tf2 else conf.set_tf1_image(config)
    config = conf.set_profiling_enabled(config)

    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.cv_examples_path("cifar10_tf_keras"), 1
    )
    trials = exp.experiment_trials(experiment_id)
    assert len(trials) == 1

    # Test exporting a checkpoint.
    exp.export_and_load_model(experiment_id)
    collect_trial_profiles(trials[0]["id"])


@pytest.mark.e2e_gpu  # type: ignore
@pytest.mark.tensorflow2  # type: ignore
@pytest.mark.parametrize("tf2", [True, False])  # type: ignore
def test_tf_keras_single_gpu(tf2: bool, collect_trial_profiles: Callable[[int], None]) -> None:
    config = conf.load_config(conf.cv_examples_path("cifar10_tf_keras/const.yaml"))
    config = conf.set_slots_per_trial(config, 1)
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_tf2_image(config) if tf2 else conf.set_tf1_image(config)
    config = conf.set_profiling_enabled(config)

    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.cv_examples_path("cifar10_tf_keras"), 1
    )
    trials = exp.experiment_trials(experiment_id)
    assert len(trials) == 1

    # Test exporting a checkpoint.
    exp.export_and_load_model(experiment_id)
    collect_trial_profiles(trials[0]["id"])


@pytest.mark.parallel  # type: ignore
@pytest.mark.tensorflow2  # type: ignore
def test_tf_keras_mnist_parallel(collect_trial_profiles: Callable[[int], None]) -> None:
    config = conf.load_config(conf.tutorials_path("fashion_mnist_tf_keras/const.yaml"))
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_profiling_enabled(config)

    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.tutorials_path("fashion_mnist_tf_keras"), 1
    )
    trials = exp.experiment_trials(experiment_id)
    assert len(trials) == 1
    collect_trial_profiles(trials[0]["id"])


@pytest.mark.tensorflow2_cpu  # type: ignore
@pytest.mark.tensorflow2  # type: ignore
def test_tf_keras_tf2_disabled(collect_trial_profiles: Callable[[int], None]) -> None:
    """Keras on tf2 with tf2 and eager execution disabled."""
    config = conf.load_config(conf.fixtures_path("keras_tf2_disabled_no_op/const.yaml"))
    config = conf.set_max_length(config, {"batches": 1})
    config = conf.set_tf2_image(config)
    config = conf.set_profiling_enabled(config)

    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.fixtures_path("keras_tf2_disabled_no_op"), 1
    )
    trials = exp.experiment_trials(experiment_id)
    assert len(trials) == 1
    exp.export_and_load_model(experiment_id)
    collect_trial_profiles(trials[0]["id"])


@pytest.mark.tensorflow2  # type: ignore
@pytest.mark.parametrize(  # type: ignore
    "tf2",
    [pytest.param(False, marks=pytest.mark.tensorflow1_cpu)],
)
def test_tf_keras_mnist_data_layer_lfs(
    tf2: bool, collect_trial_profiles: Callable[[int], None]
) -> None:
    exp_id = run_tf_keras_mnist_data_layer_test(tf2, "lfs")
    trial_id = exp.experiment_trials(exp_id)[0]["id"]
    collect_trial_profiles(trial_id)


@pytest.mark.e2e_gpu  # type: ignore
@pytest.mark.tensorflow2  # type: ignore
@pytest.mark.parametrize("tf2", [False])  # type: ignore
@pytest.mark.parametrize("storage_type", ["s3"])  # type: ignore
def test_tf_keras_mnist_data_layer_s3(
    tf2: bool,
    storage_type: str,
    secrets: Dict[str, str],
    collect_trial_profiles: Callable[[int], None],
) -> None:
    exp_id = run_tf_keras_mnist_data_layer_test(tf2, storage_type)
    trial_id = exp.experiment_trials(exp_id)[0]["id"]
    collect_trial_profiles(trial_id)


def run_tf_keras_mnist_data_layer_test(tf2: bool, storage_type: str) -> int:
    config = conf.load_config(conf.features_examples_path("data_layer_mnist_tf_keras/const.yaml"))
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_min_validation_period(config, {"batches": 1000})
    config = conf.set_tf2_image(config) if tf2 else conf.set_tf1_image(config)
    config = conf.set_profiling_enabled(config)

    if storage_type == "lfs":
        config = conf.set_shared_fs_data_layer(config)
    else:
        config = conf.set_s3_data_layer(config)

    return exp.run_basic_test_with_temp_config(
        config, conf.features_examples_path("data_layer_mnist_tf_keras"), 1
    )


@pytest.mark.parallel  # type: ignore
@pytest.mark.tensorflow2  # type: ignore
@pytest.mark.parametrize("tf2", [False])  # type: ignore
@pytest.mark.parametrize("storage_type", ["lfs", "s3"])  # type: ignore
def test_tf_keras_mnist_data_layer_parallel(
    tf2: bool,
    storage_type: str,
    secrets: Dict[str, str],
    collect_trial_profiles: Callable[[int], None],
) -> None:
    config = conf.load_config(conf.features_examples_path("data_layer_mnist_tf_keras/const.yaml"))
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_tf2_image(config) if tf2 else conf.set_tf1_image(config)
    config = conf.set_profiling_enabled(config)

    if storage_type == "lfs":
        config = conf.set_shared_fs_data_layer(config)
    else:
        config = conf.set_s3_data_layer(config)

    exp_id = exp.run_basic_test_with_temp_config(
        config, conf.features_examples_path("data_layer_mnist_tf_keras"), 1
    )

    trial_id = exp.experiment_trials(exp_id)[0]["id"]
    collect_trial_profiles(trial_id)


@pytest.mark.parallel  # type: ignore
@pytest.mark.tensorflow2  # type: ignore
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
    trial_id = exp.experiment_trials(exp_id)[0]["id"]
    collect_trial_profiles(trial_id)
