import enum
import json
import pathlib
import shutil
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
        uuid: str,
        experiment_config: Dict[str, Any],
        experiment_id: int,
        trial_id: int,
        hparams: Dict[str, Any],
        batch_number: int,
        start_time: str,
        end_time: str,
        resources: Dict[str, Any],
        validation: Dict[str, Any],
        metadata: Dict[str, Any],
        determined_version: Optional[str] = None,
        framework: Optional[str] = None,
        format: Optional[str] = None,  # noqa: A002
        model_version: Optional[int] = None,
        model_name: Optional[str] = None,
    ):
        self._session = session
        self.uuid = uuid
        self.experiment_config = experiment_config
        self.experiment_id = experiment_id
        self.trial_id = trial_id
        self.hparams = hparams
        self.batch_number = batch_number
        self.start_time = start_time
        self.end_time = end_time
        self.resources = resources
        self.validation = validation
        self.framework = framework
        self.format = format
        self.determined_version = determined_version
        self.model_version = model_version
        self.model_name = model_name
        self.metadata = metadata

    def _find_shared_fs_path(self) -> pathlib.Path:
        """Attempt to find the path of the checkpoint if being configured to shared fs.
        This function assumes the host path of the shared fs exists.
        """
        host_path = self.experiment_config["checkpoint_storage"]["host_path"]
        storage_path = self.experiment_config["checkpoint_storage"].get("storage_path")
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
            if self.experiment_config["checkpoint_storage"]["type"] == "shared_fs":
                src_ckpt_dir = self._find_shared_fs_path()
                shutil.copytree(str(src_ckpt_dir), str(local_ckpt_dir))
            else:
                local_ckpt_dir.mkdir(parents=True, exist_ok=True)
                manager = storage.build(
                    self.experiment_config["checkpoint_storage"],
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
                            self.experiment_config["checkpoint_storage"]["type"]
                        )
                    )

                manager.download(self.uuid, str(local_ckpt_dir))

        if not local_ckpt_dir.joinpath("metadata.json").exists():
            with open(local_ckpt_dir.joinpath("metadata.json"), "w") as f:
                json.dump(
                    {
                        "determined_version": self.determined_version,
                        "framework": self.framework,
                        "format": self.format,
                        "experiment_id": self.experiment_id,
                        "trial_id": self.trial_id,
                        "hparams": self.hparams,
                        "experiment_config": self.experiment_config,
                        "metadata": self.metadata,
                    },
                    f,
                    indent=2,
                )

        return str(local_ckpt_dir)

    def load(
        self, path: Optional[str] = None, tags: Optional[List[str]] = None, **kwargs: Any
    ) -> Any:
        """
        Loads a Determined checkpoint into memory. If the checkpoint is not
        present on disk it will be downloaded from persistent storage.

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
        """
        ckpt_path = self.download(path)
        return Checkpoint.load_from_path(ckpt_path, tags=tags, **kwargs)

    def add_metadata(self, metadata: Dict[str, Any]) -> None:
        """
        Adds user-defined metadata to the checkpoint. The ``metadata`` argument must be a
        JSON-serializable dictionary. If any keys from this dictionary already appear in
        the checkpoint metadata, the corresponding dictionary entries in the checkpoint are
        replaced by the passed-in dictionary values.

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
        """
        Loads a Determined checkpoint from a local file system path into
        memory. If the checkpoint is a PyTorch model, a ``torch.nn.Module`` is returned.
        If the checkpoint contains a TensorFlow SavedModel, a TensorFlow
        autotrackable object is returned.

        Arguments:
            path (string): Local path to the checkpoint directory.
            tags (list string, optional): Only relevant for TensorFlow
                SavedModel checkpoints. Specifies which tags are loaded from
                the TensorFlow SavedModel. See documentation for
                `tf.compat.v1.saved_model.load_v2
                <https://www.tensorflow.org/versions/r1.15/api_docs/python/tf/saved_model/load_v2>`_.
        """
        checkpoint_dir = pathlib.Path(path)
        metadata = Checkpoint.parse_metadata(checkpoint_dir)
        checkpoint_type = Checkpoint.get_type(metadata)

        if checkpoint_type == ModelFramework.PYTORCH:
            import determined.common.experimental.checkpoint._torch

            return determined.common.experimental.checkpoint._torch.load_model(
                checkpoint_dir, metadata, **kwargs
            )

        elif checkpoint_type == ModelFramework.TENSORFLOW:
            import determined.common.experimental.checkpoint._tf

            return determined.common.experimental.checkpoint._tf.load_model(
                checkpoint_dir, metadata, tags=tags
            )

        raise AssertionError("Unknown checkpoint format at {}".format(checkpoint_dir))

    @staticmethod
    def parse_metadata(directory: pathlib.Path) -> Dict[str, Any]:
        metadata_path = directory.joinpath("metadata.json")
        with metadata_path.open() as f:
            metadata = json.load(f)

        return cast(Dict[str, Any], metadata)

    @staticmethod
    def get_type(metadata: Dict[str, Any]) -> ModelFramework:
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

    def __repr__(self) -> str:
        if self.model_name is not None:
            return "Checkpoint(uuid={}, trial_id={}, model={}, version={})".format(
                self.uuid, self.trial_id, self.model_name, self.model_version
            )
        return "Checkpoint(uuid={}, trial_id={})".format(self.uuid, self.trial_id)

    @staticmethod
    def from_json(data: Dict[str, Any], session: session.Session) -> "Checkpoint":
        validation = {
            "metrics": data.get("metrics", {}),
            "state": data.get("validation_state", None),
        }

        return Checkpoint(
            session,
            data["uuid"],
            data.get("experiment_config", data.get("experimentConfig")),
            data.get("experiment_id", data.get("experimentId")),
            data.get("trial_id", data.get("trialId")),
            data["hparams"],
            data.get("batch_number", data.get("batchNumber")),
            data.get("start_time", data.get("startTime")),
            data.get("end_time", data.get("endTime")),
            data["resources"],
            validation,
            data.get("metadata", {}),
            framework=data.get("framework"),
            format=data.get("format"),
            determined_version=data.get("determined_version", data.get("determinedVersion")),
            model_version=data.get("model_version"),
            model_name=data.get("model_name"),
        )
