import dataclasses
import enum
import json
import logging
import pathlib
import shutil
import tarfile
from typing import Any, Dict, List, Optional

from determined import errors
from determined.common import api, constants, storage
from determined.common.api import bindings
from determined.common.storage import shared


class DownloadMode(enum.Enum):
    """A list of supported checkpoint download modes."""

    DIRECT = "direct"  # Download directly from checkpoint storage.
    MASTER = "master"  # Proxy download through the master.
    AUTO = "auto"  # Attemp DIRECT and fall back to MASTER.

    def __str__(self) -> str:
        return self.value


class ModelFramework(enum.Enum):
    PYTORCH = 1
    TENSORFLOW = 2


class CheckpointState(enum.Enum):
    UNSPECIFIED = bindings.checkpointv1State.STATE_UNSPECIFIED.value
    ACTIVE = bindings.checkpointv1State.STATE_ACTIVE.value
    COMPLETED = bindings.checkpointv1State.STATE_COMPLETED.value
    ERROR = bindings.checkpointv1State.STATE_ERROR.value
    DELETED = bindings.checkpointv1State.STATE_DELETED.value


@dataclasses.dataclass
class CheckpointTrainingMetadata:
    experiment_config: Dict[str, Any]
    experiment_id: int
    trial_id: int
    hparams: Dict[str, Any]
    validation_metrics: Dict[str, Any]

    @classmethod
    def _from_bindings(
        cls, tm: bindings.v1CheckpointTrainingMetadata
    ) -> "Optional[CheckpointTrainingMetadata]":
        if not tm.trialId:
            return None
        assert tm.experimentConfig
        assert tm.experimentId

        return cls(
            experiment_config=tm.experimentConfig,
            experiment_id=tm.experimentId,
            trial_id=tm.trialId,
            hparams=tm.hparams or {},
            validation_metrics=tm.validationMetrics and tm.validationMetrics.to_json() or {},
        )


class Checkpoint:
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
        session: api.Session,
        task_id: Optional[str],
        allocation_id: Optional[str],
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

    def download(self, path: Optional[str] = None, mode: DownloadMode = DownloadMode.AUTO) -> str:
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
            mode (DownloadMode): Mode governs how a checkpoint is downloaded. Refer to
                the definition of DownloadMode for details.
        """
        if self.state != CheckpointState.COMPLETED:
            raise errors.CheckpointStateException(
                "Only COMPLETED checkpoints can be downloaded. "
                f"Checkpoint state: {self.state.value}"
            )
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
            if mode == DownloadMode.DIRECT:
                self._download_direct(checkpoint_storage, local_ckpt_dir)

            elif mode == DownloadMode.MASTER:
                self._download_via_master(self._session, self.uuid, local_ckpt_dir)

            elif mode == DownloadMode.AUTO:
                self._download_auto(checkpoint_storage, local_ckpt_dir)

            else:
                raise ValueError(f"Unknown download mode {mode}")

        # As of v0.18.0, we write metadata.json once at upload time.  Checkpoints uploaded prior to
        # 0.18.0 will not have a metadata.json present.  Unfortunately, checkpoints earlier than
        # 0.17.7 depended on this file existing in order to be loaded.  Therefore, when we detect
        # that the metadata.json file is not present, we write it to make sure those checkpoints can
        # still load.
        metadata_path = local_ckpt_dir.joinpath("metadata.json")
        if not metadata_path.exists():
            self.write_metadata_file(str(metadata_path))

        return str(local_ckpt_dir)

    def _download_auto(
        self, checkpoint_storage: Dict[str, Any], local_ckpt_dir: pathlib.Path
    ) -> None:
        try:
            self._download_direct(checkpoint_storage, local_ckpt_dir)

        except errors.NoDirectStorageAccess:
            if checkpoint_storage["type"] != "s3" and checkpoint_storage["type"] != "gcs":
                raise

            logging.info("Unable to download directly, proxying download through master")
            try:
                self._download_via_master(self._session, self.uuid, local_ckpt_dir)
            except Exception as e:
                raise errors.MultipleDownloadsFailed(
                    "Auto checkpoint download mode was enabled. "
                    "Attempted direct download and proxied download through master "
                    "but they both failed."
                ) from e

    def _download_direct(
        self, checkpoint_storage: Dict[str, Any], local_ckpt_dir: pathlib.Path
    ) -> None:
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
                    "Downloading from Azure, S3 or GCS requires the experiment "
                    "to be configured with Azure, S3 or GCS checkpointing"
                    ", {} found instead".format(checkpoint_storage["type"])
                )

            manager.download(self.uuid, str(local_ckpt_dir))

    @staticmethod
    def _download_via_master(sess: api.Session, uuid: str, local_ckpt_dir: pathlib.Path) -> None:
        """Downloads a checkpoint through the master.
        Arguments:
            sess (api.Session): a session for the download
            uuid (string): the uuid of the checkpoint to be downloaded
            local_ckpt_dir (Path-like): the local directory where the checkpoint is downloaded
        """
        local_ckpt_dir.mkdir(parents=True, exist_ok=True)

        resp = sess.get(f"/checkpoints/{uuid}", headers={"Accept": "application/gzip"}, stream=True)
        if not resp.ok:
            raise errors.ProxiedDownloadFailed(
                "unable to download checkpoint from master:", resp.status_code, resp.reason
            )
        # gunzip and untar. tarfile.open can detect the compression algorithm
        with tarfile.open(fileobj=resp.raw) as tf:
            tf.extractall(local_ckpt_dir)

    def write_metadata_file(self, path: str) -> None:
        """
        Write a file with this Checkpoint's metadata inside of it.

        This is normally executed as part of Checkpoint.download().  However, in the special case
        where you are accessing the checkpoint files directly (not via Checkpoint.download) you may
        use this method directly to obtain the latest metadata.
        """
        with open(path, "w") as f:
            json.dump(self.metadata, f, indent=2)

    def _push_metadata(self) -> None:
        # TODO: in a future version of this REST API, an entire, well-formed Checkpoint object.
        req = bindings.v1PostCheckpointMetadataRequest(
            checkpoint=bindings.v1Checkpoint(
                uuid=self.uuid,
                metadata=self.metadata,
                resources={},
                training=bindings.v1CheckpointTrainingMetadata(),
                state=bindings.checkpointv1State.STATE_UNSPECIFIED,
            ),
        )
        bindings.post_PostCheckpointMetadata(self._session, body=req, checkpoint_uuid=self.uuid)

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

        self._push_metadata()

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

        self._push_metadata()

    def delete(self) -> None:
        """
        Notifies the master of a checkpoint deletion request, which will be handled asynchronously.
        Master will delete checkpoint and all associated data in the checkpoint storage.
        """

        delete_body = bindings.v1DeleteCheckpointsRequest(checkpointUuids=[self.uuid])
        bindings.delete_DeleteCheckpoints(self._session, body=delete_body)
        logging.info(f"Deletion of checkpoint {self.uuid} is in progress.")

    def __repr__(self) -> str:
        if self.training is not None:
            return (
                f"Checkpoint(uuid={self.uuid}, task_id={self.task_id},"
                f" trial_id={self.training.trial_id})"
            )
        else:
            return f"Checkpoint(uuid={self.uuid}, task_id={self.task_id})"

    @classmethod
    def _from_bindings(cls, ckpt: bindings.v1Checkpoint, session: api.Session) -> "Checkpoint":
        return cls(
            session=session,
            task_id=ckpt.taskId,
            allocation_id=ckpt.allocationId,
            uuid=ckpt.uuid,
            report_time=ckpt.reportTime,
            resources=ckpt.resources,
            metadata=ckpt.metadata,
            state=CheckpointState(ckpt.state.value),
            training=CheckpointTrainingMetadata._from_bindings(ckpt.training),
        )
