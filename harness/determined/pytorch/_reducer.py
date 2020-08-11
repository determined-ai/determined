import abc
import logging
from enum import Enum
from typing import Any, Callable, List, Type, Union

import numpy as np

ReducerType = Union["Reducer", Type["MetricReducer"], Callable]


class MetricReducer:
    """
    Efficiently aggregating validation metrics across a multi-slot distributed evaluation is done
    in two steps:

    1. Accumulate metrics from each batch on each slot. In the case of calculating a mean, this
       might mean keeping a running sum and a count of metrics received.

    2. Reduce metrics from each slot to calculate the final metric. In the case of calculating a
       mean, this might mean adding up the per-slot sums and dividing the result by the per-slot
       counts.

    Example implementation and usage:

    .. code:: python

        class MyAvgMetricReducer(pytorch.MetricReducer):
            def __init__(self):
               self.sum = 0
               self.counts = 0

            def accumulate(self, metric):
                self.sum += sum(metric)
                self.counts += 1
                return self.sum, self.counts

            def cross_slot_reduce(self, per_slot_metrics):
                # per_slot_metrics is a list of (sum, counts) tuples
                # returned by the final self.accumulate() on each slot
                sums, counts = zip(*per_slot_metrics)
                return sum(sums) / sum(counts)

        class MyPytorchTrial(pytorch.PyTorchTrial):
            ...
            def evaluate_batch(...):
                ...
                return {"my_averageable_metric": metric_tensor, ...}

            def evaluation_reducer(...):
                return {"my_averageable_metric": MyAvgMetricReducer, ...}

    See also: :meth:`determined.pytorch.PyTorchTrial.evaluation_reducer`.
    """

    @abc.abstractmethod
    def accumulate(self, metric: Any) -> Any:
        """
        accumulate is called for each batch in the evaluation dataset.  Batches will be distributed
        across slots, so accumulate will be called many times on each slot.

        accumulate should return the accumulated state.  After evaluation is complete, the final
        return value of accumulate will become an element of the per_slot_metrics argument to
        cross_slot_reduce.

        In the example of the calculating a distributed mean, accumulate might keep a running sum
        and a count of metrics received:

        .. code:: python

            def accumulate(self, metric):
                self.sum += metric
                self.count += 1
                return self.sum, self.count
        """
        pass

    @abc.abstractmethod
    def cross_slot_reduce(self, per_slot_metrics: List[Any]) -> Any:
        """
        cross_slot_reduce is called on the list of results from the final call to accumulate on
        each slot.  per_slot_metrics will be a list of length N, where N is the number of slots in
        the trial (or 1 in non-distributed training).  cross_slot_reduce must return the final
        metric.

        In the example of calculating a distributed mean, cross_slot_reduce might recieve a list of
        (sum, count) tuples and it would calculate the overall mean.

        .. code:: python

            def cross_slot_reduce(self, per_slot_metrics):
                sums, counts = zip(*per_slot_metrics)
                return np.array(sum(sums) / sum(counts))
        """
        pass


class _SimpleMetricReducer(MetricReducer):
    """_SimpleMetricReducer takes a one-step reducer function and converts it to a MetricReducer."""

    def __init__(self, reduce_fn: Callable[[List[Any]], Any]):
        self.updates = []  # type: List[Any]
        self.reduce_fn = reduce_fn

    def accumulate(self, metric: Any) -> List[Any]:
        self.updates.append(metric)
        return self.updates

    def cross_slot_reduce(self, per_slot_metrics: List[List[Any]]) -> Any:
        flat_metrics = [item for sublist in per_slot_metrics for item in sublist]
        return self.reduce_fn(flat_metrics)


class AvgMetricReducer(MetricReducer):
    def __init__(self) -> None:
        self.sum = None
        self.count = 0

    def accumulate(self, metric: Any) -> Any:
        if self.sum is None:
            self.sum = metric
        else:
            self.sum += metric
        self.count += 1
        return self.sum, self.count

    def cross_slot_reduce(self, per_slot_metrics: List[Any]) -> Any:
        sums, counts = zip(*per_slot_metrics)
        return np.sum(sums) / np.sum(counts)


class SumMetricReducer(MetricReducer):
    def __init__(self) -> None:
        self.sum = None

    def accumulate(self, metric: Any) -> Any:
        if self.sum is None:
            self.sum = metric
        else:
            self.sum += metric
        return self.sum

    def cross_slot_reduce(self, per_slot_metrics: List[Any]) -> Any:
        return np.sum(per_slot_metrics)


class MaxMetricReducer(MetricReducer):
    def __init__(self) -> None:
        self.max = None

    def accumulate(self, metric: Any) -> Any:
        if self.max is None:
            self.max = metric
        else:
            self.max = np.max([self.max, metric])
        return self.max

    def cross_slot_reduce(self, per_slot_metrics: List[Any]) -> Any:
        return np.max(per_slot_metrics)


class MinMetricReducer(MetricReducer):
    def __init__(self) -> None:
        self.min = None

    def accumulate(self, metric: Any) -> Any:
        if self.min is None:
            self.min = metric
        else:
            self.min = np.min([self.min, metric])
        return self.min

    def cross_slot_reduce(self, per_slot_metrics: List[Any]) -> Any:
        return np.min(per_slot_metrics)


class Reducer(Enum):
    """
    A ``Reducer`` defines a method for reducing (aggregating) evaluation
    metrics. See :meth:`determined.pytorch.PyTorchTrial.evaluation_reducer` for
    details.

    Attributes:
        AVG
        SUM
        MAX
        MIN
    """

    AVG = 1
    SUM = 2
    MAX = 3
    MIN = 4


def _make_reducer(r: ReducerType) -> MetricReducer:
    if isinstance(r, Reducer):
        replacement = {
            Reducer.AVG: AvgMetricReducer,
            Reducer.SUM: SumMetricReducer,
            Reducer.MAX: MaxMetricReducer,
            Reducer.MIN: MinMetricReducer,
        }[r]

        logging.warning(
            f"det.pytorch.{r} is deprecated, please return det.pytorch.{replacement.__name__} from "
            "PyTorchTrial.evaluation_reducer() instead.",
        )
        return replacement()  # type: ignore

    if isinstance(r, type) and issubclass(r, MetricReducer):
        return r()

    if callable(r):
        return _SimpleMetricReducer(r)

    raise AssertionError(
        "the reducer must be either a class implementing the MetricReducer or a function that can "
        "be used for one-shot metric reduction."
    )
