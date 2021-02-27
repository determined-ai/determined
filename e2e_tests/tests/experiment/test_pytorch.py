import pytest

from determined.experimental import Determined
from tests import config as conf
from tests import experiment as exp


@pytest.mark.e2e_gpu  # type: ignore
@pytest.mark.parametrize("aggregation_frequency", [1, 4])  # type: ignore
def test_pytorch_11_const(aggregation_frequency: int, using_k8s: bool) -> None:
    config = conf.load_config(conf.fixtures_path("mnist_pytorch/const-pytorch11.yaml"))
    config = conf.set_aggregation_frequency(config, aggregation_frequency)

    if using_k8s:
        pod_spec = {
            "metadata": {"labels": {"ci": "testing"}},
            "spec": {
                "containers": [
                    {
                        "name": "determined-container",
                        "volumeMounts": [{"name": "temp1", "mountPath": "/random"}],
                    }
                ],
                "volumes": [{"name": "temp1", "emptyDir": {}}],
            },
        }
        config = conf.set_pod_spec(config, pod_spec)

    exp.run_basic_test_with_temp_config(config, conf.tutorials_path("mnist_pytorch"), 1)


@pytest.mark.e2e_cpu  # type: ignore
def test_pytorch_load() -> None:
    config = conf.load_config(conf.fixtures_path("mnist_pytorch/const-pytorch11.yaml"))

    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.tutorials_path("mnist_pytorch"), 1
    )

    (
        Determined(conf.make_master_url())
        .get_experiment(experiment_id)
        .top_checkpoint()
        .load(map_location="cpu")
    )


@pytest.mark.e2e_cpu  # type: ignore
def test_pytorch_const_warm_start() -> None:
    """
    Test that specifying an earlier trial checkpoint to warm-start from
    correctly populates the later trials' `warm_start_checkpoint_id` fields.
    """
    config = conf.load_config(conf.tutorials_path("mnist_pytorch/const.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    experiment_id1 = exp.run_basic_test_with_temp_config(
        config, conf.tutorials_path("mnist_pytorch"), 1
    )

    trials = exp.experiment_trials(experiment_id1)
    assert len(trials) == 1

    first_trial = trials[0]
    first_trial_id = first_trial["id"]

    assert len(first_trial["steps"]) == 2
    first_checkpoint_id = first_trial["steps"][-1]["checkpoint"]["id"]

    config_obj = conf.load_config(conf.tutorials_path("mnist_pytorch/const.yaml"))

    # Change the search method to random, and add a source trial ID to warm
    # start from.
    config_obj["searcher"]["source_trial_id"] = first_trial_id
    config_obj["searcher"]["name"] = "random"
    config_obj["searcher"]["max_length"] = {"batches": 100}
    config_obj["searcher"]["max_trials"] = 3

    experiment_id2 = exp.run_basic_test_with_temp_config(
        config_obj, conf.tutorials_path("mnist_pytorch"), 3
    )

    trials = exp.experiment_trials(experiment_id2)
    assert len(trials) == 3
    for trial in trials:
        assert trial["warm_start_checkpoint_id"] == first_checkpoint_id


@pytest.mark.parallel  # type: ignore
@pytest.mark.parametrize("aggregation_frequency", [1, 4])  # type: ignore
@pytest.mark.parametrize("use_apex", [True, False])  # type: ignore
def test_pytorch_const_parallel(aggregation_frequency: int, use_apex: bool) -> None:
    if use_apex and aggregation_frequency > 1:
        pytest.skip("Mixed precision is not support with aggregation frequency > 1.")

    config = conf.load_config(conf.tutorials_path("mnist_pytorch/const.yaml"))
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_aggregation_frequency(config, aggregation_frequency)
    if use_apex:
        config = conf.set_apex_level(config, "O1")

    exp.run_basic_test_with_temp_config(config, conf.tutorials_path("mnist_pytorch"), 1)


@pytest.mark.e2e_gpu  # type: ignore
def test_pytorch_const_with_apex() -> None:
    config = conf.load_config(conf.tutorials_path("mnist_pytorch/const.yaml"))
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_apex_level(config, "O1")

    exp.run_basic_test_with_temp_config(config, conf.tutorials_path("mnist_pytorch"), 1)


@pytest.mark.e2e_gpu  # type: ignore
@pytest.mark.parametrize("api_style", ["auto", "manual"])  # type: ignore
def test_pytorch_const_with_amp(api_style: str) -> None:
    config = conf.load_config(conf.fixtures_path("pytorch_amp/" + api_style + "_amp.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(config, conf.fixtures_path("pytorch_amp"), 1)


@pytest.mark.parallel  # type: ignore
def test_pytorch_cifar10_parallel() -> None:
    config = conf.load_config(conf.cv_examples_path("cifar10_pytorch/const.yaml"))
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_slots_per_trial(config, 8)

    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.cv_examples_path("cifar10_pytorch"), 1
    )
    trials = exp.experiment_trials(experiment_id)
    (
        Determined(conf.make_master_url())
        .get_trial(trials[0]["id"])
        .select_checkpoint(latest=True)
        .load(map_location="cpu")
    )


@pytest.mark.parallel  # type: ignore
def test_pytorch_gan_parallel() -> None:
    config = conf.load_config(conf.gan_examples_path("gan_mnist_pytorch/const.yaml"))
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_slots_per_trial(config, 8)

    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.gan_examples_path("gan_mnist_pytorch"), 1
    )
    trials = exp.experiment_trials(experiment_id)
    (
        Determined(conf.make_master_url())
        .get_trial(trials[0]["id"])
        .select_checkpoint(latest=True)
        .load(map_location="cpu")
    )
