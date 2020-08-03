import pathlib
from typing import Any, Dict, List, Optional, cast

import tensorflow as tf
from tensorflow.python.training.tracking.tracking import AutoTrackable

from determined import experimental
from determined.keras import TFKerasTrial


def load_model(
    ckpt_dir: pathlib.Path, metadata: Dict[str, Any], tags: Optional[List[str]] = None
) -> AutoTrackable:
    save_format = metadata.get("format", None)

    # Tensorflow 1 favors saved_models for tf.estimators and h5 for tf.keras
    # models. Tensorflow is moving towards saved_model for both high level
    # APIs in tf.2.
    if not save_format or cast(str, save_format) == "saved_model":
        return load_saved_model(ckpt_dir, tags=tags)

    elif save_format == "h5":
        trial_cls, trial_context = experimental._load_trial_on_local(
            ckpt_dir.joinpath("code"),
            training=False,
            config=metadata["experiment_config"],
            hparams=metadata["hparams"],
        )

        trial = cast(TFKerasTrial, trial_cls(trial_context))
        model = trial.build_model()
        model.load_weights(str(ckpt_dir.joinpath("determined-keras-model.h5")))
        return model
    else:
        raise AssertionError("Unknown checkpoint format at {}".format(str(ckpt_dir)))


def load_saved_model(ckpt_dir: pathlib.Path, tags: Optional[List[str]] = None) -> AutoTrackable:
    saved_model_paths = list(ckpt_dir.glob("**/saved_model.pb"))

    if len(saved_model_paths) > 1:
        raise AssertionError(
            "Checkpoint directory {} contains multiple \
            nested saved_model.pb files: {}".format(
                ckpt_dir, saved_model_paths
            )
        )

    # Tensorflow uses tags to determine which metagraph to load. Most
    # commonly, users will attempt to serve or otherwise use the model for
    # inference. Therefore we default to the serve graph tag which disables
    # operations that are only relevant for training.
    if tags is None:
        print('No tags specified. Loading "serve" tag from saved_model.')
        tags = ["serve"]

    saved_model_path = saved_model_paths[0]
    return tf.compat.v1.saved_model.load_v2(str(saved_model_path.parent), tags)
