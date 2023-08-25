# type: ignore
import logging
import os
import pathlib
import typing
import unittest.mock

import pytest
import torch
from torch.distributed import launcher

from determined import core
from tests.experiment import pytorch_utils, utils  # noqa: I100
from tests.launch import test_util


def test_pytorch_mnist_example(tmp_path: pathlib.Path) -> None:
    checkpoint_dir = str(tmp_path.joinpath("checkpoint"))

    config = utils.load_config(utils.tutorials_path("mnist_pytorch/const.yaml"))
    hparams = config["hyperparameters"]

    exp_config = utils.make_default_exp_config(
        hparams,
        scheduling_unit=1,
        searcher_metric="validation_loss",
        checkpoint_dir=checkpoint_dir,
    )
    exp_config.update(config)

    example_path = utils.tutorials_path("mnist_pytorch/model_def.py")
    trial_class = utils.import_class_from_module("MNistTrial", example_path)
    trial_class._searcher_metric = "validation_loss"

    pytorch_utils.train_and_checkpoint(
        trial_class=trial_class,
        hparams=hparams,
        tmp_path=tmp_path,
        exp_config=exp_config,
        steps=(1, 1),
    )


@pytest.mark.skipif(torch.cuda.device_count() < 2, reason="not enough gpus")
@pytest.mark.gpu_parallel
def test_pytorch_parallel(tmp_path: pathlib.Path) -> None:
    launch_config = pytorch_utils.setup_torch_distributed()

    root_logfile = tmp_path.joinpath("root_test.log")

    outputs = launcher.elastic_launch(launch_config, run_mnist)(tmp_path)
    launcher.elastic_launch(launch_config, run_mnist)(tmp_path, outputs[0])

    with open(root_logfile, "r") as f:
        root_log_output = f.readlines()

    validation_size = 10000
    num_workers = 2
    global_batch_size = 64
    scheduling_unit = 1
    per_slot_batch_size = global_batch_size // num_workers
    exp_val_batches = (validation_size + (per_slot_batch_size - 1)) // per_slot_batch_size

    patterns = [
        # Expect two training reports.
        f"report_trial_metrics.*group=training.*steps_completed={1*scheduling_unit}",
        f"report_trial_metrics.*group=training.*steps_completed={2*scheduling_unit}",
        f"validated: {validation_size} records.*in {exp_val_batches} batches",
    ]

    utils.assert_patterns_in_logs(root_log_output, patterns)


@pytest.mark.skipif(torch.cuda.device_count() < 2, reason="not enough gpus")
@pytest.mark.gpu_parallel
def test_cifar10_parallel(tmp_path: pathlib.Path) -> None:
    launch_config = pytorch_utils.setup_torch_distributed()

    outputs = launcher.elastic_launch(launch_config, run_cifar10)(tmp_path)
    launcher.elastic_launch(launch_config, run_cifar10)(tmp_path, outputs[0])



def run_mnist(tmp_path: pathlib.Path, batches_trained: typing.Optional[int] = 0) -> None:
    checkpoint_dir = str(tmp_path.joinpath("checkpoint"))

    root_logfile = tmp_path.joinpath("root_test.log")

    root_file_handler = logging.FileHandler(root_logfile, mode="a+")
    root_logger = logging.getLogger()  # root logger
    root_logger.setLevel(logging.INFO)
    root_logger.addHandler(root_file_handler)

    config = utils.load_config(utils.tutorials_path("mnist_pytorch/const.yaml"))
    hparams = config["hyperparameters"]

    exp_config = utils.make_default_exp_config(
        hparams,
        scheduling_unit=1,
        searcher_metric="validation_loss",
        checkpoint_dir=checkpoint_dir,
    )
    exp_config.update(config)
    exp_config["searcher"]["smaller_is_better"] = True

    example_path = utils.tutorials_path("mnist_pytorch/model_def.py")
    trial_class = utils.import_class_from_module("MNistTrial", example_path)
    trial_class._searcher_metric = "validation_loss"

    if batches_trained == 0:
        return pytorch_utils.train_for_checkpoint(
            trial_class=trial_class,
            hparams=hparams,
            slots_per_trial=2,
            tmp_path=tmp_path,
            exp_config=exp_config,
            steps=1,
        )
    else:
        pytorch_utils.train_from_checkpoint(
            trial_class=trial_class,
            hparams=hparams,
            slots_per_trial=2,
            tmp_path=tmp_path,
            exp_config=exp_config,
            steps=(1, 1),
            batches_trained=batches_trained,
        )
        return True


def run_cifar10(tmp_path: pathlib.Path, batches_trained: int = 0):
    checkpoint_dir = str(tmp_path.joinpath("checkpoint"))

    config = utils.load_config(utils.cv_examples_path("cifar10_pytorch/const.yaml"))
    hparams = config["hyperparameters"]

    exp_config = utils.make_default_exp_config(
        hparams,
        scheduling_unit=1,
        searcher_metric="validation_loss",
        checkpoint_dir=checkpoint_dir,
    )
    exp_config.update(config)
    exp_config["searcher"]["smaller_is_better"] = True

    example_path = utils.cv_examples_path("cifar10_pytorch/model_def.py")
    trial_class = utils.import_class_from_module("CIFARTrial", example_path)
    trial_class._searcher_metric = "validation_error"

    if batches_trained == 0:
        return pytorch_utils.train_for_checkpoint(
            trial_class=trial_class,
            hparams=hparams,
            slots_per_trial=2,
            tmp_path=tmp_path,
            exp_config=exp_config,
            steps=1,
        )
    else:
        pytorch_utils.train_from_checkpoint(
            trial_class=trial_class,
            hparams=hparams,
            slots_per_trial=2,
            tmp_path=tmp_path,
            exp_config=exp_config,
            steps=(1, 1),
            batches_trained=batches_trained,
        )


@unittest.mock.patch(
    "determined.pytorch.experimental._torch_batch_process._initialize_default_inference_context"
)
@unittest.mock.patch(
    "determined.pytorch.experimental._torch_batch_process._synchronize_and_checkpoint"
)
@unittest.mock.patch(
    "determined.pytorch.experimental._torch_batch_process._report_progress_to_master"
)
@pytest.mark.parametrize("rank, num_slots", [[0, 2], [1, 2]])
def test_inference_torch_batch_process_cifar10(
    mock_report_progress_to_master: unittest.mock.MagicMock,
    mock_synchronize_and_checkpoint: unittest.mock.MagicMock,
    mock_initialize_default_inference_context: unittest.mock.MagicMock,
    rank,
    num_slots,
    tmp_path,
):
    # Get main function from example
    example_sub_path = "torch_batch_process_core_api_comparison"
    example_path = utils.features_path(
        os.path.join(example_sub_path, "torch_batch_process_inference.py")
    )
    main_fn = utils.import_class_from_module("main", example_path)

    default_addr = ["0.0.0.12"]

    checkpoint_path = os.path.join(tmp_path, "checkpoint")
    os.mkdir(checkpoint_path)

    mock_initialize_default_inference_context.return_value = core._dummy_init(
        distributed=utils.get_mock_distributed_context(rank=rank),
        checkpoint_storage=str(checkpoint_path),
    )

    with test_util.set_mock_cluster_info(default_addr, rank, num_slots):
        main_fn()
    # Dataset length is 10,000; num_slots is 2; checkpoint_interval is 5; batch_size is 200
    # expected checkpoint count = 10000 / 2 / 5 / 200 = 5
    assert mock_synchronize_and_checkpoint.call_count == 5


@pytest.mark.parametrize("rank, num_slots", [[0, 2], [1, 2]])
def test_inference_core_api_cifar10(rank, num_slots, tmp_path):
    # Get main function from example
    example_sub_path = "torch_batch_process_core_api_comparison"
    example_path = utils.features_path(os.path.join(example_sub_path, "core_api_inference.py"))
    main_fn = utils.import_class_from_module("main", example_path)

    default_addr = ["0.0.0.12"]

    checkpoint_path = os.path.join(tmp_path, "checkpoint")
    os.mkdir(checkpoint_path)

    core_context = unittest.mock.MagicMock()
    # Dataset length is 10,000; num_slots is 2; batch_size is 200
    # when rank == 0, shard_length = 10,000 / 2 / 200 = 25
    # when rank == 1, shard_length = 10,000 / 2 / 200 = 25
    core_context.distributed = utils.get_mock_distributed_context(
        rank=rank, all_gather_return_value=[25, 25]
    )
    core_context.preempt.should_preempt.return_value = False

    with test_util.set_mock_cluster_info(default_addr, rank, num_slots):
        main_fn(core_context)
    # Dataset length is 10,000; num_slots is 2; checkpoint_interval is 5; batch_size is 200
    # expected checkpoint count = 10000 / 2 / 5 / 200 = 5
    assert core_context.distributed.gather.call_count == 5
