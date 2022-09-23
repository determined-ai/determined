from typing import Any, Dict, List, Optional, Tuple, Union, cast

import numpy as np
import torch

import determined as det
from determined import pytorch, util


def _process_combined_metrics_and_batches(
    combined_metrics_and_batches: List[Any],
) -> Tuple[Dict[str, Any], List[int]]:
    # Remove entries with 0 num batches. These are from ranks that do not report metrics.
    combined_metrics_and_batches = [a for a in combined_metrics_and_batches if a[1]]

    # Reshape so e.g. all_metrics = [metrics, metrics, ...].
    all_metrics, all_num_batches = zip(*combined_metrics_and_batches)

    # convert all_metrics from List[Dict[str, Any]] to Dict[str, List[Any]].
    keys = all_metrics[0].keys()
    metrics_lists = {key: [m[key] for m in all_metrics] for key in keys}

    return metrics_lists, list(all_num_batches)


def _average_training_metrics(
    combined_timeseries: Dict[str, Any], combined_num_batches: List[int]
) -> List[Dict[str, Any]]:
    """Average combined training metrics across GPUs"""
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
    num_processes = len(combined_num_batches)
    averaged_metrics_timeseries = {}  # type: Dict[str, List]

    for metric_name in combined_timeseries.keys():
        averaged_metrics_timeseries[metric_name] = []
        for batch_idx in range(num_batches):
            batch = [
                combined_timeseries[metric_name][process_idx][batch_idx]
                for process_idx in range(num_processes)
            ]

            np_batch = np.array(batch)
            batch_avg = np.mean(np_batch[np_batch != None])  # noqa: E711
            if metric_name in array_metrics:
                batch_avg = np.array(batch_avg)
            averaged_metrics_timeseries[metric_name].append(batch_avg)
    return util._dict_to_list(averaged_metrics_timeseries)


def _combine_metrics_across_processes(
    context: det.core.DistributedContext, metrics: Dict[str, Any], num_batches: int
) -> Tuple[Optional[Dict[str, Any]], Optional[List[int]]]:
    # The chief receives the metric from every other training process.
    assert (
        context.size > 1
    ), "_combine_metrics_across_processes should only be called if context.distributed > 1"

    # all_args is a list of [(metrics, num_batches), ...] for each worker.
    all_args = context.gather((metrics, num_batches))

    if not context.rank == 0:
        return None, None

    # Remove items without keys in dictionary. These are from intermediate model parallel nodes.
    assert all_args is not None, "gathered metrics should not be None"
    return _process_combined_metrics_and_batches(all_args)


def _combine_and_average_training_metrics(
    context: det.core.DistributedContext, per_batch_metrics: List[Dict[str, Any]]
) -> List[Dict[str, Any]]:
    assert context.size > 1, "Can only average training metrics in multi-GPU training."
    metrics_timeseries = util._list_to_dict(per_batch_metrics)

    # Gather metrics across ranks onto rank 0 slot.
    # The combined_timeseries is: dict[metric_name] -> 2d-array.
    # A measurement is accessed via combined_timeseries[metric_name][process_idx][batch_idx].
    combined_timeseries, combined_num_batches = _combine_metrics_across_processes(
        context, metrics_timeseries, num_batches=len(per_batch_metrics)
    )

    if context.rank == 0:
        # We can safely cast variables here because this is all happening on the chief, which
        # is where we gather metrics.
        combined_timeseries = cast(Dict[str, List[List[Any]]], combined_timeseries)
        combined_num_batches = cast(List[int], combined_num_batches)

        per_batch_metrics = _average_training_metrics(combined_timeseries, combined_num_batches)
    return per_batch_metrics


def _prepare_metrics_reducers(
    reducer: Union[pytorch.Reducer, Dict[str, Any]], keys: Any
) -> Dict[str, pytorch.Reducer]:
    metrics_reducers = {}  # type: Dict[str, pytorch.Reducer]
    if isinstance(reducer, Dict):
        metrics_reducers = reducer
        if keys != metrics_reducers.keys():
            raise det.errors.InvalidExperimentException(
                "Please provide a single evaluation reducer or "
                "provide a reducer for every validation metric. "
                f"Expected keys: {keys}, provided keys: {metrics_reducers.keys()}.",
            )
    else:
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
    context: det.core.DistributedContext,
    batch_metrics: List,
    keys: Any,
    metrics_reducers: Dict[str, pytorch.Reducer],
) -> Dict[str, Any]:
    metrics = {}
    if len(batch_metrics):
        metrics = {
            name: pytorch._simple_reduce_metrics(
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
        combined_metrics, batches_per_process = _combine_metrics_across_processes(
            context, metrics, num_batches
        )
        if context.rank == 0:
            # Only the chief collects all the metrics.
            assert combined_metrics is not None
            combined_metrics = _convert_metrics_to_numpy(combined_metrics)
            metrics = {
                name: pytorch._simple_reduce_metrics(
                    reducer=metrics_reducers[name],
                    metrics=combined_metrics[name],
                    num_batches=batches_per_process,
                )
                for name in keys or []
            }
        else:
            return {}

    return metrics
