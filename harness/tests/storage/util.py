import os
import pathlib
import shutil
import time
import uuid
from typing import Any, Callable, Dict, List, Optional, Tuple

import pytest

from determined import errors
from determined.common import storage
from determined.tensorboard.fetchers import base

EXPECTED_FILES = {
    "root.txt": "root file",
    "subdir/": None,
    "subdir/file1.txt": "nested file 1",
    "subdir/file2.txt": "nested file 2",
    "empty_dir/": None,
}


def create_checkpoint(checkpoint_dir: pathlib.Path) -> None:
    """Create a new checkpoint."""
    for file, content in EXPECTED_FILES.items():
        path = checkpoint_dir.joinpath(file)
        path.parent.mkdir(parents=True, exist_ok=True)
        if content is None:
            path.mkdir()
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
                assert f.read() == expected_files[found]


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

    def selector(x):
        return x in ["subdir", "subdir/file1.txt", "empty_dir"]

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
        finally:
            shutil.rmtree(path, ignore_errors=True)

    # Clean up
    for storage_id in checkpoints:
        manager.delete(storage_id)
        with pytest.raises(errors.CheckpointNotFound):
            manager.download(storage_id, path)
        if post_delete_cb is not None:
            post_delete_cb(storage_id)


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
