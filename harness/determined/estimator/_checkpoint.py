import pathlib
from typing import List, Optional

import tensorflow as tf
from tensorflow.python.training.tracking.tracking import AutoTrackable

from determined_common.checkpoint import download


def load(
    trial_id: int,
    latest: bool = False,
    best: bool = False,
    uuid: Optional[str] = None,
    ckpt_path: Optional[str] = None,
    master: Optional[str] = None,
    metric_name: Optional[str] = None,
    smaller_is_better: Optional[bool] = None,
    tags: Optional[List[str]] = None,
) -> AutoTrackable:
    if not ckpt_path or not pathlib.Path(ckpt_path).exists():
        ckpt_path, _ = download(
            trial_id,
            latest=latest,
            best=best,
            uuid=uuid,
            output_dir=ckpt_path,
            master=master,
            metric_name=metric_name,
            smaller_is_better=smaller_is_better,
        )

    ckpt_dir = pathlib.Path(ckpt_path)
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
