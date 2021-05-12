import abc
from typing import Any, Callable, List, Sequence, Tuple, Union

import numpy as np
import tensorflow as tf

from determined import estimator


class MetricReducer:
    """
    Efficiently aggregating validation metrics across a multi-slot distributed evaluation is done
    in two steps:

    #. Accumulate metrics from each batch on each slot. In the case of calculating a mean, this
       might mean keeping a running sum and a count of metrics received.

    #. Reduce metrics from each slot to calculate the final metric. In the case of calculating a
       mean, this might mean adding up the per-slot sums and dividing the result by the per-slot
       counts.

    Example implementation and usage:

    .. code:: python

        class MyAvgMetricReducer(estimator.MetricReducer):
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

        def my_estimator_model_function(features, labels, mode):
            ...
            if mode == tf.estimator.ModeKeys.EVAL:

                my_avg_prediction = context.make_metric(
                     metric=predictions, reducer=MyAvgMetricReducer(), numpy_dtype=np.float32
                )

                return tf.estimator.EstimatorSpec(
                    mode,
                    loss=loss,
                    eval_metric_ops={"my_avg_prediction": my_avg_prediction},
                )

    See also: :func:`context.make_metric() <determined.estimator.EstimatorContext.make_metric>`.
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

    def reset(self) -> None:
        self.updates = []

    def accumulate(self, metric: Any) -> List[Any]:
        self.updates.append(metric)
        return self.updates

    def cross_slot_reduce(self, per_slot_metrics: List[List[Any]]) -> Any:
        flat_metrics = [item for sublist in per_slot_metrics for item in sublist]
        return self.reduce_fn(flat_metrics)


class _EstimatorReducerContext:
    """
    Context class that contains only the reducer-related information.
    """

    def __init__(self, allgather_fn: Callable[[Any], List[Any]]) -> None:
        self._allgather_fn = allgather_fn
        # allgather is not parallelizable, so we have to strictly order how they are placed in the
        # graph via tf.control_dependencies().
        self._allgather_ops = []  # type: List[tf.Operation]

    def _set_allgather_fn(self, fn: Callable[[Any], List]) -> None:
        self._allgather_fn = fn

    def _build_allgather_op(self, build_op_fn: Callable[[], tf.Operation]) -> tf.Operation:
        """Build an op that uses allgather in a way that is safely sequentialized."""

        with tf.compat.v1.control_dependencies(self._allgather_ops):
            new_op = build_op_fn()
        self._allgather_ops.append(new_op)
        return new_op

    def _reset_allgather_ops(self) -> None:
        """Every Estimator evaluation happens on a clean graph, so forget the old operations."""
        self._allgather_ops = []

    def make_metric(
        self,
        metric: Any,
        reducer: Union[Callable[[List[Any]], Any], "estimator.MetricReducer"],
        numpy_dtype: Any,
    ) -> Tuple[tf.Operation, tf.Operation]:
        """
        Return an estimator-compatible validation metric which will be calculated properly, even
        during distributed evaluation.

        During distributed evaluation, many types of metrics calculated via ``tf.metrics`` or
        ``tf.keras.metrics`` cannot be aggregated properly from the per-slot final metrics
        calculated by each separate Estimator replica. One example is ``tf.metrics.auc``, where
        the ROC AUC calculated over predictions and labels from a full dataset cannot be derived
        from the individual ROC AUC metrics evaluated over several shards of a dataset.

        Determined solves this problem by offering customizable metrics which are
        Estimator-compatible.  For example, ROC AUC could be properly calculated during distributed
        evaluation by calling ``sklearn.metrics.roc_auc_score`` in a custom ``reducer`` function
        passed to ``make_metric``.

        The ``metric`` input can be a tensor, a list of tensors, or a dictionary of tensors.
        Nested structures are not supported.

        The ``reducer`` should be either a single function that can calculate the metric from a
        list of the per-batch values of ``metric``, or it can be an instance of a
        :class:`det.estimator.MetricReducer<determined.estimator.MetricReducer>`.

        The ``numpy_dtype`` must be a numpy dtype.  It is used internally to determine the output
        type of the TensorFlow ``py_func`` to report the final metric result to the Estimator API.
        The format of ``numpy_dtype`` should be anything that ``np.dtype()`` accepts.

        The primary motivation for passing a function as the reducer is simplicity. Metrics from
        all batches will be buffered in memory and passed over the network where they will be
        reduced all at once. This introduces some overhead, but it is likely unnoticeable for
        scalar metrics or on validation datasets of small or medium size. This single function
        strategy may also be desirable for quick prototyping or for calculating metrics that are
        difficult or impossible to calculate incrementally.

        The primary motivation for passing a ``det.estimator.MetricsReducer`` as the reducer is
        performance. ``det.estimator.MetricsReducer`` allows the user to incrementally calculate
        the partial metric on each slot, taking advantage of distributed computation, minimizing
        memory usage, and minimizing the network communication before the final
        ``cross_slot_reduce`` operation.

        Evaluation performance may be improved by precomputing as much as possible in the graph so
        that less computation on the ``metric`` value is required within the reducer.

        Example usage where ``reducer`` is a function:

        .. code-block:: python

           def my_mean_reducer(all_batch_metrics):
               # Use hstack in case not all batches are equal length.
               return np.mean(np.hstack(all_batch_metrics))

           def my_estimator_model_function(features, labels, mode):
               ...
               if mode == tf.estimator.ModeKeys.EVAL:

                   my_avg_prediction = context.make_metric(
                        metric=predictions, reducer=my_mean_reducer, numpy_dtype=np.float32
                   )

                   return tf.estimator.EstimatorSpec(
                       mode,
                       loss=loss,
                       eval_metric_ops={"my_avg_prediction": my_avg_prediction},
                   )
        """
        if isinstance(reducer, estimator.MetricReducer):
            return estimator._DistributedMetricMaker(
                self, metric, reducer, numpy_dtype
            ).make_metric()

        simple_reducer = estimator._SimpleMetricReducer(reducer)
        return estimator._DistributedMetricMaker(
            self, metric, simple_reducer, numpy_dtype
        ).make_metric()


def _deconstruct_metric(metric: Any) -> Tuple[Sequence, Any]:
    """
    Break down lists and dictionaries into a list of tensors that can each be passed through the
    graph as individual inputs to a tf.compat.v1.py_func. A py_func can take arbitrary numbers of
    inputs, but if you try to pass many inputs as e.g. a single dictionary, it will attempt to
    convert that dictionary to a single tensor, and that will fail.
    """
    if isinstance(metric, (tf.Tensor, tf.Operation)):
        return [metric], (tf.Tensor, None)

    if isinstance(metric, (list, tuple)):
        for m in metric:
            if not isinstance(m, (tf.Tensor, tf.Operation)):
                raise TypeError(
                    "list-type metric parameters must be a flat list of tf.Tensors but found "
                    f"element of type {type(m)}"
                )
        return metric, (list, None)

    if isinstance(metric, dict):
        for k, v in metric.items():
            if not isinstance(k, str):
                raise TypeError(
                    "dict-type metric parameters must be a flat list mapping strings to "
                    f"tf.Tensors but found key of type {type(k)}"
                )
            if not isinstance(v, (tf.Tensor, tf.Operation)):
                raise TypeError(
                    "dict-type metric parameters must be a flat list mapping strings to "
                    f"tf.Tensors but found value of type {type(v)}"
                )
        keys, args = zip(*metric.items())
        return args, (dict, keys)

    else:
        # Try to convert the arbitrary input to a constant Tensor.
        try:
            const_metric = tf.compat.v1.constant(metric)
        except TypeError:
            raise TypeError(
                "metric parameter must be a tf.Tensor, a list of tf.Tensors, "
                f"or a dict mapping strings to tf.Tensors, not {type(metric)}"
            )
        return const_metric, (tf.Tensor, None)


def _reconstruct_metric(args: Sequence, reconstruct_info: Any) -> Any:
    """Reconstruct lists or dictionaries after passing them through the graph."""
    metric_type, update_keys = reconstruct_info
    if metric_type == tf.Tensor:
        return args[0]
    if metric_type == list:
        return args
    if metric_type == dict:
        return dict(zip(update_keys, args))
    raise AssertionError(f"invalid metric_type: {metric_type}")


class _DistributedMetricMaker:
    """
    _DistributedMetricMaker.make_metric() returns a tf.metrics-style tuple of (value_op, update_op).
    The value_op is read once after all evaluation is completed, which is where we do the allgather
    and call the user's cross_slot_reduce to calculate the distributed metric.
    """

    def __init__(
        self,
        context: _EstimatorReducerContext,
        metric: Any,
        reducer: MetricReducer,
        numpy_dtype: Any,
    ) -> None:
        self.context = context
        self.reducer = reducer

        # Determine how we are going to pass the metric parameter through the graph, so we can
        # reconstruct it for the user inside of a py_func.
        self.update_args, self.reconstruct_info = _deconstruct_metric(metric)
        self.np_dtype = np.dtype(numpy_dtype)
        self.tf_dtype = tf.compat.v1.as_dtype(numpy_dtype)

        self.last_accumulate = None

    def _update(self, *args: List) -> None:
        # Reconstruct the format of the input that the user gave us.
        metric = _reconstruct_metric(args, self.reconstruct_info)
        self.last_accumulate = self.reducer.accumulate(metric)

    def _update_op(self) -> tf.Operation:
        return tf.compat.v1.py_func(self._update, self.update_args, [])

    def _value(self) -> Any:
        allgathered = self.context._allgather_fn(self.last_accumulate)
        value = self.reducer.cross_slot_reduce(allgathered)
        return np.array(value).astype(self.np_dtype)

    def _value_op(self) -> tf.Operation:
        return tf.compat.v1.py_func(self._value, [], self.tf_dtype)

    def make_metric(self) -> Tuple[tf.Operation, tf.Operation]:
        value_op = self.context._build_allgather_op(self._value_op)
        update_op = self._update_op()

        return value_op, update_op
