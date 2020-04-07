import os
import unittest.mock
from pathlib import Path

import pytest

import determined_common.storage
from determined_common.check import CheckFailedError
from determined_common.storage import SharedFSStorageManager
from tests.unit.storage import StorableFixture


@pytest.fixture()  # type: ignore
def manager(tmp_path: Path) -> SharedFSStorageManager:
    return SharedFSStorageManager(str(tmp_path), str(tmp_path))


@pytest.fixture()  # type: ignore
def checkpoint() -> StorableFixture:
    return StorableFixture()


def test_container_relative_path() -> None:
    with pytest.raises(CheckFailedError, match="`host_path` must be an " "absolute path"):
        SharedFSStorageManager("host_path", "container_path")
    with pytest.raises(CheckFailedError, match="`container_path` must be an " "absolute path"):
        SharedFSStorageManager("/host_path", "container_path")


def test_full_checkpoint_dir() -> None:
    manager = SharedFSStorageManager("/host_path", "/container_path")
    assert manager.container_path == "/container_path"
    assert manager._base_path == "/container_path"

    manager = SharedFSStorageManager("/host_path", "/container_path", "storage_path")
    assert manager.container_path == "/container_path"
    assert manager._base_path == "/container_path/storage_path"

    manager = SharedFSStorageManager("/host_path", "/container_path/", "storage_path")
    assert manager.container_path == "/container_path/"
    assert manager._base_path == "/container_path/storage_path"

    manager = SharedFSStorageManager("/host_path", storage_path="storage_path")
    assert manager.container_path == "/determined_shared_fs"
    assert manager._base_path == "/determined_shared_fs/storage_path"

    manager = SharedFSStorageManager("/host_path", storage_path="/host_path/storage_path")
    assert manager.container_path == "/determined_shared_fs"
    assert manager._base_path == "/determined_shared_fs/storage_path"

    manager = SharedFSStorageManager("/host_path/", storage_path="/host_path/storage_path")
    assert manager.container_path == "/determined_shared_fs"
    assert manager._base_path == "/determined_shared_fs/storage_path"

    with pytest.raises(CheckFailedError, match="must be a subdirectory"):
        SharedFSStorageManager("host_path", storage_path="/storage_path")

    with pytest.raises(CheckFailedError, match="must be a subdirectory"):
        SharedFSStorageManager("/host_path", storage_path="/storage_path")

    with pytest.raises(CheckFailedError, match="must be a subdirectory"):
        SharedFSStorageManager("/host_path", storage_path="/host_path/../test")

    with pytest.raises(CheckFailedError, match="must be a subdirectory"):
        SharedFSStorageManager("/host_path", storage_path="../test")


def test_checkpoint_lifecycle(manager: SharedFSStorageManager, checkpoint: StorableFixture) -> None:
    assert len(os.listdir(manager.container_path)) == 0

    checkpoints = []
    for _ in range(5):
        metadata = manager.store(checkpoint)
        checkpoints.append(metadata)
        assert set(metadata.resources) == set(checkpoint.expected_files)
        manager.restore(checkpoint, metadata)

    assert len(os.listdir(manager.container_path)) == 5

    for index in reversed(range(5)):
        metadata = checkpoints[index]
        assert metadata.storage_id in os.listdir(manager.container_path)
        manager.delete(metadata)
        assert metadata.storage_id not in os.listdir(manager.container_path)
        assert len(os.listdir(manager.container_path)) == index


def test_verify(tmp_path: Path) -> None:
    tmpdir_s = str(tmp_path)
    config = {"type": "shared_fs", "host_path": tmpdir_s, "container_path": tmpdir_s}
    assert len(os.listdir(tmpdir_s)) == 0
    determined_common.storage.validate(config)
    assert len(os.listdir(tmpdir_s)) == 0


def test_verify_read_only_dir(tmp_path: Path) -> None:
    def permission_error(_1: str, _2: str) -> None:
        raise PermissionError("Permission denied")

    tmpdir_s = str(tmp_path)
    config = {"type": "shared_fs", "host_path": tmpdir_s, "container_path": tmpdir_s}

    with unittest.mock.patch("builtins.open", permission_error):
        assert len(os.listdir(tmpdir_s)) == 0
        with pytest.raises(PermissionError, match="Permission denied"):
            determined_common.storage.validate(config)
        assert len(os.listdir(tmpdir_s)) == 1
