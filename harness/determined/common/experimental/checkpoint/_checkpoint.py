import dataclasses
import enum
import json
import pathlib
import shutil
import warnings
from typing import Any, Dict, List, Optional, cast

from determined.common import constants, storage
from determined.common.experimental import session
from determined.common.storage import shared


class ModelFramework(enum.Enum):
    PYTORCH = 1
    TENSORFLOW = 2


class CheckpointState(enum.Enum):
    UNSPECIFIED = 0
    ACTIVE = 1
    COMPLETED = 2
    ERROR = 3
    DELETED = 4


@dataclasses.dataclass
class CheckpointTrainingMetadata:
    experiment_config: Dict[str, Any]
    experiment_id: str
    trial_id: str
    hparams: Dict[str, Any]
    validation_metrics: Dict[str, Any]


class Checkpoint(object):
    """
    A Checkpoint object is usually obtained from
    ``determined.experimental.client.get_checkpoint()``.

    A ``Checkpoint`` represents a trained model.

    This class provides helper functionality for downloading checkpoints to
    local storage and loading checkpoints into memory.

    The :class:`~determined.experimental.TrialReference` class contains methods
    that return instances of this class.
    """

    def __init__(
        self,
        session: session.Session,
        task_id: str,
        allocation_id: str,
        uuid: str,
        report_time: Optional[str],
        resources: Dict[str, Any],
        metadata: Dict[str, Any],
        state: CheckpointState,
        training: Optional[CheckpointTrainingMetadata] = None,
    ):
        self._session = session
        self.task_id = task_id
        self.allocation_id = allocation_id
        self.uuid = uuid
        self.report_time = report_time
        self.resources = resources
        self.metadata = metadata
        self.state = state
        self.training = training

    def _find_shared_fs_path(self, checkpoint_storage: Dict[str, Any]) -> pathlib.Path:
        """Attempt to find the path of the checkpoint if being configured to shared fs.
        This function assumes the host path of the shared fs exists.
        """
        host_path = checkpoint_storage["host_path"]
        storage_path = checkpoint_storage.get("storage_path")
        potential_paths = [
            pathlib.Path(shared._full_storage_path(host_path, storage_path), self.uuid),
            pathlib.Path(
                shared._full_storage_path(
                    host_path, storage_path, constants.SHARED_FS_CONTAINER_PATH
                ),
                self.uuid,
            ),
        ]

        for path in potential_paths:
            if path.exists():
                return path

        raise FileNotFoundError(
            "Checkpoint {} not found in {}. This error could be caused by not having "
            "the same shared file system mounted on the local machine as the experiment "
            "checkpoint storage configuration.".format(self.uuid, potential_paths)
        )

    def download(self, path: Optional[str] = None) -> str:
        """
        Download checkpoint to local storage.

        See also:

          - :func:`determined.pytorch.load_trial_from_checkpoint_path`
          - :func:`determined.keras.load_model_from_checkpoint_path`
          - :func:`determined.estimator.load_estimator_from_checkpoint_path`

        Arguments:
            path (string, optional): Top level directory to place the
                checkpoint under. If this parameter is not set, the checkpoint will
                be downloaded to ``checkpoints/<checkpoint_uuid>`` relative to the
                current working directory.
        """
        if path is not None:
            local_ckpt_dir = pathlib.Path(path)
        else:
            local_ckpt_dir = pathlib.Path("checkpoints", self.uuid)

        # Backward compatibility: we used MLflow's MLmodel checkpoint format for
        # serializing pytorch models. We now use our own format that contains a
        # metadata.json file. We are checking for checkpoint existence by
        # looking for both checkpoint formats in the output directory.
        potential_metadata_paths = [
            local_ckpt_dir.joinpath(f) for f in ["metadata.json", "MLmodel"]
        ]
        if not any(p.exists() for p in potential_metadata_paths):
            # If the target directory doesn't already appear to contain a
            # checkpoint, attempt to fetch one.
            if self.training is None:
                raise NotImplementedError("Non-training checkpoints cannot be downloaded")

            checkpoint_storage = self.training.experiment_config["checkpoint_storage"]
            if checkpoint_storage["type"] == "shared_fs":
                src_ckpt_dir = self._find_shared_fs_path(checkpoint_storage)
                shutil.copytree(str(src_ckpt_dir), str(local_ckpt_dir))
            else:
                local_ckpt_dir.mkdir(parents=True, exist_ok=True)
                manager = storage.build(
                    checkpoint_storage,
                    container_path=None,
                )
                if not isinstance(
                    manager,
                    (
                        storage.S3StorageManager,
                        storage.GCSStorageManager,
                        storage.AzureStorageManager,
                    ),
                ):
                    raise AssertionError(
                        "Downloading from Azure, S3 or GCS requires the experiment to be "
                        "configured with Azure, S3 or GCS checkpointing, {} found instead".format(
                            checkpoint_storage["type"]
                        )
                    )

                manager.download(self.uuid, str(local_ckpt_dir))

        # As of v0.18.0, we write metadata.json once at upload time.  Checkpoints uploaded prior to
        # 0.18.0 will not have a metadata.json present.  Unfortunately, checkpoints earlier than
        # 0.17.7 depended on this file existing in order to be loaded.  Therefore, when we detect
        # that the metadata.json file is not present, we write it to make sure those checkpoints can
        # still load.
        metadata_path = local_ckpt_dir.joinpath("metadata.json")
        if not metadata_path.exists():
            self.write_metadata_file(str(metadata_path))

        return str(local_ckpt_dir)

    def write_metadata_file(self, path: str) -> None:
        """
        Write a file with this Checkpoint's metadata inside of it.

        This is normally executed as part of Checkpoint.download().  However, in the special case
        where you are accessing the checkpoint files directly (not via Checkpoint.download) you may
        use this method directly to obtain the latest metadata.
        """
        with open(path, "w") as f:
            json.dump(self.metadata, f, indent=2)

    def load(
        self, path: Optional[str] = None, tags: Optional[List[str]] = None, **kwargs: Any
    ) -> Any:
        """Loads a Determined checkpoint into memory.

        If the checkpoint is not present on disk it will be downloaded from persistent storage.
        The behavior here is different for TensorFlow and PyTorch checkpoints.

        For PyTorch checkpoints, the return type is an object that inherits from
        ``determined.pytorch.PyTorchTrial`` as defined by the ``entrypoint`` field
        in the experiment config.

        For TensorFlow checkpoints, the return type is a TensorFlow autotrackable object.

        Arguments:
            path (string, optional): Top level directory to load the
                checkpoint from. (default: ``checkpoints/<UUID>``)
            tags (list string, optional): Only relevant for TensorFlow
                SavedModel checkpoints. Specifies which tags are loaded from
                the TensorFlow SavedModel. See documentation for
                `tf.compat.v1.saved_model.load_v2
                <https://www.tensorflow.org/versions/r1.15/api_docs/python/tf/saved_model/load_v2>`_.
            kwargs: Only relevant for PyTorch checkpoints. The keyword arguments
                will be applied to ``torch.load``. See documentation for `torch.load
                <https://pytorch.org/docs/stable/torch.html?highlight=torch%20load#torch.load>`_.

        .. warning::

           Checkpoint.load() has been deprecated and will be removed in a future version.

           Please combine Checkpoint.download() with one of the following instead:
             - ``det.pytorch.load_trial_from_checkpoint()``
             - ``det.keras.load_model_from_checkpoint()``
             - ``det.estimator.load_estimator_from_checkpoint_path()``
        """
        warnings.warn(
            "Checkpoint.load() has been deprecated and will be removed in a future version.\n"
            "\n"
            "Please combine Checkpoint.download() with one of the following instead:\n"
            "  - det.pytorch.load_trial_from_checkpoint_path()\n"
            "  - det.keras.load_model_from_checkpoint_path()\n"
            "  - det.estimator.load_estimator_from_checkpoint_path()\n",
            FutureWarning,
        )
        ckpt_path = self.download(path)
        return Checkpoint.load_from_path(ckpt_path, tags=tags, **kwargs)

    def add_metadata(self, metadata: Dict[str, Any]) -> None:
        """
        Adds user-defined metadata to the checkpoint. The ``metadata`` argument must be a
        JSON-serializable dictionary. If any keys from this dictionary already appear in
        the checkpoint metadata, the corresponding dictionary entries in the checkpoint are
        replaced by the passed-in dictionary values.

        Warning: this metadata change is not propagated to the checkpoint storage.

        Arguments:
            metadata (dict): Dictionary of metadata to add to the checkpoint.
        """
        for key, val in metadata.items():
            self.metadata[key] = val

        self._session.post(
            "/api/v1/checkpoints/{}/metadata".format(self.uuid),
            json={"checkpoint": {"metadata": self.metadata}},
        )

    def remove_metadata(self, keys: List[str]) -> None:
        """
        Removes user-defined metadata from the checkpoint. Any top-level keys that
        appear in the ``keys`` list are removed from the checkpoint.

        Warning: this metadata change is not propagated to the checkpoint storage.

        Arguments:
            keys (List[string]): Top-level keys to remove from the checkpoint metadata.
        """

        for key in keys:
            if key in self.metadata:
                del self.metadata[key]

        self._session.post(
            "/api/v1/checkpoints/{}/metadata".format(self.uuid),
            json={"checkpoint": {"metadata": self.metadata}},
        )

    @staticmethod
    def load_from_path(path: str, tags: Optional[List[str]] = None, **kwargs: Any) -> Any:
        """Loads a Determined checkpoint from a local file system path into memory.

        For PyTorch checkpoints, the return type is an object that inherits from
        ``determined.pytorch.PyTorchTrial`` as defined by the ``entrypoint`` field
        in the experiment config.

        For TensorFlow checkpoints, the return type is a TensorFlow autotrackable object.

        Arguments:
            path (string): Local path to the checkpoint directory.
            tags (list string, optional): Only relevant for TensorFlow
                SavedModel checkpoints. Specifies which tags are loaded from
                the TensorFlow SavedModel. See documentation for
                `tf.compat.v1.saved_model.load_v2
                <https://www.tensorflow.org/versions/r1.15/api_docs/python/tf/saved_model/load_v2>`_.

        .. warning::

           Checkpoint.load_from_path() has been deprecated and will be removed in a future version.

           Please use one of the following instead to load your checkpoint:
             - ``det.pytorch.load_trial_from_checkpoint_path()``
             - ``det.keras.load_model_from_checkpoint_path()``
             - ``det.estimator.load_estimator_from_checkpoint_path()``
        """
        warnings.warn(
            "Checkpoint.load_from_path() has been deprecated and will be removed in a future "
            "version.\n"
            "\n"
            "Please use one of the following instead to load your checkpoint:\n"
            "  - det.pytorch.load_trial_from_checkpoint_path()\n"
            "  - det.keras.load_model_from_checkpoint_path()\n"
            "  - det.estimator.load_estimator_from_checkpoint_path()\n",
            FutureWarning,
        )
        checkpoint_dir = pathlib.Path(path)
        metadata = Checkpoint._parse_metadata(checkpoint_dir)
        checkpoint_type = Checkpoint._get_type(metadata)

        if checkpoint_type == ModelFramework.PYTORCH:
            from determined import pytorch

            return pytorch.load_trial_from_checkpoint_path(path, **kwargs)

        if checkpoint_type == ModelFramework.TENSORFLOW:
            save_format = metadata.get("format", "saved_model")

            # For tf.estimators we save the entire model using the saved_model format.
            # For tf.keras we save only the weights also using the saved_model format,
            # which we call saved_weights.
            if cast(str, save_format) == "saved_model":
                from determined import estimator

                return estimator.load_estimator_from_checkpoint_path(path, tags)

            if save_format in ("saved_weights", "h5"):
                from determined import keras

                return keras.load_model_from_checkpoint_path(path, tags)

        raise AssertionError("Unknown checkpoint format at {}".format(path))

    @staticmethod
    def _parse_metadata(directory: pathlib.Path) -> Dict[str, Any]:
        metadata_path = directory.joinpath("metadata.json")
        with metadata_path.open() as f:
            metadata = json.load(f)

        return cast(Dict[str, Any], metadata)

    @staticmethod
    def parse_metadata(directory: pathlib.Path) -> Dict[str, Any]:
        warnings.warn(
            "Checkpoint.parse_metadata() is deprecated and will be removed from the public API "
            "in a future version",
            FutureWarning,
        )
        return Checkpoint._parse_metadata(directory)

    @staticmethod
    def _get_type(metadata: Dict[str, Any]) -> ModelFramework:
        if "framework" in metadata:
            if metadata["framework"].startswith("torch"):
                return ModelFramework.PYTORCH

            if metadata["framework"].startswith("tensorflow"):
                return ModelFramework.TENSORFLOW

        # Older metadata layout contained torch_version and tensorflow_version
        # as keys. Eventually, we should drop support for the older format.
        if "torch_version" in metadata:
            return ModelFramework.PYTORCH

        elif "tensorflow_version" in metadata:
            return ModelFramework.TENSORFLOW

        raise AssertionError("Unknown checkpoint format")

    @staticmethod
    def get_type(metadata: Dict[str, Any]) -> ModelFramework:
        warnings.warn(
            "Checkpoint.get_type() is deprecated and will be removed from the public API "
            "in a future version",
            FutureWarning,
        )
        return Checkpoint._get_type(metadata)

    def __repr__(self) -> str:
        if self.training is not None:
            return (
                f"Checkpoint(uuid={self.uuid}, task_id={self.task_id},"
                f" trial_id={self.training.trial_id})"
            )
        else:
            return f"Checkpoint(uuid={self.uuid}, task_id={self.task_id})"

    @classmethod
    def _from_json(cls, data: Dict[str, Any], session: session.Session) -> "Checkpoint":
        metadata = data.get("metadata", {})
        training_data = data.get("training")
        training = (
            CheckpointTrainingMetadata(
                training_data["experimentConfig"],
                training_data["experimentId"],
                training_data["trialId"],
                training_data["hparams"],
                training_data["validationMetrics"],
            )
            if training_data
            else None
        )

        return cls(
            session,
            data["taskId"],
            data["allocationId"],
            data["uuid"],
            data.get("reportTime"),
            data["resources"],
            metadata,
            data["state"],
            training,
        )

    @classmethod
    def from_json(cls, data: Dict[str, Any], session: session.Session) -> "Checkpoint":
        warnings.warn(
            "Checkpoint.from_json() is deprecated and will be removed from the public API "
            "in a future version",
            FutureWarning,
        )
        return cls._from_json(data, session)
