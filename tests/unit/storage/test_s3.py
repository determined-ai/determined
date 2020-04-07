import os
import tempfile
from pathlib import Path

import pytest
from _pytest.monkeypatch import MonkeyPatch
from boto3.exceptions import S3UploadFailedError

from determined_common import storage
from tests.unit import s3
from tests.unit.storage import StorableFixture


@pytest.fixture  # type: ignore
def manager(tmp_path: Path, monkeypatch: MonkeyPatch) -> storage.S3StorageManager:
    monkeypatch.setattr("boto3.client", s3.s3_client)
    return storage.S3StorageManager(
        bucket="bucket", access_key="key", secret_key="secret", temp_dir=str(tmp_path)
    )


@pytest.fixture()  # type: ignore
def checkpoint() -> StorableFixture:
    return StorableFixture()


def test_remote_default_values(monkeypatch: MonkeyPatch) -> None:
    monkeypatch.setattr("boto3.client", s3.s3_client)
    manager = storage.S3StorageManager(bucket="bucket", access_key="key", secret_key="secret")
    assert manager._base_path == tempfile.gettempdir()


def test_s3_lifecycle(manager: storage.S3StorageManager, checkpoint: StorableFixture) -> None:
    assert len(os.listdir(manager._base_path)) == 0

    checkpoints = []
    for _ in range(5):
        metadata = manager.store(checkpoint)
        assert len(os.listdir(manager._base_path)) == 0
        checkpoints.append(metadata)
        assert set(metadata.resources) == set(checkpoint.expected_files.keys())
        manager.restore(checkpoint, metadata)
        assert len(os.listdir(manager._base_path)) == 0

    for metadata in checkpoints:
        manager.restore(checkpoint, metadata)
        manager.delete(metadata)
        with pytest.raises(KeyError):
            manager.restore(checkpoint, metadata)


def test_verify_s3_upload_error(tmp_path: Path, monkeypatch: MonkeyPatch) -> None:
    tmpdir_s = str(tmp_path)
    monkeypatch.setattr("boto3.client", s3.s3_faulty_client)
    config = {
        "type": "s3",
        "bucket": "bucket",
        "access_key": "key",
        "secret_key": "secret",
        "temp_dir": tmpdir_s,
    }
    assert len(os.listdir(tmpdir_s)) == 0
    with pytest.raises(S3UploadFailedError):
        storage.validate(config)
    # Checkpoints are not cleaned up properly on failure (see #1674).
    assert len(os.listdir(tmpdir_s)) == 1
