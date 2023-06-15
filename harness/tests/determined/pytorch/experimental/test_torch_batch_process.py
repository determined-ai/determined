import math
from typing import Type
from unittest.mock import MagicMock, Mock, call, patch

import pytest
from torch.utils.data import Dataset

from determined import pytorch
from determined.pytorch.experimental._torch_batch_process import (
    TorchBatchProcessor,
    torch_batch_process,
)


@pytest.fixture
def batch_processor_with_avg_metric_reducer() -> Type[TorchBatchProcessor]:
    class MySumMetricReducer(pytorch.MetricReducer):
        def __init__(self):
            self.reset()

        def reset(self):
            self.sum = 0

        def update(self, value):
            self.sum += sum(value)

        def per_slot_reduce(self):
            return self.sum

        def cross_slot_reduce(self, per_slot_metrics):
            return sum(per_slot_metrics)

    class MyProcessor(TorchBatchProcessor):
        def __init__(self, context):
            self.context = context
            self.reducer_sum = context.wrap_reducer(reducer=MySumMetricReducer(), name="sum_metric")

        def process_batch(self, batch, batch_idx) -> None:
            self.reducer_sum.update(batch)

    return MyProcessor


def _get_index_dataset(data_length=50) -> Dataset:
    class IndexData(Dataset):
        def __init__(self, data_length):
            self.data = data_length

        def __len__(self):
            return self.data

        def __getitem__(self, idx):
            return idx

    return IndexData(data_length)


def _get_core_context(rank=0, should_preempt_results=None) -> MagicMock:
    mock_core_context = MagicMock()
    mock_distributed_context = MagicMock()
    mock_distributed_context.get_rank.return_value = rank
    mock_distributed_context.broadcast.return_value = "mock_checkpoint_uuid"
    mock_core_context.__enter__().distributed = mock_distributed_context
    if should_preempt_results is None:
        mock_core_context.__enter__().preempt.should_preempt.return_value = False
    else:
        mock_core_context.__enter__().preempt.should_preempt.side_effect = should_preempt_results
    return mock_core_context


def _get_det_info(slot_ids=[0], container_addrs=["0.0.0.12"], latest_checkpoint=None) -> MagicMock:
    mock_cluster_info = MagicMock()
    mock_cluster_info.slot_ids = slot_ids
    mock_cluster_info.container_addrs = container_addrs
    mock_cluster_info.latest_checkpoint = latest_checkpoint
    return mock_cluster_info


@patch("determined.pytorch.experimental._torch_batch_process.initialize_default_inference_context")
@patch("determined.get_cluster_info")
@patch("determined.pytorch.experimental._torch_batch_process._synchronize_and_checkpoint")
@patch("determined.pytorch.experimental._torch_batch_process._report_progress_to_master")
@patch("determined.pytorch.experimental._torch_batch_process._reduce_metrics")
@pytest.mark.parametrize(
    "dataloader_kwargs, batch_size",
    [
        [{"shuffle": True}, 10],
        [{"sampler": Mock()}, 10],
        [{"batch_sampler": Mock()}, 10],
        [{"batch_size": 20}, 10],
    ],
)
def test_torch_batch_process_dataloader_kwargs_validation(
    mock_reduce_metrics,
    mock_report_progress_to_master,
    mock_synchronize_and_checkpoint,
    mock_get_cluster_info,
    mock_initialize_default_inference_context,
    dataloader_kwargs,
    batch_size,
):
    mock_get_cluster_info.return_value = _get_det_info()
    mock_initialize_default_inference_context.return_value = _get_core_context()
    index_dataset = _get_index_dataset()

    MyProcessorCLS = Mock()
    my_processor_instance = Mock()
    MyProcessorCLS.return_value = my_processor_instance

    # Test calling torch_batch_process with invalid arguments
    with pytest.raises(Exception):
        torch_batch_process(
            dataset=index_dataset,
            batch_processor_cls=MyProcessorCLS,
            batch_size=batch_size,
            dataloader_kwargs=dataloader_kwargs,
        )


@patch("determined.pytorch.experimental._torch_batch_process.initialize_default_inference_context")
@patch("determined.get_cluster_info")
@patch("determined.pytorch.experimental._torch_batch_process._synchronize_and_checkpoint")
@patch("determined.pytorch.experimental._torch_batch_process._report_progress_to_master")
@patch("determined.pytorch.experimental._torch_batch_process._reduce_metrics")
@pytest.mark.parametrize(
    "data_length, batch_size, checkpoint_interval, rank, slot_ids",
    [  # Group 1: data length is divisible by batch_size
        [50, 10, 1, 0, [0, 1]],  # multi-slots, chief
        [50, 10, 1, 1, [0, 1]],  # multi-slots, non-chief
        # Group 2: data length is not divisible by batch_size
        [50, 20, 1, 0, [0, 1]],  # multi-slots, chief
        [50, 20, 1, 1, [0, 1]],  # multi-slots, non-chief
    ],
)
def test_torch_batch_process_times_synchronize(
    mock_reduce_metrics,
    mock_report_progress_to_master,
    mock_synchronize_and_checkpoint,
    mock_get_cluster_info,
    mock_initialize_default_inference_context,
    data_length,
    batch_size,
    checkpoint_interval,
    rank,
    slot_ids,
):
    mock_get_cluster_info.return_value = _get_det_info(slot_ids=slot_ids)
    mock_initialize_default_inference_context.return_value = _get_core_context(rank=rank)
    index_dataset = _get_index_dataset(data_length)

    MyProcessorCLS = Mock()
    my_processor_instance = Mock()
    MyProcessorCLS.return_value = my_processor_instance

    torch_batch_process(
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
        math.ceil(data_length / batch_size / len(slot_ids)) / checkpoint_interval
    )

    assert mock_synchronize_and_checkpoint.call_count == times_iterate
    # on_checkpoint_start should be called the same number of times as _synchronize_and_checkpoint
    assert my_processor_instance.on_checkpoint_start.call_count == times_iterate
    # For chief, _report_progress_to_master should be called the same number of times as
    # _synchronize_and_checkpoint
    assert mock_report_progress_to_master.call_count == (times_iterate if rank == 0 else 0)


@patch("determined.pytorch.experimental._torch_batch_process.initialize_default_inference_context")
@patch("determined.get_cluster_info")
@patch("determined.pytorch.experimental._torch_batch_process._synchronize_and_checkpoint")
@patch("determined.pytorch.experimental._torch_batch_process._report_progress_to_master")
@patch("determined.pytorch.experimental._torch_batch_process._reduce_metrics")
@pytest.mark.parametrize(
    "data_length, batch_size, checkpoint_interval, rank, slot_ids, "
    "expected_process_batch_call_count",
    [  # Group 1: data length is divisible by batch_size
        # Total batches for worker (5), iterate for 5 times
        [50, 10, 1, 0, [0], 5],
        # Total batches for worker 0 (3), iterate for 3 times
        [50, 10, 1, 0, [0, 1], 3],
        # Total batches for worker 1 (2), iterate for 2 times
        [50, 10, 1, 1, [0, 1], 2],
        # Group 2: data length is NOT divisible by batch_size
        # Total batches for worker (3), iterate for 3 times
        [50, 20, 1, 0, [0], 3],
        # Total batches for worker 0 (2), iterate for 2 times
        [50, 20, 1, 0, [0, 1], 2],
        # Total batches for worker 1 (1), iterate for 1 times
        [50, 20, 1, 1, [0, 1], 1],
    ],
)
def test_torch_batch_process_times_process_batch(
    mock_reduce_metrics,
    mock_report_progress_to_master,
    mock_synchronize_and_checkpoint,
    mock_get_cluster_info,
    mock_initialize_default_inference_context,
    data_length,
    batch_size,
    checkpoint_interval,
    rank,
    slot_ids,
    expected_process_batch_call_count,
):
    mock_get_cluster_info.return_value = _get_det_info(slot_ids=slot_ids)
    mock_initialize_default_inference_context.return_value = _get_core_context(rank=rank)
    index_dataset = _get_index_dataset(data_length)

    MyProcessorCLS = Mock()
    my_processor_instance = Mock()
    MyProcessorCLS.return_value = my_processor_instance

    torch_batch_process(
        dataset=index_dataset,
        batch_processor_cls=MyProcessorCLS,
        batch_size=batch_size,
        checkpoint_interval=checkpoint_interval,
    )

    assert my_processor_instance.process_batch.call_count == expected_process_batch_call_count
    # on_finish should only be called once
    assert my_processor_instance.on_finish.call_count == 1


@patch("determined.pytorch.experimental._torch_batch_process._load_state")
@patch("determined.pytorch.experimental._torch_batch_process.initialize_default_inference_context")
@patch("determined.get_cluster_info")
@patch("determined.pytorch.experimental._torch_batch_process._synchronize_and_checkpoint")
@patch("determined.pytorch.experimental._torch_batch_process._report_progress_to_master")
@patch("determined.pytorch.experimental._torch_batch_process._reduce_metrics")
@pytest.mark.parametrize(
    "data_length, batch_size, checkpoint_interval, rank, slot_ids, "
    "steps_completed, expected_process_batch_call_count",
    [  # total batches for worker (5), completed (2), iterate for 3 batches
        [50, 10, 1, 0, [0], 2, 3],
        # total batches for worker 0 (3), completed (2), iterate for 1 batches
        [50, 10, 1, 0, [0, 1], 2, 1],
        # total batches for worker 1 (2), completed (2), iterate for 0 batches
        [50, 10, 1, 1, [0, 1], 2, 0],
    ],
)
def test_torch_batch_process_times_process_batch_with_skip(
    mock_reduce_metrics,
    mock_report_progress_to_master,
    mock_synchronize_and_checkpoint,
    mock_get_cluster_info,
    mock_initialize_default_inference_context,
    mock_load_state,
    data_length,
    batch_size,
    checkpoint_interval,
    rank,
    slot_ids,
    steps_completed,
    expected_process_batch_call_count,
):
    mock_get_cluster_info.return_value = _get_det_info(
        slot_ids=slot_ids, latest_checkpoint="fake_latest_checkpoint"
    )
    mock_initialize_default_inference_context.return_value = _get_core_context(rank=rank)
    index_dataset = _get_index_dataset(data_length)

    MyProcessorCLS = Mock()
    my_processor_instance = Mock()
    MyProcessorCLS.return_value = my_processor_instance

    mock_load_state.return_value = {
        "steps_completed": steps_completed,
        "default_output_uuid": "abc",
    }

    torch_batch_process(
        dataset=index_dataset,
        batch_processor_cls=MyProcessorCLS,
        batch_size=batch_size,
        checkpoint_interval=checkpoint_interval,
    )

    assert my_processor_instance.process_batch.call_count == expected_process_batch_call_count
    # on_finish should only be called once
    assert my_processor_instance.on_finish.call_count == 1


@patch("determined.pytorch.experimental._torch_batch_process._load_state")
@patch("determined.pytorch.experimental._torch_batch_process.initialize_default_inference_context")
@patch("determined.get_cluster_info")
@patch("determined.pytorch.experimental._torch_batch_process._synchronize_and_checkpoint")
@patch("determined.pytorch.experimental._torch_batch_process._report_progress_to_master")
@patch("determined.pytorch.experimental._torch_batch_process._reduce_metrics")
@pytest.mark.parametrize(
    "data_length, batch_size, checkpoint_interval, rank, slot_ids, "
    "max_batches, expected_process_batch_call_count",
    [  # max_batches (2) < total batches (3) -> iterate for 2 batches
        [50, 10, 1, 0, [0, 1], 2, 2],
        # max_batches (10) > total batches (3) -> iterate for 3 batches
        [50, 10, 1, 0, [0, 1], 10, 3],
    ],
)
def test_torch_batch_process_max_batches(
    mock_reduce_metrics,
    mock_report_progress_to_master,
    mock_synchronize_and_checkpoint,
    mock_get_cluster_info,
    mock_initialize_default_inference_context,
    mock_load_state,
    data_length,
    batch_size,
    checkpoint_interval,
    rank,
    slot_ids,
    max_batches,
    expected_process_batch_call_count,
):
    mock_get_cluster_info.return_value = _get_det_info(slot_ids=slot_ids)
    mock_initialize_default_inference_context.return_value = _get_core_context(rank=rank)
    index_dataset = _get_index_dataset(data_length)

    MyProcessorCLS = Mock()
    my_processor_instance = Mock()
    MyProcessorCLS.return_value = my_processor_instance

    if max_batches > expected_process_batch_call_count:
        # Warning is raised when max_batches > total number of batches
        with pytest.warns(Warning):
            torch_batch_process(
                dataset=index_dataset,
                batch_processor_cls=MyProcessorCLS,
                batch_size=batch_size,
                checkpoint_interval=checkpoint_interval,
                max_batches=max_batches,
            )
    else:
        torch_batch_process(
            dataset=index_dataset,
            batch_processor_cls=MyProcessorCLS,
            batch_size=batch_size,
            checkpoint_interval=checkpoint_interval,
            max_batches=max_batches,
        )

    assert my_processor_instance.process_batch.call_count == expected_process_batch_call_count
    # on_finish should only be called once
    assert my_processor_instance.on_finish.call_count == 1


@patch("determined.pytorch.experimental._torch_batch_process.initialize_default_inference_context")
@patch("determined.get_cluster_info")
@patch("determined.pytorch.experimental._torch_batch_process._synchronize_and_checkpoint")
@patch("determined.pytorch.experimental._torch_batch_process._report_progress_to_master")
@patch("determined.pytorch.experimental._torch_batch_process._reduce_metrics")
def test_torch_batch_process_preemption(
    mock_reduce_metrics,
    mock_report_progress_to_master,
    mock_synchronize_and_checkpoint,
    mock_get_cluster_info,
    mock_initialize_default_inference_context,
):
    mock_get_cluster_info.return_value = _get_det_info()
    mock_initialize_default_inference_context.return_value = _get_core_context(
        should_preempt_results=[False, True]
    )
    index_dataset = _get_index_dataset(100)

    MyProcessorCLS = Mock()
    my_processor_instance = Mock()
    MyProcessorCLS.return_value = my_processor_instance

    torch_batch_process(
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


@patch("determined.pytorch.experimental._torch_batch_process.initialize_default_inference_context")
@patch("determined.get_cluster_info")
@patch("determined.pytorch.experimental._torch_batch_process._synchronize_and_checkpoint")
@patch("determined.pytorch.experimental._torch_batch_process._report_progress_to_master")
def test_torch_batch_process_reduce_metrics(
    mock_report_progress_to_master,
    mock_synchronize_and_checkpoint,
    mock_get_cluster_info,
    mock_initialize_default_inference_context,
    batch_processor_with_avg_metric_reducer,
):
    # Simulate a two worker run, and we are worker 0 (chief)
    # The entire dataset is [0, 1, 2, 3 , 4, 5, 6, 7, 8, 9]
    # With batch_size of 5,
    # - worker 0 works on [0, 1, 2, 3, 4], sum = 10
    # - worker 1 works on [5, 6, 7, 8, 9], sum = 35
    # The metric reducer simpler sum up all the values.
    # Therefore, by the end of the iteration, metric_reducer 0 would have self.sum == 10
    # metric_reducer 1 would have self.sum == 35
    mock_get_cluster_info.return_value = _get_det_info(slot_ids=[0, 1])
    index_dataset = _get_index_dataset(10)
    mock_core_context = _get_core_context(rank=0)
    mock_initialize_default_inference_context.return_value = mock_core_context

    mock_core_context.__enter__().distributed.gather.return_value = [[10], [35]]

    torch_batch_process(
        dataset=index_dataset,
        batch_processor_cls=batch_processor_with_avg_metric_reducer,
        batch_size=5,
        checkpoint_interval=1,
    )

    # Assert gather was first called with worker 0's sum amount: 10
    assert mock_core_context.__enter__().distributed.gather.call_args_list[0] == call([10])
    # Assert that we report metrics with the sum across workers: 45
    mock_core_context.__enter__().train.report_validation_metrics.assert_called_once_with(
        steps_completed=1,
        metrics={"sum_metric": 45},
    )
