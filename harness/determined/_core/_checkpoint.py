import contextlib
import enum
import logging
import os
import pathlib
import uuid
from typing import Any, Dict, Iterator, Optional, Tuple, Union

import determined as det
from determined import _core, tensorboard
from determined.common import storage
from determined.common.api import bindings
from determined.common.experimental.session import Session

logger = logging.getLogger("determined.core")


class DownloadMode(enum.Enum):
    """
    DownloadMode defines the calling behavior of the .download() and the .restore_path() methods of
    CheckpointContext. Frequently in Determined,

    When mode=LocalWorkersShareDownload (the default), workers on the same physical node (the same
    distributed.cross_rank) will share a single downloaded version of the checkpoint.  On an 8-GPU
    node, this will frequently result in 8x bandwidth savings.  In this mode, all workers must call
    .download() or .restore_path() in-step.

    When mode=NoSharedDownload, no coordination is done.  This is useful if you either have
    configured your own coordination, or if only a single worker needs a particular checkpoint.
    There is no in-step calling requirement.
    """

    LocalWorkersShareDownload = "LOCAL_WORKERS_SHARE_DOWNLOAD"
    NoSharedDownload = "NO_SHARED_DOWNLOAD"


class CheckpointContext:
    """
    CheckpointContext gives access to checkpoint-related features of a Determined cluster.
    """

    def __init__(
        self,
        dist: _core.DistributedContext,
        storage_manager: storage.StorageManager,
        session: Session,
        api_path: str,
        static_metadata: Optional[Dict[str, Any]] = None,
        tbd_mgr: Optional[tensorboard.TensorboardManager] = None,
    ) -> None:
        self._dist = dist
        self._storage_manager = storage_manager
        self._session = session
        self._static_metadata = static_metadata or {}
        self._static_metadata["determined_version"] = det.__version__
        self._api_path = api_path
        self._tbd_mgr = tbd_mgr

    def upload(
        self, ckpt_dir: Union[str, os.PathLike], metadata: Optional[Dict[str, Any]] = None
    ) -> str:
        """
        upload() chooses a random storage_id, then uploads the contents of ckpt_dir to checkpoint
        storage into a directory by the name of the storage_id.  The name of the ckpt_dir will not
        be preserved.

        Note that with multiple workers, only the chief worker (distributed.rank==0) is allowed to
        call upload.

        Returns:  The storage_id for this checkpoint.
        """

        if self._dist.rank != 0:
            raise RuntimeError(
                "cannot call CheckpointContext.upload() from non-chief worker "
                f"(rank={self._dist.rank})"
            )

        ckpt_dir = os.fspath(ckpt_dir)

        storage_id = str(uuid.uuid4())
        self._storage_manager.upload(src=ckpt_dir, dst=storage_id)
        resources = self._storage_manager._list_directory(ckpt_dir)
        self._report_checkpoint(storage_id, resources, metadata)
        return storage_id

    def download(
        self,
        storage_id: str,
        ckpt_dir: Union[str, os.PathLike],
        download_mode: DownloadMode = DownloadMode.LocalWorkersShareDownload,
    ) -> None:
        """
        Download the contents of a checkpoint from checkpoint storage into a directory specified by
        ckpt_dir, which will be created if it does not exist.

        .. note::

            This .download() method is similar to but less flexible than the .download() method of
            the :class:`~determined.experiment.common.Checkpoint` class in the Determined Python
            SDK.  This .download() is here as a convenience.
        """
        ckpt_dir = os.fspath(ckpt_dir)
        download_mode = DownloadMode(download_mode)

        if download_mode == DownloadMode.NoSharedDownload:
            self._storage_manager.download(src=storage_id, dst=ckpt_dir)
            return

        # LocalWorkersShareDownload case.
        if self._dist.local_rank == 0:
            self._storage_manager.download(src=storage_id, dst=ckpt_dir)
            # Tell local workers we finished.
            _ = self._dist.broadcast_local(None)
        else:
            # Wait for chief to finish.
            _ = self._dist.broadcast_local(None)

    def get_metadata(self, storage_id: str) -> Dict[str, Any]:
        """
        Returns the current metadata associated with the checkpoint.
        """

        resp = bindings.get_GetCheckpoint(self._session, checkpointUuid=storage_id)
        if not resp.checkpoint or not resp.checkpoint.metadata:
            return {}
        return resp.checkpoint.metadata

    @contextlib.contextmanager
    def store_path(
        self, metadata: Optional[Dict[str, Any]] = None
    ) -> Iterator[Tuple[pathlib.Path, str]]:
        """
        store_path is a context manager which chooses a random path and prepares a directory you
        should save your model to.  When the context manager exits, the model will be automatically
        uploaded (at least, for cloud-backed checkpoint storage backends).

        Note that with multiple workers, only the chief worker (distributed.rank==0) is allowed to
        call store_path.

        Example:

        .. code::

           with core_context.checkpoint.store_path() as (path, storage_id):
               my_save_model(my_model, path)
               print(f"done saving checkpoint {storage_id}")
           print(f"done uploading checkpoint {storage_id}")
        """

        if self._dist.rank != 0:
            raise RuntimeError(
                "cannot call CheckpointContext.store_path() from non-chief worker "
                f"(rank={self._dist.rank})"
            )

        storage_id = str(uuid.uuid4())
        with self._storage_manager.store_path(storage_id) as path:
            yield path, storage_id
            resources = self._storage_manager._list_directory(path)
        self._report_checkpoint(storage_id, resources, metadata)

    @contextlib.contextmanager
    def restore_path(
        self,
        storage_id: str,
        download_mode: DownloadMode = DownloadMode.LocalWorkersShareDownload,
    ) -> Iterator[pathlib.Path]:
        """
        restore_path is a context manager which downloads a checkpoint (if required by the storage
        backend) and cleans up the temporary files afterwards (if applicable).

        In multi-worker scenarios, with the default download_mode (LocalWorkersShareDownload),
        all workers must call restore_path, but only the local chief worker on each node
        (distributed.local_rank==0) will actually download data.

        Example:

        .. code::

           with core_context.checkpoint.restore_path(my_checkpoint_uuid) as path:
               my_model = my_load_model(path)
        """
        download_mode = DownloadMode(download_mode)

        if download_mode == DownloadMode.NoSharedDownload:
            with self._storage_manager.restore_path(storage_id) as path:
                yield path
            return

        # LocalWorkersShareDownload case.
        if self._dist.local_rank == 0:
            with self._storage_manager.restore_path(storage_id) as path:
                # Broadcast to local workers.
                _ = self._dist.broadcast_local(path)
                try:
                    yield path
                finally:
                    # Wait for local workers to finish.
                    _ = self._dist.gather_local(None)
        else:
            # Wait for local chief to broadcast.
            path = self._dist.broadcast_local(None)
            try:
                yield path
            finally:
                # Tell local chief we're done.
                _ = self._dist.gather_local(None)

    def delete(self, storage_id: str) -> None:
        """
        Delete a checkpoint from the storage backend.
        """
        self._storage_manager.delete(storage_id)

    def _report_checkpoint(
        self,
        storage_id: str,
        resources: Optional[Dict[str, int]] = None,
        metadata: Optional[Dict[str, Any]] = None,
    ) -> None:
        """
        After having uploaded a checkpoint, report its existence to the master.
        """

        resources = resources or {}
        metadata = metadata or {}
        required = {"latest_batch"}
        allowed = required.union({"framework", "format", "total_records", "total_epochs"})
        missing = [k for k in required if k not in metadata]
        extra = [k for k in metadata.keys() if k not in allowed]
        if missing:
            raise ValueError(
                "metadata for reported checkpoints, in the current implementation, requires all of "
                f"the following items that have not been provided: {missing}"
            )
        if extra:
            raise ValueError(
                "metadata for reported checkpoints, in the current implementation, cannot support "
                f"the following items that were provided: {extra}"
            )

        body = {
            "uuid": storage_id,
            "resources": resources,
            **self._static_metadata,
            **metadata,
        }
        logger.info(f"Reported checkpoint to master {storage_id}")
        self._session.post(self._api_path, data=det.util.json_encode(body))

        # Also sync tensorboard.
        if self._tbd_mgr:
            self._tbd_mgr.sync()


class DummyCheckpointContext(CheckpointContext):
    def __init__(
        self,
        dist: _core.DistributedContext,
        storage_manager: storage.StorageManager,
    ) -> None:
        self._dist = dist
        self._storage_manager = storage_manager

    def _report_checkpoint(
        self,
        storage_id: str,
        resources: Optional[Dict[str, int]] = None,
        metadata: Optional[Dict[str, Any]] = None,
    ) -> None:
        # No master to report to; just log the event.
        logger.info(f"saved checkpoint {storage_id}")

    def get_metadata(self, storage_id: str) -> Dict[str, Any]:
        # TODO: when the StorageManager supports downloading with a file filter, we should attempt
        # to download metadata.json from the checkpoint and read it here.
        raise NotImplementedError(
            "DummyCheckpointContext is not able to read metadata from checkpoint storage yet."
        )
