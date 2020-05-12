import enum
import json
import pathlib
import shutil
from typing import Any, Dict, List, Optional

from determined_common import api, storage


class ModelFramework(enum.Enum):
    PYTORCH = 1
    TENSORFLOW = 2


class Checkpoint(object):
    """
    Class representing a checkpoint. Contains methods for downloading
    checkpoints to a local path and loading checkpoints into memory.

    The ``det.experimental.Trial`` class contains methods that return instances
    of this class.
    """

    def __init__(
        self,
        uuid: str,
        storage_config: Dict[str, Any],
        batch_number: int,
        start_time: str,
        end_time: str,
        resources: Dict[str, Any],
        validation: Dict[str, Any],
    ):
        """
        Arguments:
            uuid (string): UUID of the checkpoint.
            storage_config: The checkpoint_storage key of the experiment
                configuration related to checkpoint.
            batch_number: Batch number of the checkpoint.
            start_time: Timestamp of when the checkpoint began being saved to
                persistent storage.
            end_time: Timestamp of when the checkpoint completed being saved to
                persistent storage.
            resources:  Dictionary of file paths to file sizes in bytes of all
                files related to the checkpoint.
            validation: Dictionary of validation metric names to their values.
        """

        self.uuid = uuid
        self.batch_number = batch_number
        self.storage_config = storage_config
        self.start_time = start_time
        self.end_time = end_time
        self.resources = resources
        self.validation = validation

    def _find_shared_fs_path(self) -> pathlib.Path:
        potential_paths = [
            [
                self.storage_config["container_path"],
                self.storage_config.get("storage_path", ""),
                self.uuid,
            ],
            [
                self.storage_config["host_path"],
                self.storage_config.get("storage_path", ""),
                self.uuid,
            ],
        ]

        for path in potential_paths:
            maybe_ckpt = pathlib.Path(*path)
            if maybe_ckpt.exists():
                return maybe_ckpt

        raise FileNotFoundError("Checkpoint {} not found".format(self.uuid))

    def download(self, path: Optional[str] = None) -> str:
        """
        Download checkpoint from the checkpoint storage location locally.

        Arguments:
            path (string, optional): Top level directory to place the
                checkpoint under. If this parameter is not set the checkpoint will
                be downloaded to `checkpoints/<checkpoint_uuid>` relative to the
                current working directory.
        """
        if path is not None:
            local_ckpt_dir = pathlib.Path(path)
        else:
            local_ckpt_dir = pathlib.Path("checkpoints", self.uuid)

        # If the target directory doesn't already appear to contain a
        # checkpoint, attempt to fetch one.

        # We used MLflow's MLmodel checkpoint format in the past for
        # serializing pytorch models. We now use our own format that contains a
        # metadata.json file. We are checking for checkpoint existence by
        # looking for both checkpoint formats in the output directory.
        potential_metadata_paths = [
            local_ckpt_dir.joinpath(f) for f in ["metadata.json", "MLmodel"]
        ]
        if not any(p.exists() for p in potential_metadata_paths):
            if self.storage_config["type"] == "shared_fs":
                src_ckpt_dir = self._find_shared_fs_path()
                shutil.copytree(str(src_ckpt_dir), str(local_ckpt_dir))
            else:
                local_ckpt_dir.mkdir(parents=True, exist_ok=True)
                manager = storage.build(self.storage_config)
                if not isinstance(manager, (storage.S3StorageManager, storage.GCSStorageManager)):
                    raise AssertionError(
                        "Downloading from S3 or GCS requires the experiment to be configured with "
                        "S3 or GCS checkpointing, {} found instead".format(
                            self.storage_config["type"]
                        )
                    )

                metadata = storage.StorageMetadata.from_json(
                    {"uuid": self.uuid, "resources": self.resources}
                )
                manager.download(metadata, str(local_ckpt_dir))

        return str(local_ckpt_dir)

    def load(
        self, path: Optional[str] = None, tags: Optional[List[str]] = None, **kwargs: Any
    ) -> Any:
        """
        Loads a Determined checkpoint into memory. If the checkpoint is not
        present on disk it will be downloaded from persistent storage.

        Arguments:
            path (string, optional): Top level directory to load the
                checkpoint from. (default: ``checkpoint/<UUID>``)
            tags (list string, optional): Only relevant for tensorflow
                saved_model checkpoints. Specifies which tags are loaded from
                the tensoflow saved_model. See documentation for
                `tf.compat.v1.saved_model.load_v2
                <https://www.tensorflow.org/versions/r1.15/api_docs/python/tf/saved_model/load_v2>`_.
            kwargs: Only relevant for PyTorch checkpoints. The keyword arguments
                will be applied to torch.load. See documentation for `torch.load
                <https://pytorch.org/docs/stable/torch.html?highlight=torch%20load#torch.load>`_.
        """
        ckpt_path = self.download(path)
        return Checkpoint.load_from_path(ckpt_path, tags, **kwargs)

    @staticmethod
    def load_from_path(path: str, tags: Optional[List[str]] = None, **kwargs: Any) -> Any:
        """
        Loads a Determined checkpoint from a local file system path into
        memory. If the checkpoint is a pytorch model a ``torch.nn.Module`` is returned.
        If the checkpoint contains a tensorflow saved_model a tensorflow
        autotrackable object is returned.

        Arguments:
            path (string): Local path to the top level directory of a checkpoint.
            tags (list string, optional): Only relevant for tensorflow
                saved_model checkpoints. Specifies which tags are loaded from
                the tensoflow saved_model. See documentation for
                `tf.compat.v1.saved_model.load_v2
                <https://www.tensorflow.org/versions/r1.15/api_docs/python/tf/saved_model/load_v2>`_.
        """
        checkpoint_dir = pathlib.Path(path)

        checkpoint_type = Checkpoint.get_type(checkpoint_dir)
        if checkpoint_type == ModelFramework.PYTORCH:
            import determined_common.experimental.checkpoint._torch

            return determined_common.experimental.checkpoint._torch.load_model(
                checkpoint_dir, **kwargs
            )

        elif checkpoint_type == ModelFramework.TENSORFLOW:
            import determined_common.experimental.checkpoint._tf

            return determined_common.experimental.checkpoint._tf.load_model(
                checkpoint_dir, tags=tags
            )

        raise AssertionError("Unknown checkpoint format at {}".format(checkpoint_dir))

    @staticmethod
    def get_type(directory: pathlib.Path) -> ModelFramework:
        # We used MLflow's MLmodel checkpoint format in the past for
        # serializing pytorch models.
        if directory.joinpath("MLmodel").exists():
            return ModelFramework.PYTORCH

        metadata_path = directory.joinpath("metadata.json")
        with metadata_path.open() as f:
            metadata = json.load(f)

        if "torch_version" in metadata:
            return ModelFramework.PYTORCH

        elif "tensorflow_version" in metadata:
            return ModelFramework.TENSORFLOW

        raise AssertionError("Unknown checkpoint format at {}".format(directory))

    def __repr__(self) -> str:
        return "Checkpoint(uuid={})".format(self.uuid)


def get_checkpoint(uuid: str, master: str) -> Checkpoint:
    r = api.get(master, "checkpoints/{}".format(uuid)).json()
    return from_json(r)


def from_json(data: Dict[str, Any]) -> Checkpoint:
    validation = {
        "metrics": data.get("metrics", {}),
        "state": data.get("validation_state", None),
    }
    return Checkpoint(
        data["uuid"],
        data["checkpoint_storage"],
        data["batch_number"],
        data["start_time"],
        data["end_time"],
        data["resources"],
        validation,
    )
