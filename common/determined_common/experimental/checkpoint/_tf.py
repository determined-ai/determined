import pathlib
from typing import List, Optional

import tensorflow as tf
from tensorflow.python.training.tracking.tracking import AutoTrackable


def load_model(ckpt_dir: pathlib.Path, tags: Optional[List[str]] = None) -> AutoTrackable:
    saved_model_paths = list(ckpt_dir.glob("**/saved_model.pb"))
    if not saved_model_paths:
        raise FileNotFoundError(
            f"Checkpoint directory {ckpt_dir} does not contain a nested saved_model.pb"
        )
    elif len(saved_model_paths) > 1:
        raise AssertionError(
            f"Checkpoint directory {ckpt_dir} contains multiple \
            nested saved_model.pb files {saved_model_paths}"
        )

    if not tags:
        print('No tags specified. Loading "serve" tag from saved_model.')
        tags = ["serve"]

    saved_model_path = saved_model_paths[0]
    return tf.compat.v1.saved_model.load_v2(str(saved_model_path.parent), tags)
