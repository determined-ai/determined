import json
import pathlib
import shutil
from typing import Any, Dict, List, Optional

from determined_common import storage


class Checkpoint(object):
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
        if path is not None:
            local_ckpt_dir = pathlib.Path(path)
        else:
            local_ckpt_dir = pathlib.Path("checkpoints", self.uuid)

        potential_metadata_paths = ["metadata.json", "MLmodel"]
        is_ckpt_cached = False
        for metadata_file in potential_metadata_paths:
            maybe_ckpt = local_ckpt_dir.joinpath(metadata_file)
            if maybe_ckpt.exists():
                is_ckpt_cached = True
                break

        if not is_ckpt_cached and self.storage_config["type"] == "shared_fs":
            src_ckpt_dir = self._find_shared_fs_path()
            shutil.copytree(str(src_ckpt_dir), str(local_ckpt_dir))

            return str(local_ckpt_dir)

        if not is_ckpt_cached:
            local_ckpt_dir.mkdir(parents=True, exist_ok=True)
            manager = storage.build(self.storage_config)
            if not isinstance(manager, (storage.S3StorageManager, storage.GCSStorageManager)):
                raise AssertionError(
                    "Downloading from S3 or GCS requires the experiment to be configured with "
                    "S3 or GCS checkpointing, {} found instead".format(self.storage_config["type"])
                )

            metadata = storage.StorageMetadata.from_json(
                {"uuid": self.uuid, "resources": self.resources}
            )
            manager.download(metadata, str(local_ckpt_dir))

        return str(local_ckpt_dir)

    def load(self, path: Optional[str] = None, tags: Optional[List[str]] = None) -> Any:
        ckpt_path = self.download(path)
        return Checkpoint.load_from_path(ckpt_path, tags)

    @staticmethod
    def load_from_path(path: str, tags: Optional[List[str]] = None) -> Any:
        ckpt_dir = pathlib.Path(path)

        if is_mlflow(ckpt_dir):
            from determined.api._load_torch import load_model as load_torch

            return load_torch(ckpt_dir)

        with ckpt_dir.joinpath("metadata.json").open() as f:
            meta = json.load(f)

        if "torch_version" in meta:
            from determined.api._load_torch import load_model as load_torch

            return load_torch(ckpt_dir)

        elif "tensorflow_version" in meta:
            from determined.api._load_tf import load_model as load_tf

            return load_tf(ckpt_dir, tags=tags)

        raise AssertionError("Unknown checkpoint format at {}".format(ckpt_dir))


def is_mlflow(ckpt_dir: pathlib.Path) -> bool:
    return ckpt_dir.joinpath("MLmodel").exists()
