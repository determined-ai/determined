import pytest
import torch

from determined.experimental import TrialReference
from tests.integrations import config as conf
from tests.integrations import experiment as exp
from tests.integrations.cluster_utils import skip_test_if_not_enough_gpus


@pytest.mark.integ2  # type: ignore
@pytest.mark.parametrize("aggregation_frequency", [1, 4])  # type: ignore
def test_pytorch_11_const(aggregation_frequency: int) -> None:
    config = conf.load_config(conf.fixtures_path("mnist_pytorch/const-pytorch11.yaml"))
    config = conf.set_aggregation_frequency(config, aggregation_frequency)

    exp.run_basic_test_with_temp_config(config, conf.official_examples_path("mnist_pytorch"), 1)


@pytest.mark.integ2  # type: ignore
def test_pytorch_load() -> None:
    config = conf.load_config(conf.fixtures_path("mnist_pytorch/const-pytorch11.yaml"))

    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.official_examples_path("mnist_pytorch"), 1
    )
    trials = exp.experiment_trials(experiment_id)
    nn = TrialReference(trials[0].id).select_checkpoint(latest=True).load()
    assert isinstance(nn, torch.nn.Module)


@pytest.mark.integ2  # type: ignore
def test_pytorch_const_multi_output() -> None:
    exp.run_basic_test(
        conf.official_examples_path("mnist_pytorch/const-multi-output.yaml"),
        conf.official_examples_path("mnist_pytorch"),
        1,
    )


@pytest.mark.integ2  # type: ignore
def test_pytorch_const_warm_start() -> None:
    """
    Test that specifying an earlier trial checkpoint to warm-start from
    correctly populates the later trials' `warm_start_checkpoint_id` fields.
    """
    config = conf.load_config(conf.official_examples_path("mnist_pytorch/const.yaml"))
    config = conf.set_max_steps(config, 2)

    experiment_id1 = exp.run_basic_test_with_temp_config(
        config, conf.official_examples_path("mnist_pytorch"), 1,
    )

    trials = exp.experiment_trials(experiment_id1)
    assert len(trials) == 1

    first_trial = trials[0]
    first_trial_id = first_trial["id"]

    assert len(first_trial["steps"]) == 2
    first_checkpoint_id = first_trial["steps"][-1]["checkpoint"]["id"]

    config_obj = conf.load_config(conf.official_examples_path("mnist_pytorch/const.yaml"))

    # Change the search method to random, and add a source trial ID to warm
    # start from.
    config_obj["searcher"]["source_trial_id"] = first_trial_id
    config_obj["searcher"]["name"] = "random"
    config_obj["searcher"]["max_steps"] = 1
    config_obj["searcher"]["max_trials"] = 3

    experiment_id2 = exp.run_basic_test_with_temp_config(
        config_obj, conf.official_examples_path("mnist_pytorch"), 3
    )

    trials = exp.experiment_trials(experiment_id2)
    assert len(trials) == 3
    for trial in trials:
        assert trial["warm_start_checkpoint_id"] == first_checkpoint_id


@skip_test_if_not_enough_gpus(8)
@pytest.mark.parallel  # type: ignore
def test_pytorch_const_native_parallel() -> None:
    config = conf.load_config(conf.official_examples_path("mnist_pytorch/const.yaml"))
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_native_parallel(config, True)
    config = conf.set_max_steps(config, 2)

    exp.run_basic_test_with_temp_config(config, conf.official_examples_path("mnist_pytorch"), 1)


@skip_test_if_not_enough_gpus(8)
@pytest.mark.parallel  # type: ignore
@pytest.mark.parametrize("aggregation_frequency", [1, 4])  # type: ignore
@pytest.mark.parametrize("use_amp", [True, False])  # type: ignore
def test_pytorch_const_parallel(aggregation_frequency: int, use_amp: bool) -> None:
    config = conf.load_config(conf.official_examples_path("mnist_pytorch/const.yaml"))
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_native_parallel(config, False)
    config = conf.set_max_steps(config, 2)
    config = conf.set_aggregation_frequency(config, aggregation_frequency)
    if use_amp:
        config = conf.set_amp_level(config, "O1")

    exp.run_basic_test_with_temp_config(config, conf.official_examples_path("mnist_pytorch"), 1)


@skip_test_if_not_enough_gpus(1)
@pytest.mark.integ2  # type: ignore
def test_pytorch_const_with_amp() -> None:
    config = conf.load_config(conf.official_examples_path("mnist_pytorch/const.yaml"))
    config = conf.set_max_steps(config, 2)
    config = conf.set_amp_level(config, "O1")

    exp.run_basic_test_with_temp_config(config, conf.official_examples_path("mnist_pytorch"), 1)


@pytest.mark.integ1  # type: ignore
def test_pytorch_cifar10_const() -> None:
    config = conf.load_config(conf.official_examples_path("cifar10_cnn_pytorch/const.yaml"))
    config = conf.set_max_steps(config, 2)

    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.official_examples_path("cifar10_cnn_pytorch"), 1
    )
    trials = exp.experiment_trials(experiment_id)
    nn = TrialReference(trials[0].id).select_checkpoint(latest=True).load()
    assert isinstance(nn, torch.nn.Module)


@skip_test_if_not_enough_gpus(8)
@pytest.mark.parallel  # type: ignore
def test_pytorch_cifar10_parallel() -> None:
    config = conf.load_config(conf.official_examples_path("cifar10_cnn_pytorch/const.yaml"))
    config = conf.set_max_steps(config, 2)
    config = conf.set_slots_per_trial(config, 8)

    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.official_examples_path("cifar10_cnn_pytorch"), 1
    )
    trials = exp.experiment_trials(experiment_id)
    nn = TrialReference(trials[0].id).select_checkpoint(latest=True).load()
    assert isinstance(nn, torch.nn.Module)
