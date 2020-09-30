from typing import Tuple

import tensorflow as tf

import determined as det
from determined import horovod
from determined.horovod import hvd


def _get_multi_gpu_model_and_optimizer(
    pre_compiled_model: tf.keras.Model,
    optimizer: tf.keras.optimizers.Optimizer,
    env: det.EnvContext,
    hvd_config: horovod.HorovodContext,
) -> Tuple[tf.keras.Model, tf.keras.optimizers.Optimizer]:
    num_gpus = len(env.container_gpus)
    new_model = pre_compiled_model
    new_optimizer = optimizer
    if num_gpus > 1 and not hvd_config.use:
        new_model = tf.keras.utils.multi_gpu_model(pre_compiled_model, num_gpus)
    # If using horovod, wrap the optimizer and check for an aggregation_frequency.
    elif hvd_config.use:
        # Horovod doesn't know how to handle string-based optimizers.
        if isinstance(optimizer, str):
            raise det.errors.InvalidExperimentException("string optimizers are not supported")

        new_optimizer = hvd.DistributedOptimizer(
            optimizer,
            aggregation_frequency=hvd_config.aggregation_frequency,
            average_aggregated_gradients=hvd_config.average_aggregated_gradients,
        )
    return new_model, new_optimizer
