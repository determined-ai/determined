from typing import Any, Dict, List, Optional, Tuple, Union, cast

import numpy as np
import torch

import determined as det
from determined import pytorch, util


def _combine_metrics_across_processes(
    context: det._core.DistributedContext, metrics: Dict[str, Any], num_batches: int
) -> Tuple[Optional[Dict[str, Any]], Optional[List[int]], Optional[int]]:
    # The chief receives the metric from every other training process.
    assert (
        context.size > 1
    ), "_combine_metrics_across_processes should only be called if context.distributed > 1"

    # all_args is a list of [(metrics, num_batches), ...] for each worker.
    all_args = context._zmq_gather((metrics, num_batches))

    if not context.rank == 0:
        return None, None, None

    # Remove items without keys in dictionary. These are from intermediate model parallel nodes.
    all_args = [a for a in cast(List, all_args) if len(a[0])]
    num_processes = len(all_args)

    # Reshape so e.g. all_metrics = [metrics, metrics, ...].
    all_metrics, all_num_batches = zip(*all_args)

    # convert all_metrics from List[Dict[str, Any]] to Dict[str, List[Any]].
    keys = all_metrics[0].keys()
    metrics_lists = {key: [m[key] for m in all_metrics] for key in keys}

    return metrics_lists, all_num_batches, num_processes


def _average_training_metrics(
    context: det._core.DistributedContext, per_batch_metrics: List[Dict[str, Any]]
) -> List[Dict[str, Any]]:
    """Average training metrics across GPUs"""
    # TODO (liam): decide whether overhead is acceptable to do this by default.
    # As part of this effort, we should benchmark zmq and torch.distributed communication
    # primitives to see which is faster.
    assert context.size > 1, "Can only average training metrics in multi-GPU training."
    metrics_timeseries = util._list_to_dict(per_batch_metrics)

    # Gather metrics across ranks onto rank 0 slot.
    # The combined_timeseries is: dict[metric_name] -> 2d-array.
    # A measurement is accessed via combined_timeseries[metric_name][process_idx][batch_idx].
    combined_timeseries, combined_num_batches, num_processes = _combine_metrics_across_processes(
        context, metrics_timeseries, num_batches=len(per_batch_metrics)
    )

    if context.rank == 0:
        # We can safely cast variables here because this is all happening on the chief, which
        # is where we gather metrics.
        combined_timeseries = cast(Dict[str, List[List[Any]]], combined_timeseries)
        combined_num_batches = cast(List[int], combined_num_batches)

        # If the value for a metric is a single-element array, the averaging process will
        # change that into just the element. We record what metrics are single-element arrays
        # so we can wrap them in an array later (for perfect compatibility with non-averaging
        # codepath).
        array_metrics = []
        for metric_name in combined_timeseries.keys():
            process_batches = combined_timeseries[metric_name]
            if isinstance(process_batches[0][0], np.ndarray):
                array_metrics.append(metric_name)

        num_batches = combined_num_batches[0]  # num_batches matches across data parallel ranks.
        averaged_metrics_timeseries = {}  # type: Dict[str, List]

        for metric_name in combined_timeseries.keys():
            averaged_metrics_timeseries[metric_name] = []
            for batch_idx in range(num_batches):
                batch = [
                    combined_timeseries[metric_name][process_idx][batch_idx]
                    for process_idx in range(cast(int, num_processes))
                ]

                np_batch = np.array(batch)
                batch_avg = np.mean(np_batch[np_batch != None])  # noqa: E711
                if metric_name in array_metrics:
                    batch_avg = np.array(batch_avg)
                averaged_metrics_timeseries[metric_name].append(batch_avg)
        per_batch_metrics = util._dict_to_list(averaged_metrics_timeseries)
    return per_batch_metrics


def _prepare_metrics_reducers(
    reducer: Optional[Union[pytorch.Reducer, Dict[str, Any]]], keys: Any
) -> Dict[str, pytorch.Reducer]:
    # Same as that for PyTorchTrialController.
    metrics_reducers = {}  # type: Dict[str, pytorch.Reducer]
    if isinstance(reducer, Dict):
        metrics_reducers = reducer
        if keys != metrics_reducers.keys():
            raise det.errors.InvalidExperimentException(
                "Please provide a single evaluation reducer or "
                "provide a reducer for every validation metric. "
                f"Expected keys: {keys}, provided keys: {metrics_reducers.keys()}.",
            )
    elif isinstance(reducer, pytorch.Reducer):
        for key in keys:
            metrics_reducers[key] = reducer

    for key in keys:
        if not isinstance(metrics_reducers[key], pytorch.Reducer):
            raise det.errors.InvalidExperimentException(
                "Please select `determined.pytorch.Reducer` for reducing validation metrics.",
            )

    return metrics_reducers


def _convert_metrics_to_numpy(metrics: Dict[str, Any]) -> Dict[str, Any]:
    for metric_name, metric_val in metrics.items():
        if isinstance(metric_val, torch.Tensor):
            metrics[metric_name] = metric_val.cpu().numpy()
    return metrics


def _reduce_metrics(
    context: det._core.DistributedContext,
    batch_metrics: List,
    keys: Any,
    metrics_reducers: Dict[str, pytorch.Reducer],
) -> Dict[str, Any]:
    metrics = {}
    if len(batch_metrics):
        metrics = {
            name: pytorch._reduce_metrics(
                reducer=metrics_reducers[name],
                metrics=np.stack([b[name] for b in batch_metrics], axis=0),
                num_batches=None,
            )
            for name in keys or []
        }

    if context.size > 1:
        # If using distributed training, combine metrics across all processes.
        # Only the chief process will receive all the metrics.
        num_batches = len(batch_metrics)
        combined_metrics, batches_per_process, _ = _combine_metrics_across_processes(
            context, metrics, num_batches
        )
        if context.rank == 0:
            # Only the chief collects all the metrics.
            combined_metrics = _convert_metrics_to_numpy(cast(Dict[str, Any], combined_metrics))
            metrics = {
                name: pytorch._reduce_metrics(
                    reducer=metrics_reducers[name],
                    metrics=combined_metrics[name],
                    num_batches=batches_per_process,
                )
                for name in keys or []
            }
        else:
            return {}

    return metrics
