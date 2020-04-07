import pytest

from tests.integrations import config as conf
from tests.integrations import experiment as exp
from tests.integrations.cluster_utils import skip_test_if_not_enough_gpus


@skip_test_if_not_enough_gpus(1)
@pytest.mark.integ1  # type: ignore
def test_tensorpack_const() -> None:
    config = conf.load_config(conf.official_examples_path("mnist_tp/const.yaml"))
    config["checkpoint_storage"] = exp.shared_fs_checkpoint_config()
    config.get("bind_mounts", []).append(exp.root_user_home_bind_mount())

    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.official_examples_path("mnist_tp"), 1
    )
    trials = exp.experiment_trials(experiment_id)
    assert len(trials) == 1


@skip_test_if_not_enough_gpus(8)
@pytest.mark.parallel  # type: ignore
def test_tensorpack_native_parallel() -> None:
    config = conf.load_config(conf.official_examples_path("mnist_tp/const.yaml"))
    config["checkpoint_storage"] = exp.shared_fs_checkpoint_config()
    config.get("bind_mounts", []).append(exp.root_user_home_bind_mount())
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_native_parallel(config, True)
    config = conf.set_max_steps(config, 2)

    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.official_examples_path("mnist_tp"), 1
    )
    trials = exp.experiment_trials(experiment_id)
    assert len(trials) == 1


@skip_test_if_not_enough_gpus(8)
@pytest.mark.parallel  # type: ignore
@pytest.mark.parametrize("aggregation_frequency", [1, 4])  # type: ignore
def test_tensorpack_parallel(aggregation_frequency: int) -> None:
    config = conf.load_config(conf.official_examples_path("mnist_tp/const.yaml"))
    config["checkpoint_storage"] = exp.shared_fs_checkpoint_config()
    config.get("bind_mounts", []).append(exp.root_user_home_bind_mount())
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_native_parallel(config, False)
    config = conf.set_max_steps(config, 2)
    config = conf.set_aggregation_frequency(config, aggregation_frequency)

    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.official_examples_path("mnist_tp"), 1
    )
    trials = exp.experiment_trials(experiment_id)
    assert len(trials) == 1
