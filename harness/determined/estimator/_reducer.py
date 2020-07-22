import abc
from typing import Any, Callable, List

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

                my_avg_prediction = context.experimental.make_metric(
                     metric=predictions, reducer=MyAvgMetricReducer(), numpy_dtype=np.float32
                )

                return tf.estimator.EstimatorSpec(
                    mode,
                    loss=loss,
                    eval_metric_ops={"my_avg_prediction": my_avg_prediction},
                )

    See also: :func:`determined.estimator.EstimatorExperimentalContext.make_metric`.
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


class _DistributedMetric(tf.keras.metrics.Metric):  # type: ignore
    """
    _DistributedMetric is a tf.keras Metric which should be compatible with tf.estimators and
    which uses a MetricReducer to calculate the metric.
    """

    def __init__(
        self,
        context: estimator.EstimatorExperimentalContext,
        metric: Any,
        reducer: MetricReducer,
        numpy_dtype: Any,
        name: str,
    ) -> None:
        super().__init__(name=name)
        # Don't conflict with tf.keras.metrics.Metric names.
        self._det_context = context
        self._det_reducer = reducer
        self._det_last_accumulate = None

        if isinstance(numpy_dtype, tf.dtypes.DType):
            raise TypeError(f"numpy_dtype parameter must not be a TensorFlow dtype: {numpy_dtype}")
        self._det_np_dtype = np.dtype(numpy_dtype)
        self._det_tf_dtype = tf.compat.v1.as_dtype(numpy_dtype)

        self.update_state(metric)

        def build_result_op() -> tf.Operation:
            return tf.compat.v1.py_func(self._py_result, [], self._det_tf_dtype)

        # Build the result_op once, even if it is requested multiple times.
        self._det_result_op = self._det_context._build_allgather_op(build_result_op)

    def _py_update_state(self, metric: Any) -> None:
        self._det_last_accumulate = self._det_reducer.accumulate(metric)

    def update_state(self, metric: Any) -> tf.Tensor:
        update_op = tf.compat.v1.py_func(self._py_update_state, [metric], [])
        # Explicitly require including update_op in the graph, but don't return it.
        with tf.compat.v1.control_dependencies([update_op]):
            return tf.zeros([])

    def _py_result(self) -> Any:
        allgathered = self._det_context.allgather_metrics(self._det_last_accumulate)
        result = self._det_reducer.cross_slot_reduce(allgathered)
        return np.array(result).astype(self._det_np_dtype)

    def result(self) -> tf.Operation:
        return self._det_result_op
