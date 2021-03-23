import os
import tempfile
from pathlib import Path

import pytest
from _pytest.monkeypatch import MonkeyPatch
from boto3.exceptions import S3UploadFailedError

from determined.common import storage
from tests import s3
from tests.storage import util


@pytest.fixture
def manager(tmp_path: Path, monkeypatch: MonkeyPatch) -> storage.S3StorageManager:
    monkeypatch.setattr("boto3.client", s3.s3_client)
    return storage.S3StorageManager(
        bucket="bucket", access_key="key", secret_key="secret", temp_dir=str(tmp_path)
    )


def test_remote_default_values(monkeypatch: MonkeyPatch) -> None:
    monkeypatch.setattr("boto3.client", s3.s3_client)
    manager = storage.S3StorageManager(bucket="bucket", access_key="key", secret_key="secret")
    assert manager._base_path == tempfile.gettempdir()


def test_s3_lifecycle(manager: storage.S3StorageManager) -> None:
    assert len(os.listdir(manager._base_path)) == 0

    checkpoints = []
    for _ in range(5):
        with manager.store_path() as (storage_id, path):
            # Ensure no checkpoint directories exist yet.
            assert len(os.listdir(manager._base_path)) == 0
            util.create_checkpoint(path)
            metadata = storage.StorageMetadata(storage_id, manager._list_directory(path))
            checkpoints.append(metadata)
            assert set(metadata.resources) == set(util.EXPECTED_FILES.keys())

    for metadata in checkpoints:
        # Load every checkpoint:
        with manager.restore_path(metadata) as path:
            util.validate_checkpoint(path)
        manager.delete(metadata)
        with pytest.raises(KeyError):
            with manager.restore_path(metadata) as path:
                pass


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
        storage.validate_config(config, container_path=None)
    assert len(os.listdir(tmpdir_s)) == 0
