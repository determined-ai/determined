import abc
import enum
import itertools
from typing import Any, Callable, Dict, List, Optional, TypeVar, Union, overload

import numpy as np


class Reducer(enum.Enum):
    """
    A ``Reducer`` defines a method for reducing (aggregating) evaluation
    metrics. See :meth:`~determined.pytorch.PyTorchTrial.evaluation_reducer` for
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


def _simple_reduce_metrics(
    reducer: Reducer, metrics: np.ndarray, num_batches: Optional[List[int]] = None
) -> np.float64:
    if reducer == Reducer.AVG:
        if num_batches:
            if len(metrics) != len(num_batches):
                raise RuntimeError(
                    "Lengths of metrics and num_batches are not equal: "
                    f"{len(metrics)} != {len(num_batches)}."
                )
        return np.average(metrics, weights=num_batches)  # type: ignore
    elif reducer == Reducer.SUM:
        return np.sum(metrics)  # type: ignore
    elif reducer == Reducer.MAX:
        return np.max(metrics)  # type: ignore
    elif reducer == Reducer.MIN:
        return np.min(metrics)  # type: ignore
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
                self.my_avg = context.wrap_reducer(
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
              context.wrap_reducer() omitted the `name` parameter, or
           -  A non-dict metric value if the call to context.wrap_reducer() had name set to a string
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

    class _NotMetric:
        pass

    NOT_METRIC = _NotMetric()

    def cross_slot_reduce(self, per_slot_metrics: List) -> Any:
        """
        Call the reducer function on metrics in their original order.

        Interleave metrics from different slots as we flatten
        the list of metrics to undo the effect of sharding.
        Note: this will only reconstruct the original order if the user
        calls ``update()`` once per batch
        """
        interleaved = itertools.zip_longest(*per_slot_metrics, fillvalue=_SimpleReducer.NOT_METRIC)
        flat_metrics = [
            m for sublist in interleaved for m in sublist if m is not _SimpleReducer.NOT_METRIC
        ]

        return self.fn(flat_metrics)


def default_allgather_fn(metrics: Any) -> List:
    """
    A noop allgather implementation to ensure that custom reducers work outside Determined.
    """
    return [metrics]


class _WrappedReducer:
    def __init__(
        self,
        reducer: MetricReducer,
        name: Optional[str],
        for_training: bool,
        for_validation: bool,
    ) -> None:
        self.reducer = reducer
        self.name = name
        self.for_training = for_training
        self.for_validation = for_validation

    def reset(self) -> None:
        """
        Call reducer.reset() with a more useful stacktrace.

        Normally, when we call a user's MetricReducer's methods, the stack trace does not clearly
        identify which reducer actually failed, since the information about "which reducer failed"
        is not stored on the stack (it's stored in an iterator).

        This can make debugging custom reducers much more difficult, so we add information to the
        stack trace for a more pleasant user experience.
        """
        try:
            return self.reducer.reset()
        except Exception as e:
            raise ValueError(
                f'reducer of type {type(self.reducer).__name__} for name="{self.name}" '
                "failed in a call to reset()"
            ) from e

    def per_slot_reduce(self) -> Any:
        try:
            return self.reducer.per_slot_reduce()
        except Exception as e:
            raise ValueError(
                f'reducer of type {type(self.reducer).__name__} for name="{self.name}" '
                "failed in a call to per_slot_reduce()"
            ) from e

    def cross_slot_reduce(self, per_slot_metrics: List) -> Any:
        try:
            result = self.reducer.cross_slot_reduce(per_slot_metrics)
        except Exception as e:
            raise ValueError(
                f'reducer of type {type(self.reducer).__name__} for name="{self.name}" '
                f"failed in a call to cross_slot_reduce() with per_slot_metrics={per_slot_metrics}"
            ) from e
        return result


T = TypeVar("T", bound=MetricReducer)


class _PyTorchReducerContext:
    """
    _PyTorchReducerContext is a component of the PyTorchTrialContext which does not interact with
    the rest of the PyTorchTrialContext.  It has been separated into its own object to keep the
    size of PyTorchTrialContext down, and because there may be a future where we can create a
    _PyTorchReducerContext as a standalone feature for users of the Core API... though in that
    case we would rename it with a public name first.
    """

    def __init__(self, allgather_fn: Callable[[Any], List[Any]] = default_allgather_fn) -> None:
        self._wrapped_reducers = []  # type: List[_WrappedReducer]
        self._allgather_fn = allgather_fn

    def reset_reducers(self) -> None:
        for wrapped in self._wrapped_reducers:
            wrapped.reset()

    # When wrap_reducer is passed a Callable, it returns a _SimpleReducer.
    @overload
    def wrap_reducer(
        self,
        reducer: Callable,
        name: Optional[str] = None,
        for_training: bool = True,
        for_validation: bool = True,
    ) -> _SimpleReducer:
        ...

    # When wrap_reducer is passed a MetricReducer, it returns the same MetricReducer.
    @overload
    def wrap_reducer(
        self,
        reducer: T,
        name: Optional[str] = None,
        for_training: bool = True,
        for_validation: bool = True,
    ) -> T:
        ...

    def wrap_reducer(
        self,
        reducer: Union[Callable, MetricReducer],
        name: Optional[str] = None,
        for_training: bool = True,
        for_validation: bool = True,
    ) -> MetricReducer:
        """
        Register a custom reducer that will calculate a metric properly, even with distributed
        training.

        During distributed training and evaluation, many types of metrics must be calculated
        globally, rather than calculating the metric on each shard of the dataset and averaged or
        summed.  For example, an accurate ROC AUC for dataset cannot be derived from the individual
        ROC AUC metrics calculated on by each worker.

        Determined solves this problem by offering fully customizable metric reducers which are
        distributed-aware.  These are registered by calling ``context.wrap_reducer()``
        and are updated by the user during ``train_batch()`` or ``evaluate_batch()``.

        Arguments:
            reducer (Union[Callable, pytorch.MetricReducer]):
                Either a reducer function or a pytorch.MetricReducer.  See below for more details.
            name: (Optional[str] = None):
                Either a string name to associate with the metric returned by the reducer, or
                ``None`` to indicate the metric will return a dict mapping string names to metric
                values.  This allows for a single reducer to return many metrics, such as for a
                per-class mean IOU calculation.  Note that if name is a string, the returned
                metric must NOT be a dict-type metric.
            for_training: (bool = True):
                Indicate that the ``reducer`` should be used for training workloads.
            for_validation: (bool = True):
                Indicate that the ``reducer`` should be used for validation workloads.

        Return Value:
            pytorch.MetricReducer:
                If ``reducer`` was a function, the returned ``MetricReducer`` will have a single
                user-facing method like ``def update(value: Any) -> None`` that you should call
                during ``train_batch`` or ``evaluate_batch``.  Otherwise, the return value will
                just be the ``reducer`` that was passed in.

        **Reducer functions: the simple API**

        If the ``reducer`` parameter is a function, it must have the following properties:

           -  It accepts a single parameter, which will be a flat list of all inputs the users
              pass when they call ``.update()`` on the object returned by ``wrap_reducer()``.
              See the code example below for more details.
           -  It returns either a single (non-dict) metric or a dictionary mapping names to
              metrics, as described above.

        The primary motivation for passing a function as the reducer is simplicity. Metrics from
        all batches will be buffered in memory and passed over the network before they are reduced
        all at once. This introduces some overhead, but it is likely unnoticeable for scalar
        metrics or on validation datasets of small or medium size.  This single function strategy
        can be useful for quick prototyping or for calculating metrics that are difficult
        or impossible to calculate incrementally.

        For example, ROC AUC could be properly calculated by passing a small wrapper function
        calling ``sklearn.metrics.roc_auc_score``:

        .. code-block:: python

           # Custom reducer function.
           def roc_auc_reducer(values):
               # values will be a flat list of all inputs to
               # .update(), which in this code example are
               # tuples of (y_true, y_score).  We reshape
               # that list into two separate lists:
               y_trues, y_scores = zip(*values)

               # Then we return a metric value:
               return sklearn.metrics.roc_auc_score(
                   np.array(y_trues), np.array(y_scores)
               )

           class MyPyTorchTrial(PyTorchTrial):
               def __init__(self, context):
                   self.roc_auc = context.wrap_reducer(
                       roc_auc_reducer, name="roc_auc"
                   )
                   ...

               def evaluate_batch(self, batch):
                   ...
                   # Function-based reducers are updated with .update().
                   # The roc_auc_reducer function will get a list of all
                   # inputs that we pass in here:
                   self.roc_auc.update((y_true, y_score))

                   # The "roc_auc" metric will be included in the final
                   # metrics after the workload has completed; no need
                   # to return it here.  If that is your only metric,
                   # just return an empty dict.
                   return {}

        **MetricReducer objects: the advanced API**

        The primary motivation for passing a ``det.pytorch.MetricReducer`` as the reducer is
        performance. ``det.pytorch.MetricReducer`` allows the user more control in how values are
        stored and exposes a ``per_slot_reduce()`` call which lets users minimize the cost of the
        network communication before the final ``cross_slot_reduce()``.

        An additional reason for using the ``det.pytorch.MetricReducer`` is for flexibility of the
        update mechanism, which is completely user-defined when subclassing ``MetricReducer``.

        For the full details and a code example, see: :class:`~determined.pytorch.MetricReducer`.
        """

        # Detect double-wrapped reducers.
        if reducer in (wrapped.reducer for wrapped in self._wrapped_reducers):  # type: ignore
            raise AssertionError(
                f"Detected the same reducer of type {type(reducer).__name__} in "
                "context.wrap_reducer(), please avoid calling wrap_reducer() on the "
                "same reducer object twice."
            )

        if not isinstance(reducer, MetricReducer):
            if callable(reducer):
                reducer = _SimpleReducer(reducer)
            else:
                raise AssertionError(
                    f"Detected invalid reducer in wrap_reducer() of type {type(reducer).__name__}. "
                    "Reducers must either be a function or a subclass of pytorch.MetricReducer."
                )

        wrapped = _WrappedReducer(reducer, name, for_training, for_validation)

        self._wrapped_reducers.append(wrapped)
        return reducer

    def reduce_metrics(self, for_training: bool) -> Dict[str, Any]:
        # Only deal with reducers marked for this type of workload.
        reducables = [
            wrapped
            for wrapped in self._wrapped_reducers
            if (for_training and wrapped.for_training)
            or (not for_training and wrapped.for_validation)
        ]

        if not reducables:
            return {}

        metrics = {}

        gatherables = [wrapped.per_slot_reduce() for wrapped in reducables]

        # Do one allgather for all metrics to improve .
        gathered = self._allgather_fn(gatherables)

        # Reshape list from per-slot-list-of-all-metrics to a per-metric-list-of-all-slots.
        all_per_slot_metrics = zip(*gathered)

        for wrapped, per_slot_metrics in zip(reducables, all_per_slot_metrics):
            reduced = wrapped.cross_slot_reduce(per_slot_metrics)
            if wrapped.name is None:
                if not isinstance(reduced, dict):
                    # If wrap_reducer() had name=None, the reduced metric must be a dict.
                    if isinstance(wrapped.reducer, _SimpleReducer):
                        raise AssertionError(
                            f"The custom reduction function {wrapped.reducer.fn.__name__}() was "
                            "wrapped by a call to wrap_reducer() with name=None but it did not "
                            f"return a dict (it returned a {type(reduced).__name__}).  Please set "
                            "name if you wish to return a single metric or return a dictionary "
                            "mapping names to metrics if you with to return multiple metrics from "
                            "a single reducer."
                        )
                    raise AssertionError(
                        f"The custom reduction MetricReducer {type(wrapped.reducer).__name__} was "
                        "wrapped by a call to wrap_reducer() with name=None but it did not return "
                        f"return a dict (it returned a {type(reduced).__name__}).  Please "
                        "set name if you wish to return a single metric or return a dictionary "
                        "mapping names to metrics if you with to return multiple metrics from "
                        "a single reducer."
                    )
                metrics.update(reduced)
            else:
                if isinstance(reduced, dict):
                    # Disallow users from returning dict-like metrics if they provided a name,
                    # because that is just too common of an error.  In the future, if we recursed
                    # into dictionary-type metrics and rendered them in the webui, then this we
                    # could allow this case because it would be trivially easy for users to see
                    # their mistake and fix it.
                    if isinstance(wrapped.reducer, _SimpleReducer):
                        raise AssertionError(
                            f"The custom reduction function {wrapped.reducer.fn.__name__}() was "
                            "wrapped by a call to wrap_reducer() with name set but it returned a "
                            "dict anyway.  Please leave name=None (the default value) if you wish "
                            "to return a dict of multiple metrics or return a single metric (not "
                            "a dict) if you wish to return a single named metric."
                        )
                    raise AssertionError(
                        f"The custom reduction MetricReducer {type(wrapped.reducer).__name__} was "
                        "wrapped by a call to wrap_reducer() with name set but it returned a "
                        "dict anyway.  Please leave name=None (the default value) if you wish "
                        "to return a dict of multiple metrics or return a single metric (not "
                        "a dict) if you wish to return a single named metric."
                    )
                metrics[wrapped.name] = reduced

        return metrics
