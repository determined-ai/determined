import pytest

from tests.integrations import config as conf
from tests.integrations import experiment as exp
from tests.integrations.cluster_utils import skip_test_if_not_enough_gpus


@skip_test_if_not_enough_gpus(8)
@pytest.mark.parallel  # type: ignore
@pytest.mark.parametrize("tf2", [False])  # type: ignore
def test_tf_keras_native_parallel(tf2: bool) -> None:
    config = conf.load_config(conf.official_examples_path("cifar10_cnn_tf_keras/const.yaml"))
    config["checkpoint_storage"] = exp.shared_fs_checkpoint_config()
    config.get("bind_mounts", []).append(exp.root_user_home_bind_mount())
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_native_parallel(config, True)
    config = conf.set_max_steps(config, 2)
    config = conf.set_tf2_image(config) if tf2 else conf.set_tf1_image(config)

    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.official_examples_path("cifar10_cnn_tf_keras"), 1
    )
    trials = exp.experiment_trials(experiment_id)
    assert len(trials) == 1


@skip_test_if_not_enough_gpus(8)
@pytest.mark.parallel  # type: ignore
@pytest.mark.parametrize("aggregation_frequency", [1, 4])  # type: ignore
@pytest.mark.parametrize("tf2", [False, True])  # type: ignore
def test_tf_keras_parallel(aggregation_frequency: int, tf2: bool) -> None:
    if tf2 and aggregation_frequency > 1:
        pytest.skip("TF2 does not currently support aggregation_frequency.")

    config = conf.load_config(conf.official_examples_path("cifar10_cnn_tf_keras/const.yaml"))
    config["checkpoint_storage"] = exp.shared_fs_checkpoint_config()
    config.get("bind_mounts", []).append(exp.root_user_home_bind_mount())
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_native_parallel(config, False)
    config = conf.set_max_steps(config, 2)
    config = conf.set_aggregation_frequency(config, aggregation_frequency)
    config = conf.set_tf2_image(config) if tf2 else conf.set_tf1_image(config)

    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.official_examples_path("cifar10_cnn_tf_keras"), 1
    )
    trials = exp.experiment_trials(experiment_id)
    assert len(trials) == 1


@pytest.mark.integ3  # type: ignore
@pytest.mark.parametrize("tf2", [True, False])  # type: ignore
def test_tf_keras_single_gpu(tf2: bool) -> None:
    config = conf.load_config(conf.official_examples_path("cifar10_cnn_tf_keras/const.yaml"))
    config["checkpoint_storage"] = exp.shared_fs_checkpoint_config()
    config.get("bind_mounts", []).append(exp.root_user_home_bind_mount())
    config = conf.set_slots_per_trial(config, 1)
    config = conf.set_max_steps(config, 2)
    config = conf.set_tf2_image(config) if tf2 else conf.set_tf1_image(config)

    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.official_examples_path("cifar10_cnn_tf_keras"), 1
    )
    trials = exp.experiment_trials(experiment_id)
    assert len(trials) == 1


@pytest.mark.integ3  # type: ignore
@pytest.mark.parametrize("tf2", [True, False])  # type: ignore
def test_tf_keras_const_warm_start(tf2: bool) -> None:
    config = conf.load_config(conf.official_examples_path("cifar10_cnn_tf_keras/const.yaml"))
    config["checkpoint_storage"] = exp.shared_fs_checkpoint_config()
    config.setdefault("bind_mounts", []).append(exp.root_user_home_bind_mount())
    config["searcher"]["max_steps"] = 3
    config = conf.set_tf2_image(config) if tf2 else conf.set_tf1_image(config)

    experiment_id1 = exp.run_basic_test_with_temp_config(
        config, conf.official_examples_path("cifar10_cnn_tf_keras"), 1
    )
    trials = exp.experiment_trials(experiment_id1)
    assert len(trials) == 1

    first_trial = trials[0]
    first_trial_id = first_trial["id"]

    assert len(first_trial["steps"]) == 3
    first_checkpoint_id = first_trial["steps"][2]["checkpoint"]["id"]

    # Add a source trial ID to warm start from.
    config["searcher"]["source_trial_id"] = first_trial_id

    experiment_id2 = exp.run_basic_test_with_temp_config(
        config, conf.official_examples_path("cifar10_cnn_tf_keras"), 1
    )

    # The new  trials should have a warm start checkpoint ID.
    trials = exp.experiment_trials(experiment_id2)
    assert len(trials) == 1
    for trial in trials:
        assert trial["warm_start_checkpoint_id"] == first_checkpoint_id


@pytest.mark.integ1  # type: ignore
def test_iris() -> None:
    config = conf.load_config(conf.official_examples_path("iris_tf_keras/const.yaml"))
    config = conf.set_max_steps(config, 2)

    exp.run_basic_test_with_temp_config(
        config, conf.official_examples_path("iris_tf_keras"), 1,
    )


@skip_test_if_not_enough_gpus(8)
@pytest.mark.parallel  # type: ignore
def test_tf_keras_mnist_parallel() -> None:
    config = conf.load_config(conf.official_examples_path("mnist_tf_keras/const.yaml"))
    config["checkpoint_storage"] = exp.shared_fs_checkpoint_config()
    config.get("bind_mounts", []).append(exp.root_user_home_bind_mount())
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_native_parallel(config, False)
    config = conf.set_max_steps(config, 2)

    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.official_examples_path("mnist_tf_keras"), 1
    )
    trials = exp.experiment_trials(experiment_id)
    assert len(trials) == 1
