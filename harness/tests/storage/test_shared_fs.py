import os
import unittest.mock
from pathlib import Path

import pytest

from determined.common import check, constants, storage
from determined.common.storage import shared
from tests.storage import util

CONTAINER_PATH = constants.SHARED_FS_CONTAINER_PATH


@pytest.fixture()
def manager(tmp_path: Path) -> storage.StorageManager:
    return storage.FileStorageManager(str(tmp_path))


def test_full_storage_path() -> None:
    with pytest.raises(check.CheckFailedError, match="must be an absolute path"):
        shared._full_storage_path(True, "host_path")

    path = shared._full_storage_path(False, "/host_path")
    assert path == "/host_path"

    path = shared._full_storage_path(True, "/host_path")
    assert path == constants.SHARED_FS_CONTAINER_PATH

    path = shared._full_storage_path(False, "/host_path", "storage_path")
    assert path == "/host_path/storage_path"

    path = shared._full_storage_path(True, "/host_path", "storage_path")
    assert path == f"{constants.SHARED_FS_CONTAINER_PATH}/storage_path"

    path = shared._full_storage_path(False, "/host_path", "/host_path/storage_path")
    assert path == "/host_path/storage_path"

    path = shared._full_storage_path(True, "/host_path", "/host_path/storage_path")
    assert path == f"{constants.SHARED_FS_CONTAINER_PATH}/storage_path"

    with pytest.raises(check.CheckFailedError, match="must be a subdirectory"):
        shared._full_storage_path(True, "/host_path", "/storage_path")

    with pytest.raises(check.CheckFailedError, match="must be a subdirectory"):
        shared._full_storage_path(True, "/host_path", "/host_path/../test")

    with pytest.raises(check.CheckFailedError, match="must be a subdirectory"):
        shared._full_storage_path(True, "/host_path", "../test")


def test_checkpoint_lifecycle(manager: storage.StorageManager) -> None:
    def post_delete_cb(storage_id: str) -> None:
        assert storage_id not in os.listdir(manager._base_path)

    util.run_storage_lifecycle_test(manager, post_delete_cb)


def test_validate(manager: storage.StorageManager) -> None:
    assert len(os.listdir(manager._base_path)) == 0
    storage.validate_manager(manager)
    assert len(os.listdir(manager._base_path)) == 0


def test_validate_read_only_dir(manager: storage.StorageManager) -> None:
    def permission_error(_1: str, _2: str) -> None:
        raise PermissionError("Permission denied")

    with unittest.mock.patch("builtins.open", permission_error):
        assert len(os.listdir(manager._base_path)) == 0
        with pytest.raises(PermissionError, match="Permission denied"):
            storage.validate_manager(manager)
        assert len(os.listdir(manager._base_path)) == 1
