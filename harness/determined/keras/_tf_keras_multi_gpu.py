import json
import os
from typing import Any, Dict, List, Optional, Tuple

import tensorflow as tf

import determined as det
from determined import horovod
from determined.horovod import hvd


def get_horovod_config(
    exp_config: Dict[str, Any],
    hvd_config: horovod.HorovodContext,
    profile_frequency: Optional[int],
    profile_filename: str,
) -> Dict[str, Any]:

    aggregation_frequency = hvd_config.aggregation_frequency
    grad_updated_sizes_dict = None  # type: Optional[Dict[int, List[int]]]
    if aggregation_frequency > 1 and hvd_config.grad_updates_size_file:
        grad_update_sizes_file_path = os.path.join(
            exp_config.get("data", {}).get("data_dir", ""), hvd_config.grad_updates_size_file
        )
        if not os.path.isfile(grad_update_sizes_file_path):
            raise AssertionError(
                f"Please move {hvd_config.grad_updates_size_file} inside 'data_dir'."
            )
        with open(grad_update_sizes_file_path, "r") as json_file:
            grad_updated_sizes_dict = json.load(json_file)

    return {
        "aggregation_frequency": aggregation_frequency,
        "grad_updated_sizes_dict": grad_updated_sizes_dict,
        "profile_frequency": profile_frequency,
        "profile_filename": profile_filename,
        "average_aggregated_gradients": hvd_config.average_aggregated_gradients,
    }


def _get_multi_gpu_model_and_optimizer(
    pre_compiled_model: tf.keras.Model,
    optimizer: tf.keras.optimizers.Optimizer,
    env: det.EnvContext,
    hvd_config: horovod.HorovodContext,
    profile_frequency: Optional[int],
    profile_filename: str,
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
            **get_horovod_config(
                exp_config=env.experiment_config,
                hvd_config=hvd_config,
                profile_frequency=profile_frequency,
                profile_filename=profile_filename,
            ),
        )
    return new_model, new_optimizer
