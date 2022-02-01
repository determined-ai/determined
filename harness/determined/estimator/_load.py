import json
import logging
import pathlib
from typing import List, Optional

import tensorflow as tf
from tensorflow.python.training.tracking.tracking import AutoTrackable


def load_estimator_from_checkpoint_path(
    path: str, tags: Optional[List[str]] = None
) -> AutoTrackable:
    """
    Loads a checkpoint written by an EstimatorTrial.

    You should have already downloaded the checkpoint files, likely with
    :meth:`Checkpoint.download() <determined.experimental.client.Checkpoint.download()>`.

    The return type is a TensorFlow AutoTrackable object.

    Arguments:
        path (string): Top level directory to load the checkpoint from.
        tags (list string, optional): Specifies which tags are loaded from
            the TensorFlow SavedModel. See documentation for `tf.compat.v1.saved_model.load_v2
            <https://www.tensorflow.org/versions/r1.15/api_docs/python/tf/saved_model/load_v2>`_.
    """

    ckpt_dir = pathlib.Path(path)
    load_data_path = ckpt_dir.joinpath("load_data.json")
    metadata_path = ckpt_dir.joinpath("metadata.json")
    if load_data_path.exists():
        with load_data_path.open() as f:
            load_data = json.load(f)
        if load_data["trial_type"] != "EstimatorTrial":
            logging.warning(
                "Checkpoint does not appear to be a valid EstimatorTrial checkpoint, "
                "continuing anyway..."
            )
    elif metadata_path.exists():
        with metadata_path.open() as f:
            metadata = json.load(f)
        framework = metadata.get("framework", "")
        is_tf = framework.startswith("tensorflow") or "tensorflow_version" in metadata
        is_estimator = metadata.get("format", "saved_model") == "saved_model"
        if not is_tf or not is_estimator:
            logging.warning(
                "Checkpoint does not appear to be a valid EstimatorTrial checkpoint, "
                "continuing anyway..."
            )
    else:
        # With estimators, we don't actually need load_data or metadata to read the checkpoint,
        # so this is just a warning.
        logging.warning(
            "Unable to confirm that checkpoint is a valid EstimatorTrial checkpoint, "
            "continuing anyway..."
        )

    saved_model_paths = list(ckpt_dir.glob("**/saved_model.pb"))

    if len(saved_model_paths) > 1:
        raise AssertionError(
            "Checkpoint directory {} contains multiple \
            nested saved_model.pb files: {}".format(
                ckpt_dir, saved_model_paths
            )
        )

    # TensorFlow uses tags to determine which metagraph to load. Most
    # commonly, users will attempt to serve or otherwise use the model for
    # inference. Therefore we default to the serve graph tag which disables
    # operations that are only relevant for training.
    if tags is None:
        logging.info('No tags specified. Loading "serve" tag from saved_model.')
        tags = ["serve"]

    saved_model_path = saved_model_paths[0]
    return tf.compat.v1.saved_model.load_v2(str(saved_model_path.parent), tags)
