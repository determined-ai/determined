import logging
from typing import Any, Callable, Dict, Union

import tensorflow as tf

import determined as det
from determined import horovod
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

    def __init__(self, env: det.EnvContext, hvd_config: horovod.HorovodContext):
        self.hvd_config = hvd_config
        self.input_from_dataflow = env.experiment_config.input_from_dataflow()
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
        hvd.require_horovod_type("tensorflow", "EstimatorContext.wrap_optimizer was called.")

        self.optimizer_initialized = True
        if not self.hvd_config.use:
            return optimizer
        check.check_false(
            isinstance(optimizer, str),
            "Please specify an optimizer object instead of using a string name.",
        )
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

    def wrap_dataset(self, dataset: Any) -> Any:
        """
        This should be used to wrap ``tf.data.Dataset`` objects immediately after
        they have been created. Users should use the output of this wrapper as the
        new instance of their dataset. If users create multiple datasets (e.g., one
        for training and one for testing), users should wrap each dataset
        independently. E.g., If users instantiate their training dataset within
        ``build_train_spec()``, they should call ``dataset = wrap_dataset(dataset)``
        prior to passing it into ``tf.estimator.TrainSpec``.
        """
        hvd.require_horovod_type("tensorflow", "EstimatorContext.wrap_dataset was called.")

        self.dataset_initialized = True
        if not self.hvd_config.use or self.input_from_dataflow:
            return dataset
        dataset = dataset.shard(hvd.size(), hvd.rank())
        logging.debug(f"Sharded dataset to index {hvd.rank()} of {hvd.size()}.")
        return dataset


class EstimatorTrialContext(det.TrialContext, EstimatorContext):
    def __init__(self, env: det.EnvContext, hvd_config: horovod.HorovodContext) -> None:
        det.TrialContext.__init__(self, env, hvd_config)
        EstimatorContext.__init__(self, env, hvd_config)


class EstimatorNativeContext(det.NativeContext, EstimatorContext):
    def __init__(self, env: det.EnvContext, hvd_config: horovod.HorovodContext):
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
