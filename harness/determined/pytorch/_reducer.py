import abc
import enum
from typing import Any, Callable, List, Optional

import numpy as np

import determined_common.check as check


class Reducer(enum.Enum):
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


def _reduce_metrics(
    reducer: Reducer, metrics: np.array, num_batches: Optional[List[int]] = None
) -> np.float:
    if reducer == Reducer.AVG:
        if num_batches:
            check.check_eq(len(metrics), len(num_batches))
        return np.average(metrics, weights=num_batches)
    elif reducer == Reducer.SUM:
        return np.sum(metrics)
    elif reducer == Reducer.MAX:
        return np.max(metrics)
    elif reducer == Reducer.MIN:
        return np.min(metrics)
    else:
        raise NotImplementedError


class MetricReducer(metaclass=abc.ABCMeta):
    """
    Efficiently aggregating validation metrics during a multi-slot distributed trial is done in
    three steps:

    1. Gather all the values to be reduced during the reduction window (either a training or a
       validation workload).  In a multi-slot trial, this is done on each slot in parallel.

    2. Calculate the per-slot reduction.  This will return some intermediate value that each slot
       will contribute to the final metric calculation.  It can be as simple as a list of all the
       raw values from step 1, but reducing the intermediate value locally will distribute the
       final metric calculation more efficiently and will reduce network communication costs.

    3. Reduce the per-slot reduction values from Step 2 into a final metric.

    The MetricReducer API makes it possible for users to define a maximally efficient custom metric
    by exposing these steps to users:

       -  Step 1 is defined by the user; it is not part of the interface.  This flexibility
          gives the user full control when gathering individual values for reduction.

       -  Step 2 is the MetricReducer.per_slot_reduce() interface.

       -  Step 3 is the MetricReducer.cross_slot_reduce() interface.

       -  The MetricReducer.reset() interface allows for MetricReducer reuse across many train and
          validation workloads.

    Example implementation and usage:

    .. code:: python

        class MyAvgMetricReducer(pytorch.MetricReducer):
            def __init__(self):
                self.reset()

            def reset(self):
                self.sum = 0
                self.counts = 0

            # User-defined mechanism for collecting values throughout
            # training or validation. This update() mechanism demonstrates
            # a computationally- and memory-efficient way to store the values.
            def update(self, value):
                self.sum += sum(value)
                self.counts += 1

            def per_slot_reduce(self):
                # Because the chosen update() mechanism is so
                # efficient, this is basically a noop.
                return self.sum, self.counts

            def cross_slot_reduce(self, per_slot_metrics):
                # per_slot_metrics is a list of (sum, counts) tuples
                # returned by the self.pre_slot_reduce() on each slot
                sums, counts = zip(*per_slot_metrics)
                return sum(sums) / sum(counts)


        class MyPyTorchTrial(pytorch.PyTorchTrial):
            def __init__(self, context):
                # Register your custom reducer.
                self.my_avg = context.experimental.wrap_reducer(
                    MyAvgMetricReducer(), name="my_avg"
                )
                ...

            def train_batch(self, batch, epoch_idx, batch_idx):
                ...
                # You decide how/when you call update().
                self.my_avg.update(my_val)

                # The "my_avg" metric will be included in the final
                # metrics after the workload has completed; no need
                # to return it here.
                return {"loss": loss}


    See also: :meth:`determined.pytorch.PyTorchExperimentalContext.wrap_reducer`.
    """

    @abc.abstractmethod
    def reset(self) -> None:
        """
        Reset reducer state for another set of values.

        This will be called before any train or validation workload begins.
        """
        pass

    @abc.abstractmethod
    def per_slot_reduce(self) -> Any:
        """
        This will be called after all workers have finished (even when there is only one worker).

        It should return some picklable value that is meaningful for cross_slot_reduce.

        This will be called after any train or validation workload ends.
        """
        pass

    @abc.abstractmethod
    def cross_slot_reduce(self, per_slot_metrics: List) -> Any:
        """
        This will be called after per_slot_reduce has finished (even when there is only one worker).

        The per_slot_metrics will be a list containing the output of per_slot_reduce() from each
        worker.

        The return value should either be:
           -  A dict mapping string metric names to metric values, if the call to
              context.wrap_metric() omitted the `name` parameter, or
           -  A non-dict metric value if the call to context.wrap_metric() had name set to a string
              (an error will be raised if a dict-type metric is returned but name was set).

        This will be called after per_slot_reduce.
        """
        pass


class _SimpleReducer(MetricReducer):
    """
    Wrap a user-provided reduction function in a MetricReducer API.  It is not as efficient as the
    full MetricReducer API but simpler for users.
    """

    def __init__(self, fn: Callable) -> None:
        self.fn = fn
        self.reset()

    def reset(self) -> None:
        self.values = []  # type: List

    # The default way to interact with the simple API.
    def update(self, value: Any) -> None:
        self.values.append(value)

    def per_slot_reduce(self) -> Any:
        return self.values

    def cross_slot_reduce(self, per_slot_metrics: List) -> Any:
        flat_metrics = [item for sublist in per_slot_metrics for item in sublist]
        return self.fn(flat_metrics)
