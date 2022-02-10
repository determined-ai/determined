import copy
import sys
from typing import Any, Callable, List

import pytest

from determined.experimental import Determined
from tests import config as conf
from tests import experiment as exp


@pytest.mark.e2e_gpu
@pytest.mark.parametrize("aggregation_frequency", [1, 4])
def test_pytorch_11_const(
    aggregation_frequency: int, using_k8s: bool, collect_trial_profiles: Callable[[int], None]
) -> None:
    config = conf.load_config(conf.fixtures_path("mnist_pytorch/const-pytorch11.yaml"))
    config = conf.set_aggregation_frequency(config, aggregation_frequency)
    config = conf.set_profiling_enabled(config)

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

    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.tutorials_path("mnist_pytorch"), 1
    )
    trial_id = exp.experiment_trials(experiment_id)[0]["id"]
    collect_trial_profiles(trial_id)


@pytest.mark.e2e_cpu
def test_pytorch_load(collect_trial_profiles: Callable[[int], None]) -> None:
    config = conf.load_config(conf.fixtures_path("mnist_pytorch/const-pytorch11.yaml"))
    config = conf.set_profiling_enabled(config)

    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.tutorials_path("mnist_pytorch"), 1
    )

    (
        Determined(conf.make_master_url())
        .get_experiment(experiment_id)
        .top_checkpoint()
        .load(map_location="cpu")
    )
    trial_id = exp.experiment_trials(experiment_id)[0]["id"]
    collect_trial_profiles(trial_id)


@pytest.mark.e2e_cpu
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


@pytest.mark.e2e_gpu
@pytest.mark.gpu_required
@pytest.mark.parametrize("api_style", ["apex", "auto", "manual"])
def test_pytorch_const_with_amp(
    api_style: str, collect_trial_profiles: Callable[[int], None]
) -> None:
    config = conf.load_config(conf.fixtures_path("pytorch_amp/" + api_style + "_amp.yaml"))
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_profiling_enabled(config)

    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.fixtures_path("pytorch_amp"), 1
    )
    trial_id = exp.experiment_trials(experiment_id)[0]["id"]
    collect_trial_profiles(trial_id)


@pytest.mark.parallel
def test_pytorch_cifar10_parallel(collect_trial_profiles: Callable[[int], None]) -> None:
    config = conf.load_config(conf.cv_examples_path("cifar10_pytorch/const.yaml"))
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_profiling_enabled(config)

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

    collect_trial_profiles(trials[0]["id"])


@pytest.mark.parallel
def test_pytorch_gan_parallel(collect_trial_profiles: Callable[[int], None]) -> None:
    config = conf.load_config(conf.gan_examples_path("gan_mnist_pytorch/const.yaml"))
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_profiling_enabled(config)

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
    collect_trial_profiles(trials[0]["id"])


@pytest.mark.e2e_cpu
def test_pytorch_native_api() -> None:
    exp_id = exp.create_native_experiment(
        conf.fixtures_path("pytorch_no_op"), [sys.executable, "model_def.py"]
    )
    exp.wait_for_experiment_state(exp_id, "COMPLETED")


@pytest.mark.parallel
def test_pytorch_gradient_aggregation() -> None:
    base_config = conf.load_config(conf.fixtures_path("pytorch_identity/distributed.yaml"))

    def run_and_check(config: Any, expected_steps: int) -> List[float]:
        exp_id = exp.run_basic_test_with_temp_config(
            config, conf.fixtures_path("pytorch_identity"), 1
        )
        trials = exp.experiment_trials(exp_id)
        assert len(trials) == 1
        assert len(trials[0]["steps"]) == expected_steps
        steps = trials[0]["steps"]
        return [step["validation"]["metrics"]["validation_metrics"]["val_loss"] for step in steps]

    loss_without_aggregation = run_and_check(base_config, 40)

    config_with_grad_agg = copy.deepcopy(base_config)
    config_with_grad_agg["hyperparameters"]["global_batch_size"] = 4
    config_with_grad_agg["optimizations"]["aggregation_frequency"] = 2
    loss_with_aggregation = run_and_check(config_with_grad_agg, 80)

    assert loss_with_aggregation[-1] == pytest.approx(
        loss_without_aggregation[-1], 1e-4
    ), f"{loss_with_aggregation}!={loss_without_aggregation}"
    assert loss_with_aggregation[-1] == pytest.approx(0.852, 1e-4)

    # only odd-numbered steps with gradient aggregation change the loss
    odd_numbered_loss = [a for i, a in enumerate(loss_with_aggregation) if i % 2 == 1]
    assert odd_numbered_loss == pytest.approx(loss_without_aggregation)


@pytest.mark.parallel
def test_pytorch_parallel() -> None:
    config = conf.load_config(conf.tutorials_path("mnist_pytorch/const.yaml"))
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_tensor_auto_tuning(config, True)
    config = conf.set_perform_initial_validation(config, True)

    exp_id = exp.run_basic_test_with_temp_config(config, conf.tutorials_path("mnist_pytorch"), 1)
    exp.assert_performed_initial_validation(exp_id)


@pytest.mark.parallel
def test_distributed_logging() -> None:
    config = conf.load_config(conf.fixtures_path("pytorch_no_op/const.yaml"))
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_max_length(config, {"batches": 1})

    e_id = exp.run_basic_test_with_temp_config(config, conf.fixtures_path("pytorch_no_op"), 1)
    t_id = exp.experiment_trials(e_id)[0]["id"]

    for i in range(config["resources"]["slots_per_trial"]):
        assert exp.check_if_string_present_in_trial_logs(
            t_id, "finished train_batch for rank {}".format(i)
        )
