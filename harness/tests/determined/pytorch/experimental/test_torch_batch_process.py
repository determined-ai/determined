import math
import unittest.mock
from typing import Any, Dict, List, Optional

import pytest
import torch
from torch.utils import data

from determined import core, pytorch
from determined.pytorch import experimental
from tests.launch import test_util

DEFAULT_SLOT_IDS = [0]
DEFAULT_ADDRS = ["0.0.0.12"]
DEFAULT_DEVICE = torch.device("cuda", 1)
DEFAULT_STORAGE_PATH = "default_storage_path"


class MySumMetricReducer(pytorch.MetricReducer):
    def __init__(self) -> None:
        self.reset()

    def reset(self) -> None:
        self.sum = 0

    def update(self, value: List[int]) -> None:
        self.sum += sum(value)

    def per_slot_reduce(self) -> int:
        return self.sum

    def cross_slot_reduce(self, per_slot_metrics: List[int]) -> int:
        return sum(per_slot_metrics)


class MyProcessor(experimental.TorchBatchProcessor):
    def __init__(self, context: experimental.TorchBatchProcessorContext) -> None:
        self.context = context
        self.reducer_sum = context.wrap_reducer(reducer=MySumMetricReducer(), name="sum_metric")
        self.rank = context.distributed.get_rank()

    def process_batch(self, batch: Any, batch_idx: int) -> None:
        self.reducer_sum.update(batch)
        print(f"My rank is {self.rank} and current batch is {batch_idx}")


class IndexData(data.Dataset):
    def __init__(self, data_length: int = 50) -> None:
        self.data = data_length

    def __len__(self) -> int:
        return int(self.data)

    def __getitem__(self, idx: int) -> int:
        return idx


def _get_core_context(
    rank: int = 0, should_preempt_results: Optional[List[bool]] = None
) -> unittest.mock.MagicMock:
    mock_core_context = unittest.mock.MagicMock()
    mock_distributed_context = _get_dist_context(rank=rank)
    mock_core_context.__enter__().distributed = mock_distributed_context
    if should_preempt_results is None:
        mock_core_context.__enter__().preempt.should_preempt.return_value = False
    else:
        mock_core_context.__enter__().preempt.should_preempt.side_effect = should_preempt_results
    return mock_core_context


def _get_dist_context(
    rank: int = 0,
    all_gather_return_value: Optional[Any] = None,
    gather_return_value: Optional[Any] = None,
) -> unittest.mock.MagicMock:
    mock_distributed_context = unittest.mock.MagicMock()
    mock_distributed_context.get_rank.return_value = rank
    mock_distributed_context.broadcast.return_value = "mock_checkpoint_uuid"
    mock_distributed_context.allgather.return_value = all_gather_return_value
    mock_distributed_context.gather.return_value = gather_return_value
    return mock_distributed_context


@pytest.mark.parametrize(
    "dataloader_kwargs, batch_size",
    [
        [{"shuffle": True}, 10],
        [{"sampler": unittest.mock.Mock()}, 10],
        [{"batch_sampler": unittest.mock.Mock()}, 10],
        [{"batch_size": 20}, 10],
    ],
    ids=[
        "Shuffle arg provided",
        "Sampler arg provided",
        "Batch_sampler arg provided",
        "batch_size arg provided twice",
    ],
)
def test_torch_batch_process_invalid_dataloader_kwargs(
    dataloader_kwargs: Dict[str, Any],
    batch_size: int,
) -> None:
    # Test calling torch_batch_process with invalid arguments
    with pytest.raises(ValueError):
        experimental._torch_batch_process._validate_dataloader_kwargs(dataloader_kwargs, batch_size)


@unittest.mock.patch(
    "determined.pytorch.experimental._torch_batch_process._initialize_default_inference_context"
)
@unittest.mock.patch(
    "determined.pytorch.experimental._torch_batch_process._synchronize_and_checkpoint"
)
@unittest.mock.patch(
    "determined.pytorch.experimental._torch_batch_process._report_progress_to_master"
)
@pytest.mark.parametrize(
    "data_length, batch_size, checkpoint_interval,rank, num_slots",
    [  # Group 1: data length is divisible by batch_size
        [50, 10, 1, 0, 2],  # multi-slots, chief
        [50, 10, 1, 1, 2],  # multi-slots, non-chief
        # Group 2: data length is not divisible by batch_size
        [50, 20, 1, 0, 2],  # multi-slots, chief
        [50, 20, 1, 1, 2],  # multi-slots, non-chief
    ],
    ids=[
        "Data length divisible by batch_size; multi-slot; Chief",
        "Data length divisible by batch_size; multi-slot; Worker",
        "Data length not divisible by batch_size; multi-slot; Chief",
        "Data length not divisible by batch_size; multi-slot; Worker",
    ],
)
def test_torch_batch_process_all_slots_checkpoint_same_number_of_times(
    mock_report_progress_to_master: unittest.mock.MagicMock,
    mock_synchronize_and_checkpoint: unittest.mock.MagicMock,
    mock_initialize_default_inference_context: unittest.mock.MagicMock,
    data_length: int,
    batch_size: int,
    checkpoint_interval: int,
    rank: int,
    num_slots: int,
) -> None:
    with test_util.set_mock_cluster_info(DEFAULT_ADDRS, rank, num_slots):
        mock_initialize_default_inference_context.return_value = core._dummy_init(
            distributed=_get_dist_context(rank=rank)
        )
        index_dataset = IndexData(data_length)

        MyProcessorCLS = unittest.mock.Mock()
        my_processor_instance = unittest.mock.Mock()
        MyProcessorCLS.return_value = my_processor_instance

        experimental.torch_batch_process(
            dataset=index_dataset,
            batch_processor_cls=MyProcessorCLS,
            batch_size=batch_size,
            checkpoint_interval=checkpoint_interval,
        )

        # Regardless of a worker's rank, _synchronize_and_checkpoint should be called the same
        # amount of time
        # Otherwise, _synchronize_and_checkpoint will hang forever waiting for the workers.
        # Note that rank is not part of the calculation of times_iterate
        times_iterate = math.ceil(
            math.ceil(data_length / batch_size / num_slots) / checkpoint_interval
        )
        assert mock_synchronize_and_checkpoint.call_count == times_iterate
        # on_checkpoint_start should be called the same number of times as
        # _synchronize_and_checkpoint
        assert my_processor_instance.on_checkpoint_start.call_count == times_iterate


@unittest.mock.patch(
    "determined.pytorch.experimental._torch_batch_process._initialize_default_inference_context"
)
@unittest.mock.patch(
    "determined.pytorch.experimental._torch_batch_process._synchronize_and_checkpoint"
)
@unittest.mock.patch(
    "determined.pytorch.experimental._torch_batch_process._report_progress_to_master"
)
@pytest.mark.parametrize(
    "data_length, batch_size, checkpoint_interval,rank, num_slots",
    [  # Group 1: data length is divisible by batch_size
        [50, 10, 1, 0, 2],  # multi-slots, chief
        [50, 10, 1, 1, 2],  # multi-slots, non-chief
        # Group 2: data length is not divisible by batch_size
        [50, 20, 1, 0, 2],  # multi-slots, chief
        [50, 20, 1, 1, 2],  # multi-slots, non-chief
    ],
    ids=[
        "Data length divisible by batch_size; multi-slot; Chief",
        "Data length divisible by batch_size; multi-slot; Worker",
        "Data length not divisible by batch_size; multi-slot; Chief",
        "Data length not divisible by batch_size; multi-slot; Worker",
    ],
)
def test_torch_batch_process_only_cheif_reports_progress(
    mock_report_progress_to_master: unittest.mock.MagicMock,
    mock_synchronize_and_checkpoint: unittest.mock.MagicMock,
    mock_initialize_default_inference_context: unittest.mock.MagicMock,
    data_length: int,
    batch_size: int,
    checkpoint_interval: int,
    rank: int,
    num_slots: int,
) -> None:
    with test_util.set_mock_cluster_info(DEFAULT_ADDRS, rank, num_slots):
        mock_initialize_default_inference_context.return_value = core._dummy_init(
            distributed=_get_dist_context(rank=rank)
        )
        index_dataset = IndexData(data_length)

        MyProcessorCLS = unittest.mock.Mock()
        my_processor_instance = unittest.mock.Mock()
        MyProcessorCLS.return_value = my_processor_instance

        experimental.torch_batch_process(
            dataset=index_dataset,
            batch_processor_cls=MyProcessorCLS,
            batch_size=batch_size,
            checkpoint_interval=checkpoint_interval,
        )
        # For chief, _report_progress_to_master should be called the same number of times as
        # _synchronize_and_checkpoint
        times_iterate = math.ceil(
            math.ceil(data_length / batch_size / num_slots) / checkpoint_interval
        )
        assert mock_report_progress_to_master.call_count == (times_iterate if rank == 0 else 0)


@unittest.mock.patch(
    "determined.pytorch.experimental._torch_batch_process._initialize_default_inference_context"
)
@unittest.mock.patch(
    "determined.pytorch.experimental._torch_batch_process._synchronize_and_checkpoint"
)
@pytest.mark.parametrize(
    "data_length, batch_size, checkpoint_interval, rank, num_slots, "
    "expected_process_batch_call_count",
    [  # Group 1: data length is divisible by batch_size
        # Total batches for worker (5), iterate for 5 times
        [50, 10, 1, 0, 1, 5],
        # Total batches for worker 0 (3), iterate for 3 times
        [50, 10, 1, 0, 2, 3],
        # Total batches for worker 1 (2), iterate for 2 times
        [50, 10, 1, 1, 2, 2],
        # Group 2: data length is NOT divisible by batch_size
        # Total batches for worker (3), iterate for 3 times
        [50, 20, 1, 0, 1, 3],
        # Total batches for worker 0 (2), iterate for 2 times
        [50, 20, 1, 0, 2, 2],
        # Total batches for worker 1 (1), iterate for 1 times
        [50, 20, 1, 1, 2, 1],
    ],
    ids=[
        "Data length divisible by batch_size; single-slot",
        "Data length divisible by batch_size; multi-slot; Chief",
        "Data length divisible by batch_size; multi-slot; Worker",
        "Data length not divisible by batch_size; single-slot",
        "Data length not divisible by batch_size; multi-slot; Chief",
        "Data length not divisible by batch_size; multi-slot; Worker",
    ],
)
def test_torch_batch_process_process_batch_called_expected_number_of_times(
    mock_synchronize_and_checkpoint: unittest.mock.MagicMock,
    mock_initialize_default_inference_context: unittest.mock.MagicMock,
    data_length: int,
    batch_size: int,
    checkpoint_interval: int,
    rank: int,
    num_slots: int,
    expected_process_batch_call_count: int,
) -> None:
    with test_util.set_mock_cluster_info(DEFAULT_ADDRS, rank, num_slots):
        mock_initialize_default_inference_context.return_value = core._dummy_init(
            distributed=_get_dist_context(rank=rank)
        )
        index_dataset = IndexData(data_length)

        MyProcessorCLS = unittest.mock.Mock()
        my_processor_instance = unittest.mock.Mock()
        MyProcessorCLS.return_value = my_processor_instance

        experimental.torch_batch_process(
            dataset=index_dataset,
            batch_processor_cls=MyProcessorCLS,
            batch_size=batch_size,
            checkpoint_interval=checkpoint_interval,
        )

        assert my_processor_instance.process_batch.call_count == expected_process_batch_call_count
        # on_finish should only be called once
        assert my_processor_instance.on_finish.call_count == 1


@unittest.mock.patch("determined.pytorch.experimental._torch_batch_process._load_state")
@unittest.mock.patch(
    "determined.pytorch.experimental._torch_batch_process._initialize_default_inference_context"
)
@unittest.mock.patch(
    "determined.pytorch.experimental._torch_batch_process._synchronize_and_checkpoint"
)
@pytest.mark.parametrize(
    "data_length, batch_size, checkpoint_interval, rank, num_slots, "
    "steps_completed, expected_process_batch_call_count",
    [  # total batches for worker (5), completed (2), iterate for 3 batches
        [50, 10, 1, 0, 1, 2, 3],
        # total batches for worker 0 (3), completed (2), iterate for 1 batches
        [50, 10, 1, 0, 2, 2, 1],
        # total batches for worker 1 (2), completed (2), iterate for 0 batches
        [50, 10, 1, 1, 2, 2, 0],
    ],
    ids=[
        "Single-slot; total_batches=5; completed_batches=2",
        "Multi-slot; chief; total_batches=5; completed_batches=2",
        "Multi-slot; worker; total_batches=5; completed_batches=2",
    ],
)
def test_torch_batch_process_skip_completed_batches(
    mock_synchronize_and_checkpoint: unittest.mock.MagicMock,
    mock_initialize_default_inference_context: unittest.mock.MagicMock,
    mock_load_state: unittest.mock.MagicMock,
    data_length: int,
    batch_size: int,
    checkpoint_interval: int,
    rank: int,
    num_slots: int,
    steps_completed: int,
    expected_process_batch_call_count: int,
) -> None:
    with test_util.set_mock_cluster_info(
        DEFAULT_ADDRS, rank, num_slots, latest_checkpoint="fake_latest_checkpoint"
    ):
        # Mock core context as _dummy_init would call storage functions
        mock_initialize_default_inference_context.return_value = _get_core_context(rank=rank)
        index_dataset = IndexData(data_length)

        MyProcessorCLS = unittest.mock.Mock()
        my_processor_instance = unittest.mock.Mock()
        MyProcessorCLS.return_value = my_processor_instance

        mock_load_state.return_value = {
            "steps_completed": steps_completed,
            "default_output_uuid": "abc",
        }

        experimental.torch_batch_process(
            dataset=index_dataset,
            batch_processor_cls=MyProcessorCLS,
            batch_size=batch_size,
            checkpoint_interval=checkpoint_interval,
        )

        assert my_processor_instance.process_batch.call_count == expected_process_batch_call_count
        # on_finish should only be called once
        assert my_processor_instance.on_finish.call_count == 1


@unittest.mock.patch(
    "determined.pytorch.experimental._torch_batch_process._initialize_default_inference_context"
)
@unittest.mock.patch(
    "determined.pytorch.experimental._torch_batch_process._synchronize_and_checkpoint"
)
@pytest.mark.parametrize(
    "data_length, batch_size, checkpoint_interval, rank, num_slots, "
    "max_batches, expected_process_batch_call_count",
    [  # max_batches (2) < total batches (3) -> iterate for 2 batches
        [50, 10, 1, 0, 2, 2, 2],
    ],
)
def test_torch_batch_follows_valid_max_batches(
    mock_synchronize_and_checkpoint: unittest.mock.MagicMock,
    mock_initialize_default_inference_context: unittest.mock.MagicMock,
    data_length: int,
    batch_size: int,
    checkpoint_interval: int,
    rank: int,
    num_slots: int,
    max_batches: int,
    expected_process_batch_call_count: int,
) -> None:
    with test_util.set_mock_cluster_info(DEFAULT_ADDRS, rank, num_slots):
        mock_initialize_default_inference_context.return_value = core._dummy_init(
            distributed=_get_dist_context(rank=rank)
        )

        index_dataset = IndexData(data_length)

        MyProcessorCLS = unittest.mock.Mock()
        my_processor_instance = unittest.mock.Mock()
        MyProcessorCLS.return_value = my_processor_instance

        experimental.torch_batch_process(
            dataset=index_dataset,
            batch_processor_cls=MyProcessorCLS,
            batch_size=batch_size,
            checkpoint_interval=checkpoint_interval,
            max_batches=max_batches,
        )

        assert my_processor_instance.process_batch.call_count == expected_process_batch_call_count
        # on_finish should only be called once
        assert my_processor_instance.on_finish.call_count == 1


@unittest.mock.patch(
    "determined.pytorch.experimental._torch_batch_process._initialize_default_inference_context"
)
@unittest.mock.patch(
    "determined.pytorch.experimental._torch_batch_process._synchronize_and_checkpoint"
)
@pytest.mark.parametrize(
    "data_length, batch_size, checkpoint_interval, rank, num_slots, "
    "max_batches, expected_process_batch_call_count",
    [
        # max_batches (10) > total batches (3) -> iterate for 3 batches
        [50, 10, 1, 0, 2, 10, 3],
    ],
)
def test_torch_batch_process_ignores_invalid_max_batches_and_raises_warning(
    mock_synchronize_and_checkpoint: unittest.mock.MagicMock,
    mock_initialize_default_inference_context: unittest.mock.MagicMock,
    data_length: int,
    batch_size: int,
    checkpoint_interval: int,
    rank: int,
    num_slots: int,
    max_batches: int,
    expected_process_batch_call_count: int,
) -> None:
    with test_util.set_mock_cluster_info(DEFAULT_ADDRS, rank, num_slots):
        mock_initialize_default_inference_context.return_value = core._dummy_init(
            distributed=_get_dist_context(rank=rank)
        )

        index_dataset = IndexData(data_length)

        MyProcessorCLS = unittest.mock.Mock()
        my_processor_instance = unittest.mock.Mock()
        MyProcessorCLS.return_value = my_processor_instance

        with pytest.warns(Warning):
            experimental.torch_batch_process(
                dataset=index_dataset,
                batch_processor_cls=MyProcessorCLS,
                batch_size=batch_size,
                checkpoint_interval=checkpoint_interval,
                max_batches=max_batches,
            )
        assert my_processor_instance.process_batch.call_count == expected_process_batch_call_count
        # on_finish should only be called once
        assert my_processor_instance.on_finish.call_count == 1


@unittest.mock.patch(
    "determined.pytorch.experimental._torch_batch_process._initialize_default_inference_context"
)
@unittest.mock.patch(
    "determined.pytorch.experimental._torch_batch_process._synchronize_and_checkpoint"
)
@unittest.mock.patch("determined.pytorch.experimental._torch_batch_process._reduce_metrics")
def test_torch_batch_process_preemption(
    mock_reduce_metrics: unittest.mock.MagicMock,
    mock_synchronize_and_checkpoint: unittest.mock.MagicMock,
    mock_initialize_default_inference_context: unittest.mock.MagicMock,
) -> None:
    with test_util.set_mock_cluster_info(DEFAULT_ADDRS, 0, 1):
        # Mock core context to set preemption result
        mock_initialize_default_inference_context.return_value = _get_core_context(
            should_preempt_results=[False, True]
        )
        index_dataset = IndexData(100)

        MyProcessorCLS = unittest.mock.Mock()
        my_processor_instance = unittest.mock.Mock()
        MyProcessorCLS.return_value = my_processor_instance

        experimental.torch_batch_process(
            dataset=index_dataset,
            batch_processor_cls=MyProcessorCLS,
            batch_size=10,
            checkpoint_interval=2,
        )

        # Without preemption, we should iterate for 10 batches (100 / 10).
        # We check preemption every two batches, and at the second check, preemption is true
        # Thus, we expect to iterate for 4 batches total

        assert my_processor_instance.process_batch.call_count == 4
        # on_finish should not be called as the job was preempted before completion
        assert my_processor_instance.on_finish.call_count == 0
        # Before preempting, we run reduce metrics once
        # If there is no metrics reducer, this call is a no-op
        assert mock_reduce_metrics.call_count == 1


@unittest.mock.patch(
    "determined.pytorch.experimental._torch_batch_process._initialize_default_inference_context"
)
@unittest.mock.patch(
    "determined.pytorch.experimental._torch_batch_process._synchronize_and_checkpoint"
)
def test_torch_batch_process_reduce_metrics(
    mock_synchronize_and_checkpoint: unittest.mock.MagicMock,
    mock_initialize_default_inference_context: unittest.mock.MagicMock,
) -> None:
    # Simulate a two worker run, and we are worker 0 (chief)
    # The entire dataset is [0, 1, 2, 3 , 4, 5, 6, 7, 8, 9]
    # With batch_size of 5,
    # - worker 0 works on [0, 1, 2, 3, 4], sum = 10
    # - worker 1 works on [5, 6, 7, 8, 9], sum = 35
    # The metric reducer simpler sum up all the values.
    # Therefore, by the end of the iteration, metric_reducer 0 would have self.sum == 10
    # metric_reducer 1 would have self.sum == 35
    with test_util.set_mock_cluster_info(DEFAULT_ADDRS, 0, 2):
        index_dataset = IndexData(10)
        # Mock core context as we are asserting call count on train.report_validation_metrics
        mock_core_context = _get_core_context(rank=0)
        mock_core_context.__enter__().distributed.gather.return_value = [[10], [35]]
        mock_initialize_default_inference_context.return_value = mock_core_context

        experimental.torch_batch_process(
            dataset=index_dataset,
            batch_processor_cls=MyProcessor,
            batch_size=5,
            checkpoint_interval=1,
        )

        # Assert gather was first called with worker 0's sum amount: 10
        assert mock_core_context.__enter__().distributed.gather.call_args_list[
            0
        ] == unittest.mock.call([10])
        # Assert that we report metrics with the sum across workers: 45
        mock_core_context.__enter__().train.report_validation_metrics.assert_called_once_with(
            steps_completed=1,
            metrics={"sum_metric": 45},
        )


@unittest.mock.patch(
    "determined.pytorch.experimental._torch_batch_process._initialize_default_inference_context"
)
@unittest.mock.patch(
    "determined.pytorch.experimental._torch_batch_process._synchronize_and_checkpoint"
)
@pytest.mark.parametrize(
    "checkpoint_interval", [-1, 0], ids=["Invalid checkpoint_interval", "Valid checkpoint_interval"]
)
def test_torch_batch_process_invalid_checkpoint_interval_raises_error(
    mock_synchronize_and_checkpoint: unittest.mock.MagicMock,
    mock_initialize_default_inference_context: unittest.mock.MagicMock,
    checkpoint_interval: int,
) -> None:
    with test_util.set_mock_cluster_info(DEFAULT_ADDRS, 0, 1):
        mock_initialize_default_inference_context.return_value = core._dummy_init(
            distributed=_get_dist_context()
        )
        index_dataset = IndexData()

        MyProcessorCLS = unittest.mock.Mock()
        my_processor_instance = unittest.mock.Mock()
        MyProcessorCLS.return_value = my_processor_instance

        with pytest.raises(ValueError):
            experimental.torch_batch_process(
                dataset=index_dataset,
                batch_processor_cls=MyProcessorCLS,
                checkpoint_interval=checkpoint_interval,
            )


@unittest.mock.patch("determined.pytorch.to_device")
@unittest.mock.patch("determined.pytorch.experimental._torch_batch_process.get_default_device")
def test_torch_batch_processor_context_to_device_sends_tensor_to_device(
    mock_get_default_device: unittest.mock.MagicMock,
    mock_pytorch_to_device: unittest.mock.MagicMock,
) -> None:
    mock_get_default_device.return_value = DEFAULT_DEVICE

    core_context = core._dummy_init(distributed=_get_dist_context())
    torch_batch_processor_context = experimental.TorchBatchProcessorContext(
        core_context, DEFAULT_STORAGE_PATH
    )

    tensor_1 = torch.zeros(4)
    torch_batch_processor_context.to_device(tensor_1)

    to_device_call_args = mock_pytorch_to_device.call_args[0]
    assert to_device_call_args[1] == DEFAULT_DEVICE
    assert torch.all(tensor_1.eq(to_device_call_args[0]))


@unittest.mock.patch("determined.pytorch.experimental._torch_batch_process.get_default_device")
def test_torch_batch_processor_context_prepare_model_for_inference_calls_eval(
    mock_get_default_device: unittest.mock.MagicMock,
) -> None:
    mock_get_default_device.return_value = DEFAULT_DEVICE

    core_context = core._dummy_init(distributed=_get_dist_context())
    torch_batch_processor_context = experimental.TorchBatchProcessorContext(
        core_context, DEFAULT_STORAGE_PATH
    )

    model = unittest.mock.MagicMock()
    torch_batch_processor_context.prepare_model_for_inference(model)

    # Test model.eval() called
    model.eval.assert_called_once()


@unittest.mock.patch("determined.pytorch.experimental._torch_batch_process.get_default_device")
def test_torch_batch_processor_context_prepare_model_for_inference_calls_to_device(
    mock_get_default_device: unittest.mock.MagicMock,
) -> None:
    mock_get_default_device.return_value = DEFAULT_DEVICE

    core_context = core._dummy_init(distributed=_get_dist_context())
    torch_batch_processor_context = experimental.TorchBatchProcessorContext(
        core_context, DEFAULT_STORAGE_PATH
    )

    model = unittest.mock.MagicMock()
    torch_batch_processor_context.prepare_model_for_inference(model)

    # Tested model.to(device) called
    model.to.assert_called_once_with(DEFAULT_DEVICE)


@unittest.mock.patch("determined.pytorch.experimental._torch_batch_process.get_default_device")
def test_torch_batch_processor_context_upload_path_sets_use_default_storage(
    mock_get_default_device: unittest.mock.MagicMock,
) -> None:
    mock_get_default_device.return_value = DEFAULT_DEVICE

    # Mock core context to test num times called of _storage_manager
    core_context = _get_core_context()
    torch_batch_processor_context = experimental.TorchBatchProcessorContext(
        core_context, DEFAULT_STORAGE_PATH
    )

    assert torch_batch_processor_context._use_default_storage is False
    torch_batch_processor_context.upload_path()

    core_context.checkpoint._storage_manager.store_path.assert_called_once_with(
        DEFAULT_STORAGE_PATH
    )
    assert torch_batch_processor_context._use_default_storage is True


@unittest.mock.patch("determined.pytorch.experimental._torch_batch_process.torch.cuda.device_count")
@pytest.mark.parametrize(
    "local_rank, device_count, expected_device",
    [
        [0, 0, torch.device("cpu")],
        [0, 1, torch.device("cuda", 0)],
        [1, 2, torch.device("cuda", 1)],
    ],
    ids=[
        "No CUDA device available",
        "One CUDA device available",
        "Two CUDA devices available; local_rank=1",
    ],
)
def test_get_default_device(
    mock_torch_device_count: unittest.mock.MagicMock,
    local_rank: int,
    device_count: int,
    expected_device: torch.device,
) -> None:
    core_context = core._dummy_init(distributed=_get_dist_context())
    core_context.distributed.local_rank = local_rank
    mock_torch_device_count.return_value = device_count

    default_device = experimental.get_default_device(core_context)
    assert expected_device == default_device
