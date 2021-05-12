import inspect
import logging
from typing import Any, Callable, Dict, List, Tuple, Union

import tensorflow as tf

import determined as det
from determined import _data_layer, estimator, horovod, util
from determined.common import check
from determined.horovod import hvd

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


class EstimatorContext(estimator._EstimatorReducerContext):
    """
    Base context class that contains runtime information for any Determined
    workflow that uses the ``tf.estimator`` API.
    """

    def __init__(
        self,
        env: det.EnvContext,
        hvd_config: horovod.HorovodContext,
        allgather_fn: Callable[[Any], List[Any]],
    ) -> None:
        super().__init__(allgather_fn)
        self.env = env
        self.hvd_config = hvd_config

        self.experimental = EstimatorExperimentalContext(
            env=env,
            hvd_config=hvd_config,
            parent=self,
        )

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

        # The signature of our horovod optimizer changed after we rebased onto 0.21.
        hvd_sig = inspect.signature(hvd.DistributedOptimizer)
        horovod_kwargs = {
            "compression": hvd.compression.Compression.fp16
            if use_compression
            else hvd.compression.Compression.none,
            "average_aggregated_gradients": self.hvd_config.average_aggregated_gradients,
        }
        if "aggregation_frequency" in hvd_sig.parameters:
            horovod_kwargs["aggregation_frequency"] = self.hvd_config.aggregation_frequency
        else:
            horovod_kwargs["backward_passes_per_step"] = self.hvd_config.aggregation_frequency

        optimizer = hvd.DistributedOptimizer(optimizer, **horovod_kwargs)
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
    def __init__(
        self,
        env: det.EnvContext,
        hvd_config: horovod.HorovodContext,
        rendezvous_info: det.RendezvousInfo,
    ) -> None:
        det.TrialContext.__init__(self, env, hvd_config, rendezvous_info)
        EstimatorContext.__init__(self, env, hvd_config, self.distributed._zmq_allgather)


class EstimatorNativeContext(det.NativeContext, EstimatorContext):
    def __init__(
        self,
        env: det.EnvContext,
        hvd_config: horovod.HorovodContext,
        rendezvous_info: det.RendezvousInfo,
    ) -> None:
        det.NativeContext.__init__(self, env, hvd_config, rendezvous_info)
        EstimatorContext.__init__(self, env, hvd_config, self.distributed._zmq_allgather)

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


class EstimatorExperimentalContext(_data_layer.DataLayerContext):
    """
    Context class that contains experimental runtime information and features
    for any Determined workflow that uses the ``tf.estimator`` API.

    ``EstimatorExperimentalContext`` extends ``EstimatorTrialContext`` under
    the ``context.experimental`` namespace.
    """

    def __init__(
        self, env: det.EnvContext, hvd_config: horovod.HorovodContext, parent: EstimatorContext
    ) -> None:
        super().__init__(env=env, hvd_config=hvd_config)
        self._parent = parent

    @util.deprecated(
        "context.experimental.allgather_metrics() is deprecated since 0.15.2 and will be removed "
        "in a future version.  It is not intended to have a replacement; please contact Determined "
        "if you depend on this experimental method."
    )
    def allgather_metrics(self, metrics: Any) -> List:
        return self._parent._allgather_fn(metrics)

    @util.deprecated(
        "context.experimental.make_metric() is deprecated since 0.15.2 and will be removed in a "
        "future version; use context.make_metric() directly."
    )
    def make_metric(
        self,
        metric: Any,
        reducer: Union[Callable[[List[Any]], Any], "estimator.MetricReducer"],
        numpy_dtype: Any,
    ) -> Tuple[tf.Operation, tf.Operation]:
        return self._parent.make_metric(metric, reducer, numpy_dtype)
