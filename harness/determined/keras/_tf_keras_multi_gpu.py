import tensorflow as tf

import determined as det
from determined import horovod
from determined.horovod import hvd


def _get_multi_gpu_model_if_using_native_parallel(
    pre_compiled_model: tf.keras.Model,
    env: det.EnvContext,
    hvd_config: horovod.HorovodContext,
) -> tf.keras.Model:
    num_gpus = len(env.container_gpus)
    new_model = pre_compiled_model
    if num_gpus > 1 and not hvd_config.use:
        new_model = tf.keras.utils.multi_gpu_model(pre_compiled_model, num_gpus)

    return new_model


def _get_horovod_optimizer_if_using_horovod(
    optimizer: tf.keras.optimizers.Optimizer,
    hvd_config: horovod.HorovodContext,
) -> tf.keras.optimizers.Optimizer:
    if not hvd_config.use:
        return optimizer

    # Horovod doesn't know how to handle string-based optimizers.
    if isinstance(optimizer, str):
        raise det.errors.InvalidExperimentException("string optimizers are not supported")

    return hvd.DistributedOptimizer(
        optimizer,
        aggregation_frequency=hvd_config.aggregation_frequency,
        average_aggregated_gradients=hvd_config.average_aggregated_gradients,
    )
