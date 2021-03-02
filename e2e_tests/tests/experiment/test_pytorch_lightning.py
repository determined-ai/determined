import pytest

from determined.experimental import Determined
from tests import config as conf
from tests import experiment as exp


@pytest.mark.e2e_cpu  # type: ignore
def test_pytorch_lightning_const_warm_start() -> None:
    """
    Test that specifying an earlier trial checkpoint to warm-start from
    correctly populates the later trials' `warm_start_checkpoint_id` fields.
    """
    config_path = conf.gan_examples_path("gan_mnist_pytorch_lightning/const.yaml")
    context_dir = conf.gan_examples_path("gan_mnist_pytorch_lightning")

    config = conf.load_config(config_path)
    config = conf.set_max_length(config, {"batches": 200})

    experiment_id1 = exp.run_basic_test_with_temp_config(config, context_dir, 1)

    trials = exp.experiment_trials(experiment_id1)
    assert len(trials) == 1

    first_trial = trials[0]
    first_trial_id = first_trial["id"]

    assert len(first_trial["steps"]) == 2
    first_checkpoint_id = first_trial["steps"][-1]["checkpoint"]["id"]

    config_obj = conf.load_config(config_path)

    # Change the search method to random, and add a source trial ID to warm
    # start from.
    config_obj["searcher"]["source_trial_id"] = first_trial_id
    config_obj["searcher"]["name"] = "random"
    config_obj["searcher"]["max_length"] = {"batches": 100}
    config_obj["searcher"]["max_trials"] = 3

    experiment_id2 = exp.run_basic_test_with_temp_config(config_obj, context_dir, 3)

    trials = exp.experiment_trials(experiment_id2)
    assert len(trials) == 3
    for trial in trials:
        assert trial["warm_start_checkpoint_id"] == first_checkpoint_id


@pytest.mark.e2e_gpu  # type: ignore
def test_pytorch_single_gpu_and_load() -> None:
    config = conf.load_config(conf.gan_examples_path("gan_mnist_pytorch_lightning/const.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    experiment_id = exp.run_basic_test_with_temp_config(
        config,
        conf.gan_examples_path("gan_mnist_pytorch_lightning"),
        1,
    )

    (
        Determined(conf.make_master_url())
        .get_experiment(experiment_id)
        .top_checkpoint()
        .load(map_location="cpu")
    )


@pytest.mark.parallel  # type: ignore
def test_pytorch_lightning_gan_parallel() -> None:
    config = conf.load_config(conf.gan_examples_path("gan_mnist_pytorch_lightning/const.yaml"))
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_slots_per_trial(config, 8)

    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.gan_examples_path("gan_mnist_pytorch_lightning"), 1
    )
    trials = exp.experiment_trials(experiment_id)
    (
        Determined(conf.make_master_url())
        .get_trial(trials[0]["id"])
        .select_checkpoint(latest=True)
        .load(map_location="cpu")
    )
