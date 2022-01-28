import json
import logging
import pathlib
from typing import List, Optional, cast

from tensorflow.python.training.tracking.tracking import AutoTrackable

import determined as det
from determined import keras


def load_model_from_checkpoint_path(path: str, tags: Optional[List[str]] = None) -> AutoTrackable:
    """
    Loads a checkpoint written by a TFKerasTrial.

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
        if load_data["trial_type"] != "TFKerasTrial":
            logging.warning(
                "Checkpoint does not appear to be a valid TFKerasTrial checkpoint, "
                "continuing anyway..."
            )
        experiment_config = load_data["experiment_config"]
        hparams = load_data["hparams"]
        trial_cls_spec = load_data["trial_cls_spec"]
        filename = "determined-keras-model-weights"
    elif metadata_path.exists():
        # Old checkpoints (<=0.17.7) used to depend on metadata coming from the master in
        # Checkpoint.download().
        with metadata_path.open() as f:
            metadata = json.load(f)
        framework = metadata.get("framework", "")
        is_tf = framework.startswith("tensorflow") or "tensorflow_version" in metadata
        save_format = metadata.get("format")
        is_keras = save_format not in ("saved_weights", "h5")
        if not is_tf or not is_keras:
            logging.warning(
                "Checkpoint does not appear to be a valid TFKerasTrial checkpoint, "
                "continuing anyway..."
            )
        if save_format == "saved_weights":
            filename = "determined-keras-model-weights"
        elif save_format == "h5":
            # This is how tf.keras models were saved prior to Determined 0.13.8.
            filename = "determined-keras-model.h5"
        else:
            raise AssertionError("Unknown checkpoint format at {}".format(str(ckpt_dir)))
        experiment_config = metadata["experiment_config"]
        hparams = metadata["hparams"]
        # When this format was in use, all entrypoints were trial classes.
        trial_cls_spec = experiment_config["entrypoint"]
    else:
        raise AssertionError(
            "Checkpoint does not have either load_data.json or metadata.json.  Checkpoints written "
            "by Determined 0.17.7 and earlier did not save enough information to be loaded "
            "directly from the files in the checkpoint.  Instead, a metadata.json was written "
            "during the call to Checkpoint.download().  If you are reading an old checkpoint "
            "directly from checkpoint storage, you can either use Checkpoint.download() instead or "
            "you can use Checkpoint.write_metatdata_file('metadata.json') to create a suitable "
            "metadata file for loading a legacy checkpoint."
        )

    trial_cls, trial_context = det._load_trial_for_checkpoint_export(
        ckpt_dir.joinpath("code"),
        managed_training=False,
        trial_cls_spec=trial_cls_spec,
        config=experiment_config,
        hparams=hparams,
    )

    trial = cast(keras.TFKerasTrial, trial_cls(trial_context))
    model = trial.build_model()
    model.load_weights(str(ckpt_dir.joinpath(filename)))
    return model
