import os
import shutil
import unittest.mock
import uuid
from pathlib import Path
from typing import Any, Dict, List

import pytest

from determined.common import check, storage
from determined.common.storage import shared
from determined.tensorboard.fetchers.shared import SharedFSFetcher
from tests import parallel
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


def test_checkpoint_lifecycle(caplog: Any, manager: storage.SharedFSStorageManager) -> None:
    def post_delete_cb(storage_id: str) -> None:
        assert storage_id not in os.listdir(manager._base_path)

    util.run_storage_lifecycle_test(manager, post_delete_cb, caplog)


def test_validate(manager: storage.SharedFSStorageManager) -> None:
    assert len(os.listdir(manager._base_path)) == 0
    storage.validate_manager(manager)
    assert len(os.listdir(manager._base_path)) == 0


def test_validate_read_only_dir(manager: storage.SharedFSStorageManager) -> None:
    def permission_error(_: Any, __: str) -> None:
        raise PermissionError("Permission denied")

    with unittest.mock.patch("pathlib.Path.open", permission_error):
        with unittest.mock.patch("builtins.open", permission_error):
            assert len(os.listdir(manager._base_path)) == 0
            with pytest.raises(PermissionError, match="Permission denied"):
                storage.validate_manager(manager)
            assert len(os.listdir(manager._base_path)) == 1


@pytest.mark.cloud
def test_tensorboard_fetcher_shared(require_secrets: bool, tmp_path: Path) -> None:

    local_sync_dir = os.path.join(tmp_path, "sync_dir")
    storage_dir = os.path.join(tmp_path, "storage_dir")
    storage_relpath = local_sync_dir

    # Create two paths as multi-trial sync could happen.
    paths_to_sync = [
        os.path.join(storage_dir, "test_dir", str(uuid.uuid4()), "subdir") for _ in range(2)
    ]

    fetcher = SharedFSFetcher({}, paths_to_sync, local_sync_dir)

    def put_files(filepath_content: Dict[str, bytes]) -> None:
        for filepath, content in filepath_content.items():
            full_path = os.path.join(storage_dir, filepath)
            os.makedirs(os.path.dirname(full_path), exist_ok=True)
            with open(full_path, "wb") as f:
                f.write(content)

    def rm_files(filepaths: List[str]) -> None:
        for filepath in filepaths:
            full_path = os.path.join(storage_dir, filepath)
            os.remove(full_path)

    util.run_tensorboard_fetcher_test(local_sync_dir, fetcher, storage_relpath, put_files, rm_files)


def clean_up(storage_id: str, storage_manager: storage.S3StorageManager) -> None:
    shutil.rmtree(os.path.join(storage_manager._base_path, storage_id))
    assert storage_id not in os.listdir(storage_manager._base_path)


def test_checkpoint_sharded_upload_download(
    tmp_path: Path, manager: storage.SharedFSStorageManager
) -> None:

    with parallel.Execution(4, local_size=2) as pex:

        @pex.run
        def do_test() -> None:
            util.run_storage_upload_download_sharded_test(pex, manager, tmp_path, clean_up)


def test_checkpoint_sharded_store_restore(manager: storage.SharedFSStorageManager) -> None:

    with parallel.Execution(4, local_size=2) as pex:

        @pex.run
        def do_test() -> None:
            util.run_storage_store_restore_sharded_test(pex, manager, clean_up)


def test_copytree(tmp_path: Path, manager: storage.SharedFSStorageManager) -> None:
    src_dir = tmp_path.joinpath("src")
    util.create_checkpoint(src_dir, util.EXPECTED_FILES)

    dst_dir = tmp_path.joinpath("dst0")
    shared.copytree(str(src_dir), str(dst_dir), selector=None)
    util.validate_checkpoint(dst_dir, expected_files=util.EXPECTED_FILES)

    dst_dir = tmp_path.joinpath("dst1")

    def selector1(x: str) -> bool:
        return False

    shared.copytree(str(src_dir), str(dst_dir), selector=selector1)
    assert not dst_dir.exists()

    dst_dir = tmp_path.joinpath("dst2")

    def selector2(x: str) -> bool:
        return True

    shared.copytree(str(src_dir), str(dst_dir), selector=selector2)
    util.validate_checkpoint(dst_dir, expected_files=util.EXPECTED_FILES)

    dst_dir = tmp_path.joinpath("dst3")

    def selector3(x: str) -> bool:
        return x == "empty_dir/"

    shared.copytree(str(src_dir), str(dst_dir), selector=selector3)
    util.validate_checkpoint(dst_dir, expected_files={"empty_dir/": None})

    dst_dir = tmp_path.joinpath("dst4")

    def selector4(x: str) -> bool:
        return x in ["root.txt", "subdir/"]

    shared.copytree(str(src_dir), str(dst_dir), selector=selector4)
    util.validate_checkpoint(dst_dir, expected_files={"root.txt": "root file", "subdir/": None})

    dst_dir = tmp_path.joinpath("dst5")

    def selector5(x: str) -> bool:
        return x == "subdir/file2.txt"

    shared.copytree(str(src_dir), str(dst_dir), selector=selector5)
    util.validate_checkpoint(
        dst_dir, expected_files={"subdir/file2.txt": "nested file 2", "subdir/": None}
    )

    # Test copytree with symlinks.
    src_dir1 = tmp_path.joinpath("src1")
    util.create_checkpoint(src_dir1, {"file_1": "content 1"})

    src_dir2 = tmp_path.joinpath("src2")
    util.create_checkpoint(
        src_dir2,
        {"symlink_dir/": None, "symlink_dir/file_nested": "nested file", "file_2": "content 2"},
    )

    os.symlink(
        src_dir2.joinpath("symlink_dir"), src_dir1.joinpath("subdir"), target_is_directory=True
    )

    dst_dir = tmp_path.joinpath("dst6")
    shared.copytree(str(src_dir1), str(dst_dir))

    util.validate_checkpoint(
        dst_dir,
        expected_files={
            "file_1": "content 1",
            "subdir/": None,
            "subdir/file_nested": "nested file",
        },
    )
