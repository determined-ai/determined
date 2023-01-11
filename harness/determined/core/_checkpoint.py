import contextlib
import enum
import hashlib
import json
import logging
import os
import pathlib
import uuid
from datetime import datetime, timezone
from typing import Any, Callable, Dict, Iterator, List, Optional, Set, Tuple, Union

from determined import core, tensorboard
from determined.common import api, storage
from determined.common.api import bindings

logger = logging.getLogger("determined.core")


class DownloadMode(enum.Enum):
    """
    ``DownloadMode`` defines the calling behavior of the .download() and the .restore_path() methods
    of ``CheckpointContext``.

    When mode is ``LocalWorkersShareDownload`` (the default), workers on the same physical node (the
    same ``distributed.cross_rank``) will share a single downloaded version of the checkpoint.  On
    an 8-GPU node, this will frequently result in 8x bandwidth savings.  In this mode, all workers
    must call ``.download()`` or ``.restore_path()`` in step.

    When mode is ``NoSharedDownload``, no coordination is done.  This is useful if you either have
    configured your own coordination, or if only a single worker needs a particular checkpoint.
    There is no in-step calling requirement.
    """

    LocalWorkersShareDownload = "LOCAL_WORKERS_SHARE_DOWNLOAD"
    NoSharedDownload = "NO_SHARED_DOWNLOAD"


def _merge_metadata(
    merged: Dict[str, Any],
    rank_metadata: Dict[str, Any],
    rank: int,
    key_ranks: Dict[str, Any],
    key_conflicts: Dict[str, List[int]],
    prev_key: str,
) -> Tuple[Dict[str, Any], Dict[str, Any]]:

    for key, metadata in rank_metadata.items():

        # First, update key_ranks. We will use it later
        if key not in key_ranks:
            key_ranks[key] = {"_ranks_": []}
        key_ranks[key]["_ranks_"].append(rank)

        full_key = f"{prev_key}/{key}"

        # Merge metadata. We consider two possibilities:
        # 1. metadata is dictionary AND :
        #   * if key is not in merged, then recursively unroll and merge metadata; OR
        #   * if key is in merged, and metadata and merged[key] are dictionaries, merge recursively.
        # 2. otherwise:
        #   * key is in merged, then updated key_conflicts and don't merge,
        #   * key is not in merged, then add key and metadata to merged.
        if (key not in merged and isinstance(metadata, dict)) or (
            key in merged and isinstance(metadata, dict) and isinstance(merged[key], dict)
        ):
            merged[key], key_ranks[key] = _merge_metadata(
                {} if key not in merged else merged[key],
                metadata,
                rank,
                key_ranks[key],
                key_conflicts,
                full_key,
            )
        else:
            if key in merged and merged[key] != metadata:
                if full_key not in key_conflicts:
                    key_conflicts[full_key] = key_ranks[key]["_ranks_"].copy()
                else:
                    key_conflicts[full_key].append(rank)
            else:
                merged[key] = metadata

    return merged, key_ranks


def merge_metadata(
    all_metadata: List[Dict[str, Any]]
) -> Tuple[Dict[str, Any], Dict[str, List[int]]]:
    """
    Given a list of metadata, return:
      - merged metadata
      - a dict mapping conflicting keys to ranks that would upload them

    Merging scenarios:
      - merge dictionaries under the same key
      - report conflicts for any repeated key that is not a dictionary

    Examples:
    #  Metadata 1     | Metadata 2       | Conflict |  Result
    # ----------------|------------------|----------|-----------------
    #  a: 1           | b: 2             | no       | {a: 1, b: 2}
    #  a: 1           | a: 1             | no       | {a: 1}
    #  a: [1]         | a: [2]           | yes      | n/a
    #  a: [1]         | a: [1]           | no       | {a: [1]}
    #  a: []          | a: 1             | yes      | n/a
    #  a: []          | a: {c: 1}        | yes      | n/a
    #  a: {c: 1}      | a: {d: 1}        | no       | a: {c: 1, d: 1}
    #  a: {c: 1}      | a: 1             | yes      | n/a
    """

    # Stores merged metadata.
    merged: Dict[str, Any] = {}

    # Maps keys to reporting ranks. It has the same structure as merged with each key
    # having a list of ranks that reports it. Helps to detect conflicts.
    key_ranks: Dict[str, Any] = {}

    # Maps fullpath keys (from root) to the list of rank reporting it.
    key_conflicts: Dict[str, List[int]] = {}

    for rank, rank_metadata in enumerate(all_metadata):
        merged, key_ranks = _merge_metadata(
            merged, rank_metadata, rank, key_ranks, key_conflicts, ""
        )

    return merged, key_conflicts


def merge_resources(
    all_resources: List[Dict[str, int]]
) -> Tuple[Dict[str, int], Dict[str, List[int]]]:
    """
    Given a list of all resources, return:
      - a merged list of resources
      - a dict mapping conflicting files to ranks that would upload them

    Note that we allow multiple ranks to upload directories, but only one rank may upload any
    given file.
    """
    files: Set[str] = set()
    uploaders: Dict[str, List] = {}
    merged: Dict[str, int] = {}
    for rank, rscs in enumerate(all_resources):

        for name in rscs:
            size = rscs[name]
            if name.endswith(os.sep):
                # Dir name.
                stripped = name.rstrip(os.sep)
                uploaders.setdefault(stripped, []).append(rank)
            else:
                # File name.
                files.add(name)
                uploaders.setdefault(name, []).append(rank)

            merged[name] = size

    # Overlapping name situations:
    #
    #  A uploads |  B uploads | Conflict |  Detection
    # -----------|------------|----------|-------------------------------
    #  dir       |  dir       | no       |  n/a
    #  dir       |  file      | yes      |  len(uploaders[name]) > 1
    #  file      |  file      | yes      |  len(uploaders[name]) > 1
    #
    # Conclusion: all conflicts can be detected by checking the names in `files`.
    conflicts = {}
    for name in files:
        uploading_ranks = uploaders[name]
        if len(uploading_ranks) > 1:
            conflicts[name] = uploading_ranks

    return merged, conflicts


class CheckpointContext:
    """
    ``CheckpointContext`` gives access to checkpoint-related features of a Determined cluster.
    """

    def __init__(
        self,
        dist: core.DistributedContext,
        storage_manager: storage.StorageManager,
        session: api.Session,
        task_id: str,
        allocation_id: str,
        tbd_sync_mode: core.TensorboardMode,
        tensorboard_manager: tensorboard.TensorboardManager,
    ) -> None:
        self._dist = dist
        self._storage_manager = storage_manager
        self._session = session
        self._task_id = task_id
        self._allocation_id = allocation_id
        self._tensorboard_mode = tbd_sync_mode
        self._tensorboard_manager = tensorboard_manager

    def upload(
        self,
        ckpt_dir: Optional[Union[str, os.PathLike]],
        metadata: Optional[Dict[str, Any]] = None,
        *,
        shard: bool = False,
    ) -> str:
        """
        ``upload()`` chooses a random ``storage_id``, then uploads the contents of ``ckpt_dir`` to
        checkpoint storage into a directory by the name of the ``storage_id``.  The name of the
        ``ckpt_dir`` is not preserved.

        When ``shard=False``, only the chief worker (``distributed.rank==0``) may call ``upload()``.

        When ``shard=True``, ``upload()`` becomes a synchronization point between workers, so all
        workers must call upload().  Those workers with nothing to upload may pass
        ``ckpt_dir=None``.  The final checkpoint stored in checkpoint storage will contain a union
        of the contents from each ckpt_dir.

        Returns:  The ``storage_id`` for this checkpoint.

        Example:

        .. code::

           if core_context.distributed.rank == 0:
               storage_id = core_context.checkpoint.upload(ckpt_dir, shard=False)
               print(f"done uploading checkpoint {storage_id}")
        """
        if ckpt_dir is not None:
            ckpt_dir = os.fspath(ckpt_dir)

        # The simple and sharded cases can technically be written as one function but it becomes
        # far more complicated than having two codepaths.
        if not shard:
            if self._dist.rank != 0:
                raise RuntimeError(
                    f"cannot call .upload(shard=False) from non-chief worker "
                    f"(rank={self._dist.rank})"
                )
            if ckpt_dir is None:
                raise RuntimeError(
                    "cannot call .upload(ckpt_dir=None, shard=False), which would result in doing "
                    "nothing at all"
                )
            return self._upload_single(ckpt_dir, metadata)
        else:
            storage_id = None
            if self._dist.rank == 0:
                storage_id = str(uuid.uuid4())
            storage_id = self._dist.broadcast(storage_id)

            assert storage_id
            return self._upload_sharded(ckpt_dir, storage_id, metadata)

    def _upload_single(self, ckpt_dir: str, metadata: Optional[Dict[str, Any]] = None) -> str:
        storage_id = str(uuid.uuid4())
        resources = self._storage_manager._list_directory(ckpt_dir)

        # Add metadata pre-upload but without counting it among resources.
        self._write_metadata_file(ckpt_dir, metadata or {})

        self._storage_manager.upload(src=ckpt_dir, dst=storage_id)
        self._report_checkpoint(storage_id, resources, metadata)
        return storage_id

    def _upload_sharded(
        self, ckpt_dir: Optional[str], storage_id: str, metadata: Optional[Dict[str, Any]] = None
    ) -> str:
        ckpt_dir_mask = self._dist.allgather(ckpt_dir is not None)
        if not any(ckpt_dir_mask):
            raise RuntimeError(
                "cannot call .upload(ckpt_dir=None, shard=True), from all ranks; "
                "at least one rank must have a valid ckpt_dir"
            )

        # Deconflict locally-shared directories; if every worker uploads /tmp/ckpt, then only
        # the lowest rank on each node will actually upload this directory.
        if ckpt_dir is None:
            file_uid = None
        else:
            st = os.stat(ckpt_dir)
            file_uid = (st.st_dev, st.st_ino)
        all_file_uids = self._dist.allgather(file_uid)
        # Decide if our rank is the lowest rank trying to upload this ckpt_dir.
        want_upload = file_uid and all_file_uids.index(file_uid) == self._dist.rank

        # Decide what we are going to upload.
        if want_upload:
            assert ckpt_dir
            resources = self._storage_manager._list_directory(ckpt_dir)
        else:
            resources = {}

        # Merge resources, detect conflicts.
        all_resources = self._dist.allgather(resources)
        merged_resources, conflicts = merge_resources(all_resources)
        if conflicts:
            self._try_resolving_conflicts(ckpt_dir, conflicts)

        # Merge and save merged metadata locally for each rank to avoid conflicts
        # after pausing and unpausing experiment.
        all_metadata = self._merge_and_save_metadata(ckpt_dir, metadata=metadata or {})

        if want_upload:
            assert ckpt_dir
            self._storage_manager.upload(src=ckpt_dir, dst=storage_id)

        # Synchronize workers.
        _ = self._dist.allgather(None)
        if self._dist.rank == 0:
            self._report_checkpoint(storage_id, merged_resources, all_metadata)

        return storage_id

    def _try_resolving_conflicts(
        self, ckpt_dir: Optional[str], conflicts: Dict[str, List[int]]
    ) -> None:
        all_conflicts = conflicts.copy()

        for fname in conflicts:
            ranks = conflicts[fname]
            if self._dist.rank in ranks:
                assert ckpt_dir
                md5 = hashlib.md5(open(os.path.join(ckpt_dir, fname), "rb").read()).hexdigest()
                md5_ranks = self._dist.allgather(md5)
            else:
                md5_ranks = self._dist.allgather(None)

            md5_ranks = [x for x in md5_ranks if x is not None]

            if len(set(md5_ranks)) == 1:
                # All files have the same md5 checksum, which means there is no conflict.
                all_conflicts.pop(fname)

        if len(all_conflicts) > 0:
            self._print_conflict_error(all_conflicts, "files")

    def _print_conflict_error(self, conflicts: Dict[str, List], conflict_dtype: str) -> None:
        # Try to keep the logs easier to read; print the whole failure only on the chief.
        if self._dist.rank > 0:
            raise RuntimeError(f"refusing to upload with {conflict_dtype} conflicts: {conflicts}")
        msgs = [f"    {f} uploaded by ranks {ranks}" for f, ranks in sorted(conflicts.items())]

        raise RuntimeError(
            f"refusing to upload with {conflict_dtype} conflicts:\n" + "\n".join(msgs)
        )

    def download(
        self,
        storage_id: str,
        ckpt_dir: Union[str, os.PathLike],
        download_mode: DownloadMode = DownloadMode.LocalWorkersShareDownload,
        *,
        selector: Optional[Callable[[str], bool]] = None,
    ) -> None:
        """
        Download the contents of a checkpoint from checkpoint storage into a directory specified by
        ``ckpt_dir``, which is created if it does not exist.

        .. note::

            This ``.download()`` method is similar to but less flexible than the ``.download()``
            method of the :class:`~determined.experiment.common.Checkpoint` class in the Determined
            Python SDK.  This ``.download()`` is here as a convenience.
        """
        ckpt_dir = os.fspath(ckpt_dir)
        download_mode = DownloadMode(download_mode)

        if download_mode == DownloadMode.NoSharedDownload:
            self._storage_manager.download(src=storage_id, dst=ckpt_dir, selector=selector)
            return

        want_filter = any(self._dist.allgather(selector is not None))

        # LocalWorkersShareDownload case.
        if self._dist.local_rank == 0:

            def _selector(path: str) -> bool:
                if not want_filter:
                    return True
                # If anyone has a selector, coordinate every filename across workers' filters.
                # Functions can't be reliably serialized, so instead we pass each filename between
                # all workers. But because this traffic is local (unix sockets by default) it
                # should be far faster than any download.
                _ = self._dist.broadcast_local(path)
                # If selector is None return True => all files will be downloaded.
                upload_path = self._dist.gather_local(
                    selector(path) if selector is not None else True
                )
                # Upload_path is not None when we are on the local chief.
                assert upload_path
                return any(upload_path)

            self._storage_manager.download(src=storage_id, dst=ckpt_dir, selector=_selector)
            # Tell local workers we finished.
            _ = self._dist.broadcast_local(None)
        else:
            while True:
                name = self._dist.broadcast_local(None)
                if name is None:
                    # Chief is done downloading files.
                    break
                assert want_filter, "want_filter is not set but name was not None"
                _ = self._dist.gather_local(selector(name) if selector is not None else True)

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
        self, metadata: Optional[Dict[str, Any]] = None, *, shard: bool = False
    ) -> Iterator[Tuple[pathlib.Path, str]]:
        """
        ``store_path()`` is a context manager which chooses a random path and prepares a directory
        you should save your model to.  When the context manager exits, the model is automatically
        uploaded (at least, for cloud-backed checkpoint storage backends).

        When ``shard=False``, only the chief worker (``distributed.rank==0``) may call
        ``store_path()``.

        When ``shard=True``, ``store_path()`` becomes a synchronization point between workers, so
        all workers must call store_path(), even workers which will not write any checkpoint files.

        Example:

        .. code::

           if core_context.distributed.rank == 0:
               with core_context.checkpoint.store_path(shard=False) as (path, storage_id):
                   my_save_model(my_model, path)
                   print(f"done saving checkpoint {storage_id}")
               print(f"done uploading checkpoint {storage_id}")
        """
        if not shard:
            return self._store_path_single(metadata)
        else:
            return self._store_path_sharded(metadata)

    def _store_path_single(
        self, metadata: Optional[Dict[str, Any]] = None
    ) -> Iterator[Tuple[pathlib.Path, str]]:
        if self._dist.rank != 0:
            raise RuntimeError(
                "cannot call CheckpointContext.store_path(shard=False) from non-chief worker "
                f"(rank={self._dist.rank})"
            )

        storage_id = str(uuid.uuid4())
        with self._storage_manager.store_path(storage_id) as path:
            yield path, storage_id
            resources = self._storage_manager._list_directory(path)
            self._write_metadata_file(os.fspath(path), metadata or {})

        self._report_checkpoint(storage_id, resources, metadata)

    def _store_path_sharded(
        self, metadata: Optional[Dict[str, Any]] = None
    ) -> Iterator[Tuple[pathlib.Path, str]]:
        storage_id = None
        if self._dist.rank == 0:
            storage_id = str(uuid.uuid4())
        storage_id = self._dist.broadcast(storage_id)
        assert storage_id

        path = self._storage_manager.pre_store_path(storage_id)
        yield path, storage_id

        ckpt_dir = os.fspath(path)

        if self._storage_manager.store_path_is_direct_access():
            # Each rank saves files directly to ckpt_dir which means there is no conflict
            # detection on upload. Metadata still needs to be merged and saved,
            # and checkpoint has to be reported.
            if self._dist.rank == 0:
                resources = self._storage_manager._list_directory(ckpt_dir)

            all_metadata = self._merge_and_save_metadata(ckpt_dir, metadata=metadata or {})

            if self._dist.rank == 0:
                self._report_checkpoint(storage_id, resources, all_metadata)

            return

        # Deconflict locally-shared directories; if every worker uploads /tmp/ckpt, then only
        # the lowest rank on each node will actually upload this directory.
        st = os.stat(ckpt_dir)
        file_uid = (st.st_dev, st.st_ino)
        all_file_uids = self._dist.allgather(file_uid)
        # Decide if our rank is the lowest rank trying to upload this ckpt_dir.
        want_upload = all_file_uids.index(file_uid) == self._dist.rank

        # Decide what we are going to upload.
        if want_upload:
            assert ckpt_dir
            resources = self._storage_manager._list_directory(ckpt_dir)
        else:
            resources = {}

        # Merge resources, detect conflicts.
        all_resources = self._dist.allgather(resources)

        merged_resources, conflicts = merge_resources(all_resources)
        if conflicts:
            self._try_resolving_conflicts(ckpt_dir, conflicts)

        # Merge and save merged metadata locally for each rank to avoid conflicts
        # after pausing and unpausing experiment.
        all_metadata = self._merge_and_save_metadata(ckpt_dir, metadata=metadata or {})

        if want_upload:
            # Use post_store_path to upload and clean up ckpt_dir after uploading.
            self._storage_manager.post_store_path(src=ckpt_dir, dst=storage_id)

        if self._dist.rank == 0:
            self._report_checkpoint(storage_id, merged_resources, all_metadata)

        # Synchronize workers.
        _ = self._dist.allgather(None)

        return storage_id

    def _merge_and_save_metadata(
        self,
        ckpt_dir: Optional[str],
        metadata: Optional[Dict[str, Any]],
    ) -> Dict[str, Any]:
        # Gather metadata across nodes.
        all_metadata = self._dist.allgather(metadata or {})
        # Merge metadata and report errors when the same keys have different values.
        merged_metadata, conflicts = merge_metadata(all_metadata)
        if conflicts:
            self._print_conflict_error(conflicts, "metadata")
        if ckpt_dir is not None and self._dist.local_rank == 0:
            self._write_metadata_file(ckpt_dir, merged_metadata)
        return merged_metadata

    @contextlib.contextmanager
    def restore_path(
        self,
        storage_id: str,
        download_mode: DownloadMode = DownloadMode.LocalWorkersShareDownload,
        *,
        selector: Optional[Callable[[str], bool]] = None,
    ) -> Iterator[pathlib.Path]:
        """
        ``restore_path()`` is a context manager which downloads a checkpoint (if required by the
        storage backend) and cleans up the temporary files afterwards (if applicable).

        In multi-worker scenarios, with the default ``download_mode``
        (``LocalWorkersShareDownload``), all workers must call ``restore_path()`` but only the local
        chief worker on each node (``distributed.local_rank==0``) actually downloads data.

        Example:

        .. code::

           with core_context.checkpoint.restore_path(my_checkpoint_uuid) as path:
               my_model = my_load_model(path)
        """
        download_mode = DownloadMode(download_mode)

        if download_mode == DownloadMode.NoSharedDownload:
            with self._storage_manager.restore_path(storage_id, selector=selector) as path:
                yield path
            return

        want_filter = any(self._dist.allgather(selector is not None))

        # LocalWorkersShareDownload case.
        if self._dist.local_rank == 0:

            def _selector(path: str) -> bool:
                if not want_filter:
                    return True
                # If anyone has a selector, coordinate every filename across workers' filters.
                # Functions can't be reliably serialized, so instead we pass each filename between
                # all workers. But because this traffic is local (unix sockets by default) it
                # should be far faster than any download.
                _ = self._dist.broadcast_local(path)
                # If selector is None return True => all files will be downloaded.
                upload_path = self._dist.gather_local(
                    selector(path) if selector is not None else True
                )
                # Upload_path is not None when we are on the local chief.
                assert upload_path
                return any(upload_path)

            with self._storage_manager.restore_path(storage_id, _selector) as path:
                # Tell local workers that download is finished.
                _ = self._dist.broadcast_local(None)
                # Broadcast to local workers.
                _ = self._dist.broadcast_local(path)
                try:
                    yield path
                finally:
                    # Wait for local workers to finish.
                    _ = self._dist.gather_local(None)
        else:
            while True:
                name = self._dist.broadcast_local(None)
                if name is None:
                    # Chief is done downloading files.
                    break
                assert want_filter, "want_filter is not set but name was not None"
                _ = self._dist.gather_local(selector(name) if selector is not None else True)

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

    def _write_metadata_file(self, ckpt_dir: str, metadata: Dict[str, Any]) -> None:
        metadata_path = pathlib.Path(ckpt_dir).joinpath("metadata.json")
        with metadata_path.open("w") as f:
            json.dump(metadata, f, indent=2)

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

        if "steps_completed" not in metadata:
            raise ValueError(
                "metadata for reported checkpoints, in the current implementation, requires a "
                "'steps_completed' item, which has not been provided"
            )

        ckpt = bindings.v1Checkpoint(
            allocationId=self._allocation_id,
            metadata=metadata,
            resources={k: str(v) for k, v in resources.items()},
            taskId=self._task_id,
            training=bindings.v1CheckpointTrainingMetadata(),
            uuid=storage_id,
            reportTime=datetime.now(timezone.utc).isoformat(),
            state=bindings.determinedcheckpointv1State.STATE_COMPLETED,
        )
        bindings.post_ReportCheckpoint(self._session, body=ckpt)
        logger.info(f"Reported checkpoint to master {storage_id}")

        # Also sync tensorboard.
        if self._tensorboard_mode == core.TensorboardMode.AUTO:
            self._tensorboard_manager.sync()


class DummyCheckpointContext(CheckpointContext):
    def __init__(
        self,
        dist: core.DistributedContext,
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
