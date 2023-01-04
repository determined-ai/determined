import logging
import os
import pathlib
import shutil
import time
import uuid
from typing import Any, Callable, Dict, List, Optional, Tuple
from unittest import mock

import pytest
import requests

from determined import core, errors
from determined.common import storage
from determined.tensorboard.fetchers import base
from tests import parallel

EXPECTED_FILES = {
    "root.txt": "root file",
    "subdir/": None,
    "subdir/file1.txt": "nested file 1",
    "subdir/file2.txt": "nested file 2",
    "empty_dir/": None,
}

# File structure for sharded checkpoint testing
EXPECTED_FILES_N0 = {
    "file0_0": "file 0 node 0",
    "file_repeated": "file repeated",
    "file1_0": "file 1 node 0",
    "subdir/": None,
    "subdir/file3_0": "nested file node 0",
    "subdir/file_repeated": "nested file repeated",
    "metadata.json": '{\n  "steps_completed": 1\n}',
}
EXPECTED_FILES_N1 = {
    "file0_1": "file 0 node 1",
    "file1_1": "file 1 node 1",
    "file_repeated1": "file repeated 1",
    "subdir/": None,
    "subdir/file3_1": "nested file node 1",
    "subdir/file_repeated": "nested file repeated",
    "metadata.json": '{\n  "steps_completed": 1\n}',
}


def create_checkpoint(checkpoint_dir: pathlib.Path, expected_files: Optional[Dict] = None) -> None:
    """Create a new checkpoint."""
    if expected_files is None:
        expected_files = EXPECTED_FILES
    for file, content in expected_files.items():
        path = checkpoint_dir.joinpath(file)
        path.parent.mkdir(parents=True, exist_ok=True)
        if content is None:
            path.mkdir(exist_ok=True)
            continue
        with path.open("w") as f:
            f.write(content)


def validate_checkpoint(checkpoint_dir: pathlib.Path, expected_files: Dict) -> None:
    """Make sure an existing checkpoint looks correct."""
    assert checkpoint_dir.exists()
    files_found = set(storage.StorageManager._list_directory(checkpoint_dir))
    assert files_found == set(expected_files.keys()), (files_found, expected_files)
    for found in files_found:
        path = checkpoint_dir.joinpath(found)
        if expected_files[found] is None:
            assert path.is_dir(), path
        else:
            assert path.is_file(), path
            with path.open() as f:
                text = f.read()
                assert text == expected_files[found], (text, expected_files[found])


def sync_and_clean(
    pex: parallel.Execution,
    clean_up: Optional[Callable],
    storage_id: str,
    storage_manager: storage.StorageManager,
) -> None:
    pex.distributed.allgather(None)
    if pex.distributed.rank == 0 and clean_up is not None:
        clean_up(storage_id, storage_manager)


def run_storage_lifecycle_test(
    manager: storage.StorageManager,
    post_delete_cb: Optional[Callable] = None,
    caplog: Any = None,
) -> None:
    checkpoints = []
    for _ in range(2):
        storage_id = str(uuid.uuid4())
        with manager.store_path(storage_id) as path:
            create_checkpoint(path)
            checkpoints.append(storage_id)

    for storage_id in checkpoints:
        # Load checkpoint.
        with manager.restore_path(storage_id) as path:
            validate_checkpoint(path, EXPECTED_FILES)
        # Delete.
        manager.delete(storage_id)
        # Ensure it is gone.
        with pytest.raises(errors.CheckpointNotFound):
            with manager.restore_path(storage_id) as path:
                pass
        # Allow for backend-specific inspection.
        if post_delete_cb is not None:
            post_delete_cb(storage_id)

    # Again, using upload/download instead of store_path/restore_path.
    checkpoints = []
    for _ in range(2):
        storage_id = str(uuid.uuid4())
        path = pathlib.Path(f"/tmp/storage_lifecycle_test-{storage_id}")
        try:
            create_checkpoint(path)
            manager.upload(path, storage_id)
            checkpoints.append(storage_id)
        finally:
            shutil.rmtree(path, ignore_errors=True)

    for storage_id in checkpoints:
        path = pathlib.Path(f"/tmp/storage_lifecycle_test-{storage_id}")
        try:
            manager.download(storage_id, path)
            validate_checkpoint(path, EXPECTED_FILES)
        finally:
            shutil.rmtree(path, ignore_errors=True)
        manager.delete(storage_id)
        with pytest.raises(errors.CheckpointNotFound):
            manager.download(storage_id, path)
        if post_delete_cb is not None:
            post_delete_cb(storage_id)

    # Upload checkpoint and test restore_path/download with selector.
    checkpoints = []
    for _ in range(2):
        storage_id = str(uuid.uuid4())
        with manager.store_path(storage_id) as path:
            create_checkpoint(path)
            checkpoints.append(storage_id)

    expected_files_subset = {
        "subdir/": None,
        "subdir/file1.txt": "nested file 1",
        "empty_dir/": None,
    }

    def selector(x: str) -> bool:
        return x in ["subdir/file1.txt", "empty_dir/"]

    # Test restore_path with selector
    # clear logs collected up to this point
    if caplog is not None:
        caplog.clear()
    for storage_id in checkpoints:
        with manager.restore_path(storage_id, selector=selector) as path:
            if isinstance(manager, storage.shared.SharedFSStorageManager):
                assert caplog
                assert (
                    caplog.messages[0] == "Ignoring partial checkpoint download from shared_fs;"
                    " all files will be directly accessible from shared_fs."
                )
                validate_checkpoint(path, EXPECTED_FILES)
            else:
                validate_checkpoint(path, expected_files_subset)

    # Test download with selector
    for storage_id in checkpoints:
        path = pathlib.Path(f"/tmp/storage_lifecycle_test-{storage_id}")
        try:
            manager.download(storage_id, path, selector=selector)
            validate_checkpoint(path, expected_files_subset)
            manager.delete(storage_id)
            with pytest.raises(errors.CheckpointNotFound):
                manager.download(storage_id, path)
            if post_delete_cb is not None:
                post_delete_cb(storage_id)
        finally:
            shutil.rmtree(path, ignore_errors=True)

    # Test upload with selector.
    checkpoints = []
    for _ in range(2):
        storage_id = str(uuid.uuid4())
        path = pathlib.Path(f"/tmp/storage_lifecycle_test-{storage_id}")
        try:
            create_checkpoint(path)
            manager.upload(path, storage_id, paths=["subdir/file1.txt", "empty_dir/"])
            checkpoints.append(storage_id)
        finally:
            shutil.rmtree(path, ignore_errors=True)

    expected_files_subset = {
        "subdir/": None,
        "subdir/file1.txt": "nested file 1",
        "empty_dir/": None,
    }

    for storage_id in checkpoints:
        path = pathlib.Path(f"/tmp/storage_lifecycle_test-{storage_id}")
        try:
            manager.download(storage_id, path)
            validate_checkpoint(path, expected_files_subset)
            manager.delete(storage_id)
            with pytest.raises(errors.CheckpointNotFound):
                manager.download(storage_id, path)
            if post_delete_cb is not None:
                post_delete_cb(storage_id)
        finally:
            shutil.rmtree(path, ignore_errors=True)


def run_tensorboard_fetcher_test(
    local_sync_dir: str,
    fetcher: base.Fetcher,
    storage_relpath: str,
    put_files: Callable,
    rm_files: Callable,
) -> None:
    def list_files(path: str) -> List[str]:
        return [os.path.join(root, file) for root, _, files in os.walk(path) for file in files]

    def get_filepath_dict(file_tups: List[Tuple[str, bytes]]) -> Dict[str, bytes]:
        return {
            os.path.join(storage_path, file_tup[0]): file_tup[1]
            for file_tup in file_tups
            for storage_path in fetcher.storage_paths
        }

    def verify_files(expected_files: Dict[str, bytes]) -> None:
        # SharedFS has absolute paths
        expected_files = {k.lstrip("/"): v for k, v in expected_files.items()}

        full_paths = list_files(local_sync_dir)
        local_files = [os.path.relpath(fp, storage_relpath) for fp in full_paths]
        for local_file in local_files:
            expected_content = expected_files.get(local_file)
            assert expected_content is not None

            with open(os.path.join(storage_relpath, local_file), "rb") as f:
                observed_content = f.read()
                if not expected_content == observed_content:
                    raise AssertionError(
                        "Expected: '{!r}', Observed: '{!r}'".format(
                            expected_content, observed_content
                        )
                    )
                expected_files.pop(local_file)

        missing_files = len(expected_files)
        if missing_files != 0:
            raise AssertionError(
                f"There were {missing_files} missing files: {expected_files.keys()}"
            )

    def timed_backoff_sync_check(expected_files: Dict[str, bytes]) -> None:
        # Prevent failures due storage propagation delay.
        num_retries = 5
        for retry_num in range(num_retries + 1):
            try:
                fetcher.fetch_new()
                verify_files(expected_files)
                break
            except AssertionError as e:
                if retry_num == num_retries:
                    raise
                sleep_time = 2 ** retry_num
                print(f"{e}: sleeping: {sleep_time} seconds")
                time.sleep(sleep_time)

    files_to_remove = []  # type: List[str]

    try:
        # (Empty Sync) Ensure empty sync is ok
        fetcher.fetch_new()
        local_file_list = list_files(local_sync_dir)
        assert len(local_file_list) == 0

        # (New Sync) Upload a set of files, sync, ensure files downloaded.
        first_files = get_filepath_dict([("foo", b"foo"), ("bar", b"bar")])
        put_files(first_files)
        files_to_remove.extend(first_files.keys())

        timed_backoff_sync_check(first_files)
        time.sleep(1)  # racy storage propagation with Azure

        # (Update Sync) Upload new content to same set of files, sync, ensure local files updated.
        second_files = get_filepath_dict([("foo", b"foo2"), ("bar", b"bar2")])
        put_files(second_files)

        # Prevent failures due storage propagation delay.
        timed_backoff_sync_check(second_files)

        # (Additional Sync) Upload a new set of files, sync, ensure new files downloaded.
        third_files = get_filepath_dict([("baz", b"baz"), ("qux", b"qux")])
        put_files(third_files)
        files_to_remove.extend(third_files.keys())

        timed_backoff_sync_check({**second_files, **third_files})
    finally:
        rm_files(files_to_remove)


def run_storage_upload_download_sharded_test(
    pex: parallel.Execution,
    storage_manager: storage.StorageManager,
    tmp_path: pathlib.Path,
    clean_up: Optional[Callable] = None,
) -> None:
    checkpoint_context = create_checkpoint_context(pex, storage_manager)
    selector: Optional[Callable[[str], bool]]  # Making lint happy.

    # Create "local" file structure.
    ckpt_dir = tmp_path.joinpath(f"ckpt_dir_{pex.distributed.rank}")
    if pex.distributed.rank == 0:
        create_checkpoint(ckpt_dir, EXPECTED_FILES_N0)
    else:
        create_checkpoint(ckpt_dir, EXPECTED_FILES_N1)

    # Wait for all ranks to save files.
    pex.distributed.allgather(None)

    logging.info(
        f"Rank {pex.distributed.rank}. "
        f"Files in ckpt_dir: "
        f"{[os.path.join(dp, f) for dp, dn, fn in os.walk(ckpt_dir) for f in fn]}"
    )

    metadata = {"steps_completed": 1}

    # Upload sharded data.
    storage_id = checkpoint_context.upload(ckpt_dir, metadata, shard=True)

    # 1. Test all sharded files were uploaded to ckpt.
    download_dir = tmp_path.joinpath(f"test1_download{pex.distributed.rank}")
    try:
        checkpoint_context.download(storage_id, download_dir)
        validate_checkpoint(download_dir, expected_files={**EXPECTED_FILES_N0, **EXPECTED_FILES_N1})
    finally:
        shutil.rmtree(download_dir, ignore_errors=True)

    # 2. Test downloading with selector: every rank gets selected files
    download_dir = tmp_path.joinpath(f"test2_download_{pex.distributed.rank}")
    if pex.distributed.rank == 0:

        def selector(x: str) -> bool:
            return x == "subdir/file3_0"

    else:

        def selector(x: str) -> bool:
            return x == "file1_1"

    try:
        checkpoint_context.download(storage_id, download_dir, selector=selector)

        if pex.distributed.rank == 0:
            validate_checkpoint(
                download_dir,
                expected_files={"subdir/file3_0": "nested file node 0", "subdir/": None},
            )
        else:
            validate_checkpoint(download_dir, expected_files={"file1_1": "file 1 node 1"})
    finally:
        shutil.rmtree(download_dir, ignore_errors=True)

    # 3.Test downloading with and w/o selector.
    download_dir = tmp_path.joinpath(f"test3_download_{pex.distributed.rank}")
    if pex.distributed.rank == 0:

        def selector(x: str) -> bool:
            return x in EXPECTED_FILES_N0

    else:
        selector = None

    try:
        checkpoint_context.download(storage_id, download_dir, selector=selector)
        if pex.distributed.rank == 0:
            validate_checkpoint(download_dir, expected_files=EXPECTED_FILES_N0)
        else:
            validate_checkpoint(
                download_dir, expected_files={**EXPECTED_FILES_N0, **EXPECTED_FILES_N1}
            )
    finally:
        shutil.rmtree(download_dir, ignore_errors=True)

    sync_and_clean(pex, clean_up, storage_id, storage_manager)

    # 4. Upload sharded data from rank 1 only.
    # Metadata should be saved and uploaded as well.
    storage_id = checkpoint_context.upload(
        ckpt_dir if pex.distributed.rank == 1 else None, metadata, shard=True
    )
    download_dir = tmp_path.joinpath(f"test4_download_{pex.distributed.rank}")
    checkpoint_context.download(storage_id, download_dir)
    validate_checkpoint(
        download_dir,
        expected_files={**EXPECTED_FILES_N1, **{"metadata.json": '{\n  "steps_completed": 1\n}'}},
    )
    sync_and_clean(pex, clean_up, storage_id, storage_manager)

    # 5. Test uploading selected paths: rank 0 uploads a subset of files,
    # rank 1 uploads all files.
    if pex.distributed.rank == 0:

        def selector(x: str) -> bool:
            return x in x in ["file_repeated", "file0_0", "subdir/file_repeated"]

    else:
        selector = None

    storage_id = checkpoint_context.upload(ckpt_dir, metadata, shard=True, selector=selector)

    download_dir = tmp_path.joinpath(f"test5_download_{pex.distributed.rank}")
    checkpoint_context.download(storage_id, download_dir)
    validate_checkpoint(
        download_dir,
        expected_files={
            **EXPECTED_FILES_N1,
            **{
                "file_repeated": "file repeated",
                "file0_0": "file 0 node 0",
                "subdir/": None,
                "subdir/file_repeated": "nested file repeated",
            },
        },
    )
    sync_and_clean(pex, clean_up, storage_id, storage_manager)

    # 6. Test uploading selected paths: ranks try to upload not existing paths.
    def selector_upload(x: str) -> bool:
        return x == "there_is_no_file_with_such_name"

    storage_id = checkpoint_context.upload(ckpt_dir, metadata, shard=True, selector=selector_upload)

    download_dir = tmp_path.joinpath(f"test6_download_{pex.distributed.rank}")
    checkpoint_context.download(storage_id, download_dir)
    validate_checkpoint(
        download_dir,
        expected_files={"metadata.json": '{\n  "steps_completed": 1\n}'},
    )
    sync_and_clean(pex, clean_up, storage_id, storage_manager)

    # 7. Test uploading two files with the same name and different content
    # raises an error.
    with parallel.raises_when(
        True,
        RuntimeError,
        match=r"refusing to upload with files conflicts:.*",
    ):
        if pex.distributed.rank == 0:
            create_checkpoint(ckpt_dir, {"filename": "content 1"})
        else:
            create_checkpoint(ckpt_dir, {"filename": "content 2"})

        pex.distributed.allgather(None)
        checkpoint_context.upload(ckpt_dir, metadata, shard=True)


def run_storage_store_restore_sharded_test(
    pex: parallel.Execution,
    storage_manager: storage.StorageManager,
    clean_up: Optional[Callable] = None,
) -> None:
    checkpoint_context = create_checkpoint_context(pex, storage_manager)
    selector: Optional[Callable[[str], bool]]  # Making lint happy.

    metadata = {"steps_completed": 1}

    with checkpoint_context.store_path(metadata, shard=True) as (path, storage_id):
        if pex.distributed.rank == 0:
            create_checkpoint(path, EXPECTED_FILES_N0)
        else:
            create_checkpoint(path, EXPECTED_FILES_N1)

    # 1. Test all sharded files were uploaded to ckpt.
    with checkpoint_context.restore_path(storage_id) as path:
        validate_checkpoint(path, expected_files={**EXPECTED_FILES_N0, **EXPECTED_FILES_N1})
        # Make sure all ranks finish validating checkpoint before exiting the context.
        pex.distributed.allgather(None)

    if isinstance(storage_manager, storage.shared.SharedFSStorageManager):
        # SharedFS does not support downloading partial checkpoints.
        # Skip all remaining tests.
        return

    # 2. Test downloading with selector: every rank gets selected files.
    if pex.distributed.rank == 0:

        def selector(x: str) -> bool:
            return x == "subdir/file3_0"

    else:

        def selector(x: str) -> bool:
            return x == "file1_1"

    with checkpoint_context.restore_path(storage_id, selector=selector) as path:
        if pex.distributed.rank == 0:
            validate_checkpoint(
                path, expected_files={"subdir/file3_0": "nested file node 0", "subdir/": None}
            )
        else:
            validate_checkpoint(path, expected_files={"file1_1": "file 1 node 1"})
        pex.distributed.allgather(None)

    # 3. Test downloading with and w/o selector.
    if pex.distributed.rank == 0:

        def selector(x: str) -> bool:
            return x in list(EXPECTED_FILES_N0.keys())

    else:
        selector = None

    with checkpoint_context.restore_path(storage_id, selector=selector) as path:
        if pex.distributed.rank == 0:
            validate_checkpoint(path, expected_files=EXPECTED_FILES_N0)
        else:
            validate_checkpoint(path, expected_files={**EXPECTED_FILES_N0, **EXPECTED_FILES_N1})
        pex.distributed.allgather(None)

    if pex.distributed.rank == 0 and clean_up is not None:
        clean_up(storage_id, storage_manager)

    # 4. Test uploading two files with the same name and different content
    # raises an error.
    with parallel.raises_when(
        True,
        RuntimeError,
        match=r"refusing to upload with files conflicts:.*",
    ):
        with checkpoint_context.store_path(metadata, shard=True) as (path, storage_id):
            if pex.distributed.rank == 0:
                create_checkpoint(path, {"filename": "content 1"})
            else:
                create_checkpoint(path, {"filename": "content 2"})


def create_checkpoint_context(
    pex: parallel.Execution, storage_manager: storage.StorageManager
) -> core.CheckpointContext:
    session = mock.MagicMock()
    response = requests.Response()
    response.status_code = 200
    session._do_request.return_value = response
    tensorboard_manager = mock.MagicMock()
    checkpoint_context = core.CheckpointContext(
        pex.distributed,
        storage_manager,
        session=session,
        task_id="task-id",
        allocation_id="allocation-id",
        tbd_sync_mode=core.TensorboardMode.AUTO,
        tensorboard_manager=tensorboard_manager,
    )
    return checkpoint_context
