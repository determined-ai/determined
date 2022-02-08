import os
import time
from typing import Callable, Dict, List, Optional, Tuple

import pytest

from determined import errors
from determined.common import storage
from determined.tensorboard.fetchers import base

EXPECTED_FILES = {
    "root.txt": "root file",
    "subdir/": None,
    "subdir/file.txt": "nested file",
    "empty_dir/": None,
}


def create_checkpoint(checkpoint_dir: str) -> None:
    """Create a new checkpoint."""
    os.makedirs(checkpoint_dir, exist_ok=False)
    for file, content in EXPECTED_FILES.items():
        file = os.path.join(checkpoint_dir, file)
        os.makedirs(os.path.dirname(file), exist_ok=True)
        if content is None:
            continue
        with open(file, "w") as fp:
            fp.write(content)


def validate_checkpoint(checkpoint_dir: str) -> None:
    """Make sure an existing checkpoint looks correct."""
    assert os.path.exists(checkpoint_dir)
    files_found = set(storage.StorageManager._list_directory(checkpoint_dir))
    assert files_found == set(EXPECTED_FILES.keys())
    for found in files_found:
        path = os.path.join(checkpoint_dir, found)
        if EXPECTED_FILES[found] is None:
            assert os.path.isdir(path)
        else:
            assert os.path.isfile(path)
            with open(path) as f:
                assert f.read() == EXPECTED_FILES[found]


def run_storage_lifecycle_test(
    manager: storage.StorageManager,
    post_delete_cb: Optional[Callable] = None,
) -> None:
    checkpoints = []
    for _ in range(5):
        with manager.store_path() as (storage_id, path):
            create_checkpoint(path)
            checkpoints.append(storage_id)

    for storage_id in checkpoints:
        # Load checkpoint.
        with manager.restore_path(storage_id) as path:
            validate_checkpoint(path)
        # Delete.
        manager.delete(storage_id)
        # Ensure it is gone.
        with pytest.raises(errors.CheckpointNotFound):
            with manager.restore_path(storage_id) as path:
                pass
        # Allow for backend-specific inspection.
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
                sleep_time = 2**retry_num
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
