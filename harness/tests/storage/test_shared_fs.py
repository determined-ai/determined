import os
import unittest.mock
from pathlib import Path

import pytest

from determined_common import storage
from determined_common.check import CheckFailedError
from tests.storage import util


@pytest.fixture()  # type: ignore
def manager(tmp_path: Path) -> storage.SharedFSStorageManager:
    return storage.SharedFSStorageManager(str(tmp_path), str(tmp_path))


def test_container_relative_path() -> None:
    with pytest.raises(CheckFailedError, match="`host_path` must be an " "absolute path"):
        storage.SharedFSStorageManager("host_path", "container_path")
    with pytest.raises(CheckFailedError, match="`container_path` must be an " "absolute path"):
        storage.SharedFSStorageManager("/host_path", "container_path")


def test_full_checkpoint_dir() -> None:
    manager = storage.SharedFSStorageManager("/host_path", "/container_path")
    assert manager.container_path == "/container_path"
    assert manager._base_path == "/container_path"

    manager = storage.SharedFSStorageManager("/host_path", "/container_path", "storage_path")
    assert manager.container_path == "/container_path"
    assert manager._base_path == "/container_path/storage_path"

    manager = storage.SharedFSStorageManager("/host_path", "/container_path/", "storage_path")
    assert manager.container_path == "/container_path/"
    assert manager._base_path == "/container_path/storage_path"

    manager = storage.SharedFSStorageManager("/host_path", storage_path="storage_path")
    assert manager.container_path == "/determined_shared_fs"
    assert manager._base_path == "/determined_shared_fs/storage_path"

    manager = storage.SharedFSStorageManager("/host_path", storage_path="/host_path/storage_path")
    assert manager.container_path == "/determined_shared_fs"
    assert manager._base_path == "/determined_shared_fs/storage_path"

    manager = storage.SharedFSStorageManager("/host_path/", storage_path="/host_path/storage_path")
    assert manager.container_path == "/determined_shared_fs"
    assert manager._base_path == "/determined_shared_fs/storage_path"

    with pytest.raises(CheckFailedError, match="must be a subdirectory"):
        storage.SharedFSStorageManager("host_path", storage_path="/storage_path")

    with pytest.raises(CheckFailedError, match="must be a subdirectory"):
        storage.SharedFSStorageManager("/host_path", storage_path="/storage_path")

    with pytest.raises(CheckFailedError, match="must be a subdirectory"):
        storage.SharedFSStorageManager("/host_path", storage_path="/host_path/../test")

    with pytest.raises(CheckFailedError, match="must be a subdirectory"):
        storage.SharedFSStorageManager("/host_path", storage_path="../test")


def test_checkpoint_lifecycle(manager: storage.SharedFSStorageManager) -> None:
    assert len(os.listdir(manager._base_path)) == 0

    checkpoints = []
    for index in range(5):
        with manager.store_path() as (storage_id, path):
            # Ensure no checkpoint directories exist yet.
            assert len(os.listdir(manager._base_path)) == index
            util.create_checkpoint(path)
            metadata = storage.StorageMetadata(storage_id, manager._list_directory(path))
            checkpoints.append(metadata)
            assert set(metadata.resources) == set(util.EXPECTED_FILES.keys())

    assert len(os.listdir(manager._base_path)) == 5

    for index in reversed(range(5)):
        metadata = checkpoints[index]
        assert metadata.storage_id in os.listdir(manager._base_path)
        with manager.restore_path(metadata) as path:
            util.validate_checkpoint(path)
        manager.delete(metadata)
        assert metadata.storage_id not in os.listdir(manager._base_path)
        assert len(os.listdir(manager._base_path)) == index


def test_verify(tmp_path: Path) -> None:
    tmpdir_s = str(tmp_path)
    config = {"type": "shared_fs", "host_path": tmpdir_s, "container_path": tmpdir_s}
    assert len(os.listdir(tmpdir_s)) == 0
    storage.validate(config)
    assert len(os.listdir(tmpdir_s)) == 0


def test_verify_read_only_dir(tmp_path: Path) -> None:
    def permission_error(_1: str, _2: str) -> None:
        raise PermissionError("Permission denied")

    tmpdir_s = str(tmp_path)
    config = {"type": "shared_fs", "host_path": tmpdir_s, "container_path": tmpdir_s}

    with unittest.mock.patch("builtins.open", permission_error):
        assert len(os.listdir(tmpdir_s)) == 0
        with pytest.raises(PermissionError, match="Permission denied"):
            storage.validate(config)
        assert len(os.listdir(tmpdir_s)) == 1
