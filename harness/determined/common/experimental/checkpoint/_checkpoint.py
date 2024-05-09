import dataclasses
import enum
import json
import logging
import pathlib
import shutil
import tarfile
import warnings
from typing import Any, Dict, Iterable, List, Optional

from determined import errors
from determined.common import api, constants, storage
from determined.common.api import bindings
from determined.common.experimental import metrics
from determined.common.storage import shared

logger = logging.getLogger("determined.client")


class DownloadMode(enum.Enum):
    """
    A list of supported checkpoint download modes.

    Attributes:
        DIRECT
            Download directly from checkpoint storage.
        MASTER
            Proxy download through the master.
        AUTO
            Attempt DIRECT and fall back to MASTER.
    """

    DIRECT = "direct"
    MASTER = "master"
    AUTO = "auto"

    def __str__(self) -> str:
        return self.value


class ModelFramework(enum.Enum):
    PYTORCH = 1
    TENSORFLOW = 2


class CheckpointState(enum.Enum):
    ACTIVE = bindings.checkpointv1State.ACTIVE.value
    COMPLETED = bindings.checkpointv1State.COMPLETED.value
    ERROR = bindings.checkpointv1State.ERROR.value
    DELETED = bindings.checkpointv1State.DELETED.value
    PARTIALLY_DELETED = bindings.checkpointv1State.PARTIALLY_DELETED.value


class CheckpointOrderBy(enum.Enum):
    """Specifies order of a sorted list of checkpoints.

    This class is deprecated in favor of ``OrderBy`` and will be removed in a future
    release.
    """

    def __getattribute__(self, name: str) -> Any:
        warnings.warn(
            "'CheckpointOrderBy' is deprecated and will be removed in a future "
            "release. Please use 'experimental.OrderBy' instead.",
            FutureWarning,
            stacklevel=1,
        )
        return super().__getattribute__(name)

    ASC = bindings.v1OrderBy.ASC.value
    DESC = bindings.v1OrderBy.DESC.value

    def _to_bindings(self) -> bindings.v1OrderBy:
        return bindings.v1OrderBy(self.value)


class CheckpointSortBy(enum.Enum):
    """Specifies checkpoint parameters that can be used for sorting checkpoints."""

    UUID = bindings.checkpointv1SortBy.UUID.value
    TRIAL_ID = bindings.checkpointv1SortBy.TRIAL_ID.value
    BATCH_NUMBER = bindings.checkpointv1SortBy.BATCH_NUMBER.value
    END_TIME = bindings.checkpointv1SortBy.END_TIME.value
    STATE = bindings.checkpointv1SortBy.STATE.value
    SEARCHER_METRIC = bindings.checkpointv1SortBy.SEARCHER_METRIC.value

    def _to_bindings(self) -> bindings.checkpointv1SortBy:
        return bindings.checkpointv1SortBy(self.value)


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
    A class representing a Checkpoint instance of a trained model.

    A Checkpoint object is usually obtained from
    :func:`determined.experimental.client.get_checkpoint`. This class provides helper functionality
    for downloading checkpoints to local storage and loading checkpoints into memory.

    The :class:`~determined.experimental.client.Trial` class contains methods
    that return instances of this class.

    Attributes:
        session: HTTP request session.
        uuid: UUID of checkpoint in storage.
        task_id: (Mutable, Optional[str]) ID of associated task.
        allocation_id: (Mutable, Optional[str]) ID of associated allocation.
        report_time: (Mutable, Optional[str]) Timestamp checkpoint reported.
        resources: (Mutable, Optional[Dict]) Dictionary of file paths to file sizes in bytes of
            all files in the checkpoint.
        metadata: (Mutable, Optional[Dict]) User-defined metadata associated with the checkpoint.
        state: (Mutable, Optional[CheckpointState]) State of the checkpoint.
        training: (Mutable, Optional[CheckpointTrainingMetadata]) Training-related metadata for
            the checkpoint.

        Note:
            All attributes are cached by default.

            Some attributes are mutable and may be changed by methods that update these values,
            either automatically (eg. :meth:`add_metadata()`) or explicitly with :meth:`reload()`.
    """

    def __init__(
        self,
        session: api.Session,
        uuid: str,
    ):
        self._session = session
        self.uuid = uuid

        self.task_id: Optional[str] = None
        self.allocation_id: Optional[str] = None
        self.report_time: Optional[str] = None
        self.resources: Optional[Dict[str, Any]] = None
        self.metadata: Optional[Dict[str, Any]] = None
        self.state: Optional[CheckpointState] = None
        self.training: Optional[CheckpointTrainingMetadata] = None

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

        Arguments:
            path (string, optional): Top level directory to place the
                checkpoint under. If this parameter is not set, the checkpoint will
                be downloaded to ``checkpoints/<checkpoint_uuid>`` relative to the
                current working directory.
            mode (DownloadMode, optional): Governs how a checkpoint is downloaded. Defaults to
                ``AUTO``.
        """
        if self.state not in [CheckpointState.COMPLETED, CheckpointState.PARTIALLY_DELETED]:
            if self.state is None:
                raise ValueError(
                    "Checkpoint state is unknown. Please call Checkpoint.reload to refresh."
                )
            raise errors.CheckpointStateException(
                "Only COMPLETED or PARTIALLY_DELETED checkpoints can be downloaded. "
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

        except (errors.NoDirectStorageAccess, FileNotFoundError):
            if checkpoint_storage["type"] == "azure":
                raise

            logger.info("Unable to download directly, proxying download through master")
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
            shutil.copytree(str(src_ckpt_dir), str(local_ckpt_dir), dirs_exist_ok=True)
        elif checkpoint_storage["type"] == "directory":
            src_ckpt_dir = pathlib.Path(checkpoint_storage["container_path"], self.uuid)
            if not src_ckpt_dir.exists():
                raise FileNotFoundError(
                    "Checkpoint {} not found in {}. This error could be caused by not having "
                    "the same checkpoint storage directory present on the local machine as the "
                    "task runtime storage configuration.".format(self.uuid, src_ckpt_dir)
                )
            shutil.copytree(str(src_ckpt_dir), str(local_ckpt_dir), dirs_exist_ok=True)
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
        assert self.metadata
        # TODO: in a future version of this REST API, an entire, well-formed Checkpoint object.
        req = bindings.v1PostCheckpointMetadataRequest(
            checkpoint=bindings.v1Checkpoint(
                uuid=self.uuid,
                metadata=self.metadata,
                resources={},
                training=bindings.v1CheckpointTrainingMetadata(),
                state=bindings.checkpointv1State.UNSPECIFIED,
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
        updated_metadata = dict(self.metadata, **metadata) if self.metadata else metadata

        req = _metadata_update_request(self.uuid, updated_metadata)
        bindings.post_PostCheckpointMetadata(self._session, body=req, checkpoint_uuid=self.uuid)

        self.metadata = updated_metadata

    def remove_metadata(self, keys: List[str]) -> None:
        """
        Removes user-defined metadata from the checkpoint. Any top-level keys that
        appear in the ``keys`` list are removed from the checkpoint.

        Warning: this metadata change is not propagated to the checkpoint storage.

        Arguments:
            keys (List[string]): Top-level keys to remove from the checkpoint metadata.
        """

        updated_metadata = dict(self.metadata) if self.metadata else {}
        for key in keys:
            if key in updated_metadata:
                del updated_metadata[key]

        req = _metadata_update_request(self.uuid, updated_metadata)
        bindings.post_PostCheckpointMetadata(self._session, body=req, checkpoint_uuid=self.uuid)

        self.metadata = updated_metadata

    def delete(self) -> None:
        """
        Notifies the master of a checkpoint deletion request, which will be handled asynchronously.
        Master will delete checkpoint and all associated data in the checkpoint storage.
        """

        delete_body = bindings.v1DeleteCheckpointsRequest(checkpointUuids=[self.uuid])
        bindings.delete_DeleteCheckpoints(self._session, body=delete_body)
        logger.info(f"Deletion of checkpoint {self.uuid} is in progress.")

    def remove_files(self, globs: List[str]) -> None:
        """
        Removes any files from the checkpoint in checkpoint storage that match one or more of
        the provided ``globs``. The checkpoint resources and state will be updated in master
        asynchronously to reflect checkpoint storage. If ``globs`` is the empty list then no
        files will be deleted and the resources and state will only be refreshed in master.

        Arguments:
            globs (List[string]): Globs to match checkpoint files against.
        """
        remove_body = bindings.v1CheckpointsRemoveFilesRequest(
            checkpointGlobs=globs,
            checkpointUuids=[self.uuid],
        )
        bindings.post_CheckpointsRemoveFiles(self._session, body=remove_body)

        if len(globs) == 0:
            logger.info(f"Refresh of checkpoint {self.uuid} is in progress.")
        else:
            logger.info(f"Partial deletion of checkpoint {self.uuid} is in progress.")

    def get_metrics(self, group: Optional[str] = None) -> Iterable["metrics.TrialMetrics"]:
        """
        Gets all metrics for a given metric group associated with this checkpoint.
        The checkpoint can be originally associated by calling
        ``core_context.experimental.report_task_using_checkpoint(<CHECKPOINT>)``
        from within a task.

        Arguments:
            group (str, optional): Group name for the metrics (example: "training", "validation").
                All metrics will be returned when querying by "".
        """
        from determined.experimental import metrics

        resp = bindings.get_GetTrialMetricsByCheckpoint(
            session=self._session,
            checkpointUuid=self.uuid,
            trialSourceInfoType=bindings.v1TrialSourceInfoType.INFERENCE,
            metricGroup=group,
        )
        for d in resp.metrics:
            yield metrics.TrialMetrics._from_bindings(d, group)

    def get_pachyderm_commit(self) -> str:
        """Return the Pachyderm commit ID associated with this checkpoint."""
        if not self.training:
            # In the case that Checkpoint was constructed manually, reload to populate attributes.
            self.reload()
        assert self.training  # for mypy

        try:
            exp_conf = self.training.experiment_config
            pachyderm_commit = exp_conf["integrations"]["pachyderm"]["dataset"]["commit"]
            return str(pachyderm_commit)
        except (KeyError, TypeError):
            raise ValueError(
                f"Pachyderm configuration not found for checkpoint {self.uuid}, "
                f"experiment {self.training.experiment_id}"
            )

    def __repr__(self) -> str:
        if self.training is not None:
            return (
                f"Checkpoint(uuid={self.uuid}, task_id={self.task_id},"
                f" trial_id={self.training.trial_id})"
            )
        else:
            return f"Checkpoint(uuid={self.uuid}, task_id={self.task_id})"

    def _hydrate(self, ckpt: bindings.v1Checkpoint) -> None:
        self.task_id = ckpt.taskId
        self.allocation_id = ckpt.allocationId
        self.report_time = ckpt.reportTime
        self.resources = ckpt.resources
        self.metadata = ckpt.metadata
        self.state = CheckpointState(ckpt.state.value)
        self.training = CheckpointTrainingMetadata._from_bindings(ckpt.training)

    def reload(self) -> None:
        """
        Explicit refresh of cached properties.
        """
        resp = bindings.get_GetCheckpoint(
            session=self._session, checkpointUuid=self.uuid
        ).checkpoint
        self._hydrate(resp)

    @classmethod
    def _from_bindings(
        cls, ckpt_bindings: bindings.v1Checkpoint, session: api.Session
    ) -> "Checkpoint":
        ckpt = cls(
            session=session,
            uuid=ckpt_bindings.uuid,
        )

        ckpt._hydrate(ckpt_bindings)
        return ckpt


def _metadata_update_request(
    uuid: str, metadata: Dict[str, Any]
) -> bindings.v1PostCheckpointMetadataRequest:
    """Returns a request for updating checkpoint metadata."""
    # TODO: in a future version of this REST API, an entire, well-formed Checkpoint object.
    return bindings.v1PostCheckpointMetadataRequest(
        checkpoint=bindings.v1Checkpoint(
            uuid=uuid,
            metadata=metadata,
            resources={},
            training=bindings.v1CheckpointTrainingMetadata(),
            state=bindings.checkpointv1State.UNSPECIFIED,
        ),
    )
