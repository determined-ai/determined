import os
from typing import Optional
from unittest import mock

import pytest

from determined import core
from determined.common import storage


def test_unknown_type() -> None:
    config = {"type": "unknown"}
    with pytest.raises(TypeError, match="Unknown storage type: unknown"):
        storage.build(config, container_path=None)


def test_missing_type() -> None:
    with pytest.raises(ValueError, match="Missing 'type' parameter"):
        storage.build({}, container_path=None)


def test_illegal_type() -> None:
    config = {"type": 4}
    with pytest.raises(ValueError, match="must be a string"):
        storage.build(config, container_path=None)


def test_build_with_container_path() -> None:
    config = {"type": "shared_fs", "host_path": "/host_path", "storage_path": "storage_path"}
    manager = storage.build(config, container_path=None)
    assert manager._base_path == "/host_path/storage_path"
    manager = storage.build(config, container_path="/container_path")
    assert manager._base_path == "/container_path/storage_path"


def test_list_directory() -> None:
    root = os.path.join(os.path.dirname(__file__), "fixtures")

    assert set(storage.StorageManager._list_directory(root)) == {
        "root.txt",
        "nested/",
        "nested/nested.txt",
        "nested/another.txt",
    }


def test_list_directory_on_file() -> None:
    root = os.path.join(os.path.dirname(__file__), "fixtures", "root.txt")
    assert os.path.exists(root)
    with pytest.raises(NotADirectoryError, match=root):
        storage.StorageManager._list_directory(root)


def test_list_nonexistent_directory() -> None:
    root = "./non-existent-directory"
    assert not os.path.exists(root)
    with pytest.raises(FileNotFoundError, match=root):
        storage.StorageManager._list_directory(root)


@pytest.mark.parametrize(
    "prefix",
    ["", "myprefix"],
    ids=["gs://<bucket>", "gs://<bucket>/<prefix>"],
)
def test_gcs_shortcut_string(prefix: Optional[str]) -> None:
    bucket = "mybucket"
    shortcut = f"gs://{bucket}"
    if prefix:
        shortcut += f"/{prefix}"
    with mock.patch("determined.common.storage.GCSStorageManager") as mocked:
        _ = storage.from_string(shortcut)
    mocked.assert_called_once_with(bucket=bucket, prefix=prefix)


@pytest.mark.parametrize(
    "prefix",
    ["", "myprefix"],
    ids=["s3://<bucket>", "s3://<bucket>/<prefix>"],
)
def test_s3_shortcut_string(prefix: Optional[str]) -> None:
    bucket = "mybucket"
    shortcut = f"s3://{bucket}"
    if prefix:
        shortcut += f"/{prefix}"
    with mock.patch("determined.common.storage.S3StorageManager") as mocked:
        _ = storage.from_string(shortcut)
    mocked.assert_called_once_with(bucket=bucket, prefix=prefix)


def test_shared_fs_shortcut_string() -> None:
    shortcut = "/tmp/somewhere"
    with mock.patch("determined.common.storage.SharedFSStorageManager") as mocked:
        _ = storage.from_string(shortcut)
    mocked.assert_called_once_with(base_path=shortcut)


@pytest.mark.parametrize(
    "shortcut",
    [
        "scheme://bucket/prefix;parameters?query#fragment",
        "scheme://bucket/prefix?query&otherquery",
        "scheme://bucket/prefix#fragment",
        "scheme://bucket/prefix;parameters?query#fragment",
        "file://localhost:1234/a/b/c",
        "localhost:1234/a/b/c",
    ],
)
def test_bad_shortcut_string(shortcut: str) -> None:
    with pytest.raises(ValueError):
        _ = storage.from_string(shortcut)


def test_azure_shortcut_dict() -> None:
    shortcut = {"type": "azure", "container": "test_container", "account_url": "localhost"}
    with mock.patch("determined.common.storage.AzureStorageManager.from_config") as mocked:
        _ = core._context._get_storage_manager(checkpoint_storage=shortcut)
    shortcut.pop("type")
    mocked.assert_called_once_with(shortcut, None)


def test_gcs_shortcut_dict() -> None:
    shortcut = {"type": "gcs", "bucket": "test_bucket"}
    with mock.patch("determined.common.storage.GCSStorageManager.from_config") as mocked:
        _ = core._context._get_storage_manager(checkpoint_storage=shortcut)
    shortcut.pop("type")
    mocked.assert_called_once_with(shortcut, None)


def test_s3_shortcut_dict() -> None:
    shortcut = {"type": "s3", "bucket": "test_bucket"}
    with mock.patch("determined.common.storage.S3StorageManager.from_config") as mocked:
        _ = core._context._get_storage_manager(checkpoint_storage=shortcut)
    shortcut.pop("type")
    mocked.assert_called_once_with(shortcut, None)


def test_shared_fs_shortcut_dict() -> None:
    shortcut = {"type": "shared_fs", "base_path": "test_base_path"}
    with pytest.raises(ValueError):
        _ = core._context._get_storage_manager(checkpoint_storage=shortcut)
