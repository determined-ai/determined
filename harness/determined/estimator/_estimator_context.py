import inspect
import logging
from typing import Any, Callable, Union, cast

import tensorflow as tf

import determined as det
from determined import _data_layer, core, estimator, util
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


class EstimatorTrialContext(det.TrialContext, estimator._EstimatorReducerContext):
    """
    Base context class that contains runtime information for any Determined
    workflow that uses the ``tf.estimator`` API.
    """

    def __init__(self, *arg: Any, **kwarg: Any) -> None:
        det.TrialContext.__init__(self, *arg, **kwarg)
        estimator._EstimatorReducerContext.__init__(self, self.distributed.allgather)

        self._per_slot_batch_size, self._global_batch_size = util.calculate_batch_sizes(
            self.get_hparams(),
            self.env.experiment_config.slots_per_trial(),
            "EstimatorTrial",
        )

        self.experimental = EstimatorExperimentalContext(
            self.env,
            self.distributed,
            self._per_slot_batch_size,
        )

        if self.distributed.size > 1:
            optimizations_config = self.env.experiment_config.get_optimizations_config()
            self.aggregation_frequency = cast(
                int, optimizations_config.get("aggregation_frequency")
            )
            self.fp16_compression = cast(bool, optimizations_config.get("gradient_compression"))
            self.average_aggregated_gradients = cast(
                bool, optimizations_config.get("average_aggregated_gradients")
            )

        self.optimizer_initialized = False
        self.dataset_initialized = False

    def get_global_batch_size(self) -> int:
        """
        Return the global batch size.
        """
        return self._global_batch_size

    def get_per_slot_batch_size(self) -> int:
        """
        Return the per-slot batch size. When a model is trained with a single GPU, this is equal to
        the global batch size. When multi-GPU training is used, this is equal to the global batch
        size divided by the number of GPUs used to train the model.
        """
        return self._per_slot_batch_size

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
        if self.distributed.size == 1:
            return optimizer

        check.check_false(
            isinstance(optimizer, str),
            "Please specify an optimizer object instead of using a string name.",
        )

        hvd.require_horovod_type("tensorflow", "EstimatorTrialContext.wrap_optimizer was called.")

        # The signature of our horovod optimizer changed after we rebased onto 0.21.
        hvd_sig = inspect.signature(hvd.DistributedOptimizer)
        horovod_kwargs = {
            "compression": hvd.compression.Compression.fp16
            if self.fp16_compression
            else hvd.compression.Compression.none,
            "average_aggregated_gradients": self.average_aggregated_gradients,
        }
        if "aggregation_frequency" in hvd_sig.parameters:
            horovod_kwargs["aggregation_frequency"] = self.aggregation_frequency
        else:
            horovod_kwargs["backward_passes_per_step"] = self.aggregation_frequency

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

        hvd.require_horovod_type("tensorflow", "EstimatorTrialContext.wrap_dataset was called.")

        self.dataset_initialized = True
        if self.distributed.size == 1 or not shard_dataset:
            if self.distributed.size > 1 and not shard_dataset:
                logging.info("Dataset sharding skipped.")
            return dataset

        dataset = dataset.shard(hvd.size(), hvd.rank())
        logging.debug(f"Sharded dataset to index {hvd.rank()} of {hvd.size()}.")
        return dataset


class EstimatorExperimentalContext(_data_layer.DataLayerContext):
    """
    Context class that contains experimental runtime information and features
    for any Determined workflow that uses the ``tf.estimator`` API.

    ``EstimatorExperimentalContext`` extends ``EstimatorTrialContext`` under
    the ``context.experimental`` namespace.
    """

    def __init__(
        self,
        env: det.EnvContext,
        distributed_context: core.DistributedContext,
        per_slot_batch_size: int,
    ) -> None:
        super().__init__(env, distributed_context, per_slot_batch_size)
