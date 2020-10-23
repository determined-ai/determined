import logging

import tensorflow as tf
from packaging import version
from tensorflow.python.keras.engine import sequential

import determined as det
from determined import horovod, util
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


def _check_if_aggregation_frequency_will_work(
    model: tf.keras.Model,
    hvd_config: horovod.HorovodContext,
) -> None:
    if not hvd_config.use or hvd_config.aggregation_frequency == 1:
        return

    if model._is_graph_network or isinstance(model, sequential.Sequential):
        return

    if version.parse(tf.__version__) >= version.parse("2.4.0"):
        return

    if util.is_overridden(model.train_step, tf.keras.Model):
        logging.warning(
            "If you subclassing tf.keras.Model in TF 2.2 or TF 2.3 and defining "
            "a custom train_step() method, in order to use aggregation_frequency > 1 "
            "you need to include the following steps in your train_step(): "
            "For each optimizer you must call: `aggregated_gradients = "
            "optimizer._aggregate_gradients(grads, vars)` and then call "
            "`optimizer.apply_gradients(zip(aggregated_gradients, vars), "
            " experimental_aggregate_gradients=False)`."
        )
