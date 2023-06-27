import warnings
from typing import Callable, List

import pytest

from determined.common import util
from tests import config as conf
from tests import experiment as exp


@pytest.mark.parallel
@pytest.mark.parametrize("image_type", ["PT", "TF2"])
def test_pytorch_cifar10_parallel(
    image_type: str, collect_trial_profiles: Callable[[int], None]
) -> None:
    config = conf.load_config(conf.cv_examples_path("cifar10_pytorch/const.yaml"))
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_profiling_enabled(config)

    if image_type == "PT":
        config = conf.set_pt_image(config)
    elif image_type == "TF2":
        config = conf.set_tf2_image(config)
    else:
        warnings.warn("Using default images", stacklevel=2)

    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.cv_examples_path("cifar10_pytorch"), 1
    )
    trial_id = exp.experiment_trials(experiment_id)[0].trial.id
    collect_trial_profiles(trial_id)


@pytest.mark.parallel
@pytest.mark.parametrize("image_type", ["PT", "TF2"])
def test_pytorch_gan_parallel(
    image_type: str, collect_trial_profiles: Callable[[int], None]
) -> None:
    config = conf.load_config(conf.gan_examples_path("gan_mnist_pytorch/const.yaml"))
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_profiling_enabled(config)

    if image_type == "PT":
        config = conf.set_pt_image(config)
    elif image_type == "TF2":
        config = conf.set_tf2_image(config)
    else:
        warnings.warn("Using default images", stacklevel=2)

    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.gan_examples_path("gan_mnist_pytorch"), 1
    )
    trial_id = exp.experiment_trials(experiment_id)[0].trial.id
    collect_trial_profiles(trial_id)


@pytest.mark.parallel
def test_pytorch_gradient_aggregation() -> None:
    config = conf.load_config(conf.fixtures_path("pytorch_identity/distributed.yaml"))

    exp_id = exp.run_basic_test_with_temp_config(config, conf.fixtures_path("pytorch_identity"), 1)
    trials = exp.experiment_trials(exp_id)
    assert len(trials) == 1
    workloads = exp.workloads_with_validation(trials[0].workloads)
    actual_weights = []
    for wl in workloads:
        if wl.metrics:
            actual_weights.append(wl.metrics.avgMetrics["weight"])

    # independently compute expected metrics
    batch_size = 4
    epoch_size = 64
    num_epochs = 3
    batches = [
        (v[:], v[:])
        for v in (
            [x * 0.1 + 1.0 for x in range(y, y + batch_size)]
            for y in (z % epoch_size for z in range(0, epoch_size * num_epochs, batch_size))
        )
    ]

    lr = 0.001

    def compute_expected_weight(data: List[float], label: List[float], w: float) -> float:
        n = len(data)
        expected_step = 2.0 * lr * sum((d * (l - d * w) for d, l in zip(data, label))) / n
        return w + expected_step

    expected_weights = []
    weight = 0.0
    data: List[float] = []
    label: List[float] = []
    for i, batch in enumerate(batches):
        if i % 2 == 0:
            # for even-numbered batches the optimizer step is a no-op:
            # the weights don't change
            data, label = batch
        else:
            additional_data, additional_label = batch
            data += additional_data
            label += additional_label
            weight = compute_expected_weight(data, label, weight)
        expected_weights.append(weight)

    assert actual_weights == pytest.approx(
        expected_weights
    ), f"{actual_weights} != {expected_weights}"


@pytest.mark.parallel
def test_pytorch_parallel() -> None:
    config = conf.load_config(conf.tutorials_path("mnist_pytorch/const.yaml"))
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_tensor_auto_tuning(config, True)
    config = conf.set_perform_initial_validation(config, True)

    exp_id = exp.run_basic_test_with_temp_config(config, conf.tutorials_path("mnist_pytorch"), 1)
    exp.assert_performed_initial_validation(exp_id)

    # Check on record/batch counts we emitted in logs.
    validation_size = 10000
    num_workers = config.get("resources", {}).get("slots_per_trial", 1)
    global_batch_size = config["hyperparameters"]["global_batch_size"]
    scheduling_unit = config.get("scheduling_unit", 100)
    per_slot_batch_size = global_batch_size // num_workers
    exp_val_batches = (validation_size + (per_slot_batch_size - 1)) // per_slot_batch_size
    patterns = [
        # Expect two training reports.
        f"report_trial_metrics.*metric_type={util._LEGACY_TRAINING}"
        + f".*total_batches={scheduling_unit * 1}",
        f"report_trial_metrics.*metric_type={util._LEGACY_TRAINING}"
        + f".*total_batches={scheduling_unit * 2}",
        f"validated: {validation_size} records.*in {exp_val_batches} batches",
    ]
    trial_id = exp.experiment_trials(exp_id)[0].trial.id
    exp.assert_patterns_in_trial_logs(trial_id, patterns)


@pytest.mark.parallel
def test_distributed_logging() -> None:
    config = conf.load_config(conf.fixtures_path("pytorch_no_op/const.yaml"))
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_max_length(config, {"batches": 1})

    e_id = exp.run_basic_test_with_temp_config(config, conf.fixtures_path("pytorch_no_op"), 1)
    t_id = exp.experiment_trials(e_id)[0].trial.id

    for i in range(config["resources"]["slots_per_trial"]):
        assert exp.check_if_string_present_in_trial_logs(
            t_id, "finished train_batch for rank {}".format(i)
        )


@pytest.mark.parallel
@pytest.mark.parametrize("num_workers,global_batch_size,dataset_len", [(2, 2, 2), (2, 2, 3)])
def test_epoch_sync(num_workers: int, global_batch_size: int, dataset_len: int) -> None:
    """
    Test that epoch_idx is synchronized across all workers regardless of whether the
    number of batches is evenly divisible by the number of workers.
    """
    config = conf.load_config(conf.fixtures_path("pytorch_no_op/const.yaml"))
    config = conf.set_slots_per_trial(config, num_workers)
    max_len_batches = 10
    config = conf.set_max_length(config, {"batches": max_len_batches})
    config = conf.set_hparam(config, "dataset_len", dataset_len)
    config = conf.set_global_batch_size(config, global_batch_size)

    e_id = exp.run_basic_test_with_temp_config(config, conf.fixtures_path("pytorch_no_op"), 1)
    t_id = exp.experiment_trials(e_id)[0].trial.id

    batches_per_epoch = (dataset_len + global_batch_size - 1) // global_batch_size  # ceil

    for batch_idx in range(max_len_batches):
        epoch_idx = batch_idx // batches_per_epoch
        for rank in range(config["resources"]["slots_per_trial"]):
            assert exp.check_if_string_present_in_trial_logs(
                t_id, f"rank {rank} finished batch {batch_idx} in epoch {epoch_idx}"
            )
