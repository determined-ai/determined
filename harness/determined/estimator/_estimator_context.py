import logging
from typing import Any, Callable, Dict, List, Tuple, Union

import tensorflow as tf

import determined as det
from determined import _data_layer, estimator, horovod
from determined.horovod import hvd
from determined_common import check

"""
The normal path to create a model usually needs the use of tensorflow pre-made optimizer and
dataset objects. However, the path to create a model for Horovod is different. Some Horovod
functions needs to be called to pre-process the native optimizer and dataset. Then the processed
optimizer and dataset can be passed to instantiate the estimator.

The user interface is designed as that users need to wrap the native optimizer and dataset object
by using the functions wrap_optimizer() and wrap_dataset(). These functions allow Determined to
seamlessly distribute training across multiple workers when distributed training is configured.
"""

# The optional interface for specifying serving input receiver functions to
# export SavedModels expects the following function type.
ServingInputReceiverFn = Callable[
    ...,
    Union[tf.estimator.export.ServingInputReceiver, tf.estimator.export.TensorServingInputReceiver],
]


class EstimatorContext:
    """
    Base context class that contains runtime information for any Determined
    workflow that uses the ``tf.estimator`` API.
    """

    def __init__(self, env: det.EnvContext, hvd_config: horovod.HorovodContext) -> None:
        self.env = env
        self.hvd_config = hvd_config

        self.experimental = EstimatorExperimentalContext(env=env, hvd_config=hvd_config)

        self.optimizer_initialized = False
        self.dataset_initialized = False
        logging.debug(f"Initialized EstimatorContext with config: {self.hvd_config}.")

    def wrap_optimizer(self, optimizer: Any) -> Any:
        """
        This should be used to wrap optimizer objects immediately after they have
        been created. Users should use the output of this wrapper as the new instance
        of their optimizer. For example, if users create their optimizer within
        ``build_estimator()``, they should call ``optimizer = wrap_optimizer(optimzer)``
        prior to passing the optimizer into their Estimator.
        """
        if not self.env.managed_training:
            return optimizer

        self.optimizer_initialized = True
        if not self.hvd_config.use:
            return optimizer

        check.check_false(
            isinstance(optimizer, str),
            "Please specify an optimizer object instead of using a string name.",
        )

        hvd.require_horovod_type("tensorflow", "EstimatorContext.wrap_optimizer was called.")
        use_compression = self.hvd_config.fp16_compression
        optimizer = hvd.DistributedOptimizer(
            optimizer,
            compression=hvd.compression.Compression.fp16
            if use_compression
            else hvd.compression.Compression.none,
            aggregation_frequency=self.hvd_config.aggregation_frequency,
            average_aggregated_gradients=self.hvd_config.average_aggregated_gradients,
        )
        logging.debug("Initialized optimizer for distributed and optimized parallel training.")
        return optimizer

    def wrap_dataset(self, dataset: Any, shard_dataset: bool = True) -> Any:
        """
        This should be used to wrap ``tf.data.Dataset`` objects immediately after
        they have been created. Users should use the output of this wrapper as the
        new instance of their dataset. If users create multiple datasets (e.g., one
        for training and one for testing), users should wrap each dataset
        independently. E.g., If users instantiate their training dataset within
        ``build_train_spec()``, they should call ``dataset = wrap_dataset(dataset)``
        prior to passing it into ``tf.estimator.TrainSpec``.

        Args:
            dataset: tf.data.Dataset
            shard_dataset:
                When performing multi-slot (distributed) training, this
                controls whether the dataset is sharded so that each training process
                (one per slot) sees unique data. If set to False, users must manually
                configure each process to use unique data.

        """
        if not self.env.managed_training:
            return dataset

        hvd.require_horovod_type("tensorflow", "EstimatorContext.wrap_dataset was called.")

        self.dataset_initialized = True
        if not self.hvd_config.use or not shard_dataset:
            if self.hvd_config and not shard_dataset:
                logging.info("Dataset sharding skipped.")
            return dataset

        dataset = dataset.shard(hvd.size(), hvd.rank())
        logging.debug(f"Sharded dataset to index {hvd.rank()} of {hvd.size()}.")
        return dataset


class EstimatorTrialContext(det.TrialContext, EstimatorContext):
    def __init__(self, env: det.EnvContext, hvd_config: horovod.HorovodContext) -> None:
        det.TrialContext.__init__(self, env, hvd_config)
        EstimatorContext.__init__(self, env, hvd_config)


class EstimatorNativeContext(det.NativeContext, EstimatorContext):
    def __init__(self, env: det.EnvContext, hvd_config: horovod.HorovodContext) -> None:
        det.NativeContext.__init__(self, env, hvd_config)
        EstimatorContext.__init__(self, env, hvd_config)

        # TODO(DET-1931): Figure out the right interface to set it.
        self.serving_input_receiver_fns = {}  # type: Dict[str, ServingInputReceiverFn]

    def train_and_evaluate(
        self,
        estimator: tf.estimator.Estimator,
        train_spec: tf.estimator.TrainSpec,
        eval_spec: tf.estimator.EvalSpec,
    ) -> Any:
        self.estimator = estimator
        self.train_spec = train_spec
        self.eval_spec = eval_spec

        if self._train_fn:
            self._train_fn()


def default_allgather_fn(metrics: Any) -> List:
    """
    A noop allgather implementation to ensure that custom reducers work outside of Determined.
    """
    return [metrics]


class EstimatorExperimentalContext(_data_layer.DataLayerContext):
    """
    Context class that contains experimental runtime information and features
    for any Determined workflow that uses the ``tf.estimator`` API.

    ``EstimatorExperimentalContext`` extends ``EstimatorTrialContext`` under
    the ``context.experimental`` namespace.
    """

    def __init__(self, env: det.EnvContext, hvd_config: horovod.HorovodContext) -> None:
        super().__init__(env=env, hvd_config=hvd_config)
        self._allgather_fn = default_allgather_fn  # type: Callable[[Any], List]
        # allgather is not parallelizable, so we have to strictly order how they are placed in the
        # graph via tf.control_dependencies().
        self._allgather_ops = []  # type: List[tf.Operation]

    def _set_allgather_fn(self, fn: Callable[[Any], List]) -> None:
        self._allgather_fn = fn

    def allgather_metrics(self, metrics: Any) -> List:
        return self._allgather_fn(metrics)

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

        The ``numpy_dtype`` must be a numpy dtype.  It is used internally to determined the output
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

                   my_avg_prediction = context.experimental.make_metric(
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
