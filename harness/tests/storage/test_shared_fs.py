import os
import unittest.mock
from pathlib import Path

import pytest

from determined.common import check, storage
from determined.common.storage import shared
from tests.storage import util


@pytest.fixture()
def manager(tmp_path: Path) -> storage.SharedFSStorageManager:
    return storage.SharedFSStorageManager(str(tmp_path))


def test_full_storage_path() -> None:
    with pytest.raises(check.CheckFailedError, match="`host_path` must be an absolute path"):
        shared._full_storage_path("host_path")

    path = shared._full_storage_path("/host_path")
    assert path == "/host_path"

    path = shared._full_storage_path("/host_path", container_path="cpath")
    assert path == "cpath"

    path = shared._full_storage_path("/host_path", "storage_path")
    assert path == "/host_path/storage_path"

    path = shared._full_storage_path("/host_path", "storage_path", container_path="cpath")
    assert path == "cpath/storage_path"

    path = shared._full_storage_path("/host_path", storage_path="/host_path/storage_path")
    assert path == "/host_path/storage_path"

    path = shared._full_storage_path("/host_path", "/host_path/storage_path", "cpath")
    assert path == "cpath/storage_path"

    with pytest.raises(check.CheckFailedError, match="must be a subdirectory"):
        shared._full_storage_path("/host_path", storage_path="/storage_path")

    with pytest.raises(check.CheckFailedError, match="must be a subdirectory"):
        shared._full_storage_path("/host_path", storage_path="/host_path/../test")

    with pytest.raises(check.CheckFailedError, match="must be a subdirectory"):
        shared._full_storage_path("/host_path", storage_path="../test")


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


def test_validate(manager: storage.SharedFSStorageManager) -> None:
    assert len(os.listdir(manager._base_path)) == 0
    storage.validate_manager(manager)
    assert len(os.listdir(manager._base_path)) == 0


def test_validate_read_only_dir(manager: storage.SharedFSStorageManager) -> None:
    def permission_error(_1: str, _2: str) -> None:
        raise PermissionError("Permission denied")

    with unittest.mock.patch("builtins.open", permission_error):
        assert len(os.listdir(manager._base_path)) == 0
        with pytest.raises(PermissionError, match="Permission denied"):
            storage.validate_manager(manager)
        assert len(os.listdir(manager._base_path)) == 1
