import os
from typing import Optional, Tuple
from unittest import mock

import pytest

from determined.common import storage
from determined.common import check


def test_unknown_type() -> None:
    config = {"type": "unknown"}
    with pytest.raises(TypeError, match="Unknown storage type: unknown"):
        storage.build(config, container_path=None)


def test_missing_type() -> None:
    with pytest.raises(check.CheckFailedError, match="Missing 'type' parameter"):
        storage.build({}, container_path=None)


def test_illegal_type() -> None:
    config = {"type": 4}
    with pytest.raises(check.CheckFailedError, match="must be a string"):
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
    "connection_string",
    [None, "DefaultEndpointsProtocol=https;AccountName=myaccount;AccountKey=myaccountkey"],
)
@pytest.mark.parametrize("account_url", [None, "http://127.0.0.1:8080/"])
@pytest.mark.parametrize("credential", [None, "credential"])
@pytest.mark.parametrize("temp_dir", [None, "mytempdir/mytempsubdir"])
def test_azure_shortcut_string(
    connection_string: Optional[str],
    account_url: Optional[str],
    credential: Optional[str],
    temp_dir: Optional[str],
) -> None:
    container = "mycontainer"
    shortcut = f"ms://{container}"
    if connection_string:
        shortcut += f"/{connection_string}"
    fields = dict(
        account_url=account_url,
        credential=credential,
        temp_dir=temp_dir,
    )
    if fields:
        shortcut += "?{}".format(",".join(["{}={}".format(k, v) for k, v in fields.items() if v]))
    with mock.patch("determined.common.storage.AzureStorageManager") as mocked:
        _ = storage.from_string(shortcut)
    assert mocked.called_once_with(
        container=container,
        connection_string=connection_string,
        account_url=account_url,
        credential=credential,
        temp_dir=temp_dir,
    )


@pytest.mark.parametrize("prefix", [None, "myprefix"])
@pytest.mark.parametrize("temp_dir", [None, "mytempdir/mytempsubdir"])
def test_gcs_shortcut_string(prefix: Optional[str], temp_dir: Optional[str]) -> None:
    bucket = "mybucket"
    shortcut = f"gs://{bucket}"
    if prefix:
        shortcut += f"/{prefix}"
    if temp_dir:
        shortcut += f"?temp_dir={temp_dir}"  # Can be replaced with f"&{temp_dir=}" with Python 3.8
    with mock.patch("determined.common.storage.GCSStorageManager") as mocked:
        _ = storage.from_string(shortcut)
    assert mocked.called_once_with(bucket=bucket, prefix=prefix, temp_dir=temp_dir)


@pytest.mark.parametrize("prefix", [None, "myprefix"])
@pytest.mark.parametrize("keys", [(None, None), ("myaccesskey", "mysecretkey")])
@pytest.mark.parametrize("endpoint_url", [None, "http://127.0.0.1:8080/"])
@pytest.mark.parametrize("temp_dir", [None, "mytempdir/mytempsubdir"])
def test_s3_shortcut_string(
    keys: Optional[Tuple[str, str]],
    endpoint_url: Optional[str],
    prefix: Optional[str],
    temp_dir: Optional[str],
) -> None:
    bucket = "mybucket"
    access_key, secret_key = keys
    shortcut = f"s3://{bucket}"
    if prefix:
        shortcut += f"/{prefix}"
    fields = dict(
        access_key=access_key,
        secret_key=secret_key,
        endpoint_url=endpoint_url,
        temp_dir=temp_dir,
    )
    if fields:
        shortcut += "?{}".format("&".join(["{}={}".format(k, v) for k, v in fields.items() if v]))
    with mock.patch("determined.common.storage.S3StorageManager") as mocked:
        _ = storage.from_string(shortcut)
    assert mocked.called_once_with(
        bucket=bucket,
        prefix=prefix,
        access_key=access_key,
        secret_key=secret_key,
        endpoint_url=endpoint_url,
        temp_dir=temp_dir,
    )


def test_shared_fs_shortcut_string() -> None:
    shortcut = "/tmp/somewhere"
    with mock.patch("determined.common.storage.SharedFSStorageManager") as mocked:
        _ = storage.from_string(shortcut)
    assert mocked.called_once_with(base_path=shortcut)
