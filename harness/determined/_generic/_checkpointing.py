import contextlib
import logging
from typing import Any, Dict, Iterator, Optional, Tuple

import determined as det
from determined import _generic, tensorboard
from determined.common import storage
from determined.common.experimental.session import Session

logger = logging.getLogger("determined.generic")


class Checkpointing:
    """
    Some checkpoint-related REST API wrappers.
    """

    def __init__(
        self,
        dist: _generic.DistributedContext,
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

    @contextlib.contextmanager
    def store_path(self, metadata: Optional[Dict[str, Any]] = None) -> Iterator[Tuple[str, str]]:
        """
        store_path is a context manager which chooses a random path and prepares a directory you
        should save your model to.  When the context manager exits, the model will be automatically
        uploaded (at least, for cloud-backed checkpoint storage backends).

        Note that with multiple workers, only the chief worker (distributed.rank==0) is allowed to
        call store_path.

        Example:

        .. code::

           with checkpointing.store_path() as (uuid, path):
               my_save_model(my_model, path)
               print(f"done saving checkpoint {uuid}")
           print(f"done uploading checkpoint {uuid}")
        """

        if self._dist.rank != 0:
            raise ValueError(
                "cannot call checkpointing.store_path() from non-chief worker "
                f"(rank={self._dist.rank})"
            )

        with self._storage_manager.store_path() as (uuid, path):
            yield uuid, path
            resources = storage.StorageManager._list_directory(path)
        self._report_checkpoint(uuid, resources, metadata)

    @contextlib.contextmanager
    def restore_path(self, storage_id: str) -> Iterator[str]:
        """
        restore_path is a context manager which downloads a checkpoint (if required by the storage
        backend) and cleans it up afterwards (if necessary).

        Note that with multiple workers, all workers must call restore_path, but only the local
        chief worker on each node (distributed.local_rank==0) will actually download data.

        Example:

        .. code::

           with checkpointing.restore_path(my_checkpoint_uuid) as path:
               my_model = my_load_model(path)
        """

        restore_path = self._dist._local_chief_contextmanager(self._storage_manager.restore_path)
        with restore_path(storage_id) as path:
            yield path

    def delete(self, storage_id: str) -> None:
        """
        Delete a checkpoint from the storage backend.
        """
        self._storage_manager.delete(storage_id)

    def _report_checkpoint(
        self,
        uuid: str,
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
            "uuid": uuid,
            "resources": resources,
            **self._static_metadata,
            **metadata,
        }
        logger.debug(f"_report_checkpoint({uuid})")
        self._session.post(self._api_path, data=det.util.json_encode(body))

        # Also sync tensorboard.
        if self._tbd_mgr:
            self._tbd_mgr.sync()


class DummyCheckpointing(Checkpointing):
    def __init__(
        self,
        dist: _generic.DistributedContext,
        storage_manager: storage.StorageManager,
    ) -> None:
        self._dist = dist
        self._storage_manager = storage_manager

    def _report_checkpoint(
        self,
        uuid: str,
        resources: Optional[Dict[str, int]] = None,
        metadata: Optional[Dict[str, Any]] = None,
    ) -> None:
        # No master to report to; just log the event.
        logger.info(f"saved checkpoint {uuid}")
