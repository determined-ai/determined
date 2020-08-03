from typing import Dict

import pytest

from tests import config as conf
from tests import experiment as exp


@pytest.mark.parallel  # type: ignore
@pytest.mark.parametrize("tf2", [False])  # type: ignore
def test_tf_keras_native_parallel(tf2: bool) -> None:
    config = conf.load_config(conf.official_examples_path("trial/cifar10_cnn_tf_keras/const.yaml"))
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_native_parallel(config, True)
    config = conf.set_max_steps(config, 2)
    config = conf.set_tf2_image(config) if tf2 else conf.set_tf1_image(config)

    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.official_examples_path("trial/cifar10_cnn_tf_keras"), 1
    )
    trials = exp.experiment_trials(experiment_id)
    assert len(trials) == 1


@pytest.mark.parametrize(  # type: ignore
    "tf2",
    [
        pytest.param(True, marks=pytest.mark.tensorflow2_cpu),
        pytest.param(False, marks=pytest.mark.tensorflow1_cpu),
    ],
)
def test_tf_keras_const_warm_start(tf2: bool) -> None:
    config = conf.load_config(conf.official_examples_path("trial/cifar10_cnn_tf_keras/const.yaml"))
    config = conf.set_max_steps(config, 2)
    config = conf.set_tf2_image(config) if tf2 else conf.set_tf1_image(config)

    experiment_id1 = exp.run_basic_test_with_temp_config(
        config, conf.official_examples_path("trial/cifar10_cnn_tf_keras"), 1
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
        config, conf.official_examples_path("trial/cifar10_cnn_tf_keras"), 1
    )

    # The new  trials should have a warm start checkpoint ID.
    trials = exp.experiment_trials(experiment_id2)
    assert len(trials) == 1
    for trial in trials:
        assert trial["warm_start_checkpoint_id"] == first_checkpoint_id


@pytest.mark.parallel  # type: ignore
@pytest.mark.parametrize("aggregation_frequency", [1, 4])  # type: ignore
@pytest.mark.parametrize("tf2", [False, True])  # type: ignore
def test_tf_keras_parallel(aggregation_frequency: int, tf2: bool) -> None:
    config = conf.load_config(conf.official_examples_path("trial/cifar10_cnn_tf_keras/const.yaml"))
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_native_parallel(config, False)
    config = conf.set_max_steps(config, 2)
    config = conf.set_aggregation_frequency(config, aggregation_frequency)
    config = conf.set_tf2_image(config) if tf2 else conf.set_tf1_image(config)

    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.official_examples_path("trial/cifar10_cnn_tf_keras"), 1
    )
    trials = exp.experiment_trials(experiment_id)
    assert len(trials) == 1


@pytest.mark.e2e_gpu  # type: ignore
@pytest.mark.parametrize("tf2", [True, False])  # type: ignore
def test_tf_keras_single_gpu(tf2: bool) -> None:
    config = conf.load_config(conf.official_examples_path("trial/cifar10_cnn_tf_keras/const.yaml"))
    config = conf.set_slots_per_trial(config, 1)
    config = conf.set_max_steps(config, 2)
    config = conf.set_tf2_image(config) if tf2 else conf.set_tf1_image(config)

    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.official_examples_path("trial/cifar10_cnn_tf_keras"), 1
    )
    trials = exp.experiment_trials(experiment_id)
    assert len(trials) == 1


@pytest.mark.parallel  # type: ignore
def test_tf_keras_mnist_parallel() -> None:
    config = conf.load_config(
        conf.official_examples_path("trial/fashion_mnist_tf_keras/const.yaml")
    )
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_native_parallel(config, False)
    config = conf.set_max_steps(config, 2)

    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.official_examples_path("trial/fashion_mnist_tf_keras"), 1
    )
    trials = exp.experiment_trials(experiment_id)
    assert len(trials) == 1


@pytest.mark.parametrize(  # type: ignore
    "tf2", [pytest.param(False, marks=pytest.mark.tensorflow1_cpu)],
)
def test_tf_keras_mnist_data_layer_lfs(tf2: bool) -> None:
    run_tf_keras_mnist_data_layer_test(tf2, "lfs")


@pytest.mark.e2e_gpu  # type: ignore
@pytest.mark.parametrize("tf2", [False])  # type: ignore
@pytest.mark.parametrize("storage_type", ["s3"])  # type: ignore
def test_tf_keras_mnist_data_layer_s3(
    tf2: bool, storage_type: str, secrets: Dict[str, str]
) -> None:
    run_tf_keras_mnist_data_layer_test(tf2, storage_type)


def run_tf_keras_mnist_data_layer_test(tf2: bool, storage_type: str) -> None:
    config = conf.load_config(conf.experimental_path("trial/data_layer_mnist_tf_keras/const.yaml"))
    config = conf.set_max_steps(config, 2)
    config = conf.set_tf2_image(config) if tf2 else conf.set_tf1_image(config)
    if storage_type == "lfs":
        config = conf.set_shared_fs_data_layer(config)
    else:
        config = conf.set_s3_data_layer(config)

    exp.run_basic_test_with_temp_config(
        config, conf.experimental_path("trial/data_layer_mnist_tf_keras"), 1
    )


@pytest.mark.parallel  # type: ignore
@pytest.mark.parametrize("tf2", [False])  # type: ignore
@pytest.mark.parametrize("storage_type", ["lfs", "s3"])  # type: ignore
def test_tf_keras_mnist_data_layer_parallel(
    tf2: bool, storage_type: str, secrets: Dict[str, str]
) -> None:
    config = conf.load_config(conf.experimental_path("trial/data_layer_mnist_tf_keras/const.yaml"))
    config = conf.set_max_steps(config, 2)
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_tf2_image(config) if tf2 else conf.set_tf1_image(config)
    if storage_type == "lfs":
        config = conf.set_shared_fs_data_layer(config)
    else:
        config = conf.set_s3_data_layer(config)

    exp.run_basic_test_with_temp_config(
        config, conf.experimental_path("trial/data_layer_mnist_tf_keras"), 1
    )
