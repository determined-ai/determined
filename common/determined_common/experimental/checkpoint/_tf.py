import pathlib
from typing import List, Optional

import tensorflow as tf
from tensorflow.python.training.tracking.tracking import AutoTrackable


def load_model(ckpt_dir: pathlib.Path, tags: Optional[List[str]] = None) -> AutoTrackable:
    saved_model_paths = list(ckpt_dir.glob("**/saved_model.pb"))
    h5_paths = list(ckpt_dir.glob("**/*.h5"))

    if not h5_paths and not saved_model_paths:
        raise AssertionError(
            "No checkpoint saved_model.pb or h5 files found at {}".format(ckpt_dir)
        )

    # Tensorflow 1 favors saved_models for tf.estimators and h5 for tf.keras
    # models. Tensorflow is moving towards saved_model for both high level
    # APIs in tf.2. For this reason we favor the saved_model below but also
    # check for h5 models.
    if saved_model_paths:
        if len(saved_model_paths) > 1:
            raise AssertionError(
                "Checkpoint directory {} contains multiple \
                nested saved_model.pb files: {}".format(
                    ckpt_dir, saved_model_paths
                )
            )

        if tags is None:
            print('No tags specified. Loading "serve" tag from saved_model.')
            tags = ["serve"]

        saved_model_path = saved_model_paths[0]
        return tf.compat.v1.saved_model.load_v2(str(saved_model_path.parent), tags)

    else:
        if len(h5_paths) > 1:
            raise AssertionError(
                "Checkpoint directory {} contains multiple \
                nested .h5 files: {}".format(
                    ckpt_dir, h5_paths
                )
            )
        return tf.keras.models.load_model(h5_paths[0])
