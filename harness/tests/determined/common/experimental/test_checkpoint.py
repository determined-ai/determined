import os
import re
from typing import Dict, Optional, Union, cast
from unittest import mock

import pytest
import responses

from determined.common import api, storage
from determined.common.experimental import checkpoint
from tests.fixtures import api_responses

_MASTER = "http://localhost:8080"

StorageConfig = Dict[str, Optional[Union[str, int]]]


@pytest.fixture
def standard_session() -> api.Session:
    return api.Session(master=_MASTER, user=None, auth=None, cert=None)


@pytest.fixture
def sample_checkpoint(standard_session: api.Session) -> checkpoint.Checkpoint:
    bindings_checkpoint = api_responses.sample_get_checkpoint().checkpoint
    return checkpoint.Checkpoint._from_bindings(bindings_checkpoint, standard_session)


def storage_conf_of_checkpoint(
    sample_checkpoint: checkpoint.Checkpoint,
) -> StorageConfig:
    if sample_checkpoint.training is None or sample_checkpoint.training.experiment_config is None:
        raise ValueError(
            "Test depends on an existing experiment_config within the tested checkpoint."
        )
    storage_conf = sample_checkpoint.training.experiment_config["checkpoint_storage"]
    return cast(StorageConfig, storage_conf)


@mock.patch.object(storage.GCSStorageManager, "download")
def test_download_calls_GCSStorageManager_download_in_direct_download_mode(
    mock_download: mock.MagicMock,
    sample_checkpoint: checkpoint.Checkpoint,
    tmp_path: os.PathLike,
) -> None:
    # Patch to avoid making actual calls to GCS.
    with mock.patch("google.cloud.storage.Client"):
        storage_conf = storage_conf_of_checkpoint(sample_checkpoint)
        storage_conf.update({"type": "gcs", "prefix": None})
        del storage_conf["access_key"]
        del storage_conf["secret_key"]
        del storage_conf["endpoint_url"]

        sample_checkpoint.download(path=str(tmp_path), mode=checkpoint.DownloadMode.DIRECT)
        assert mock_download.call_count == 1


@mock.patch.object(storage.S3StorageManager, "download")
def test_download_calls_S3StorageManager_download_in_direct_download_mode(
    mock_download: mock.MagicMock,
    sample_checkpoint: checkpoint.Checkpoint,
    tmp_path: os.PathLike,
) -> None:
    # Patch boto methods to avoid making actual calls to S3.
    with mock.patch(
        "determined.common.storage.boto3_credential_manager.initialize_boto3_credential_providers"
    ), mock.patch("boto3.resource"):
        storage_conf = storage_conf_of_checkpoint(sample_checkpoint)
        storage_conf.update(
            {"type": "s3", "secret_key": None, "endpoint_url": None, "prefix": None}
        )

        sample_checkpoint.download(path=str(tmp_path), mode=checkpoint.DownloadMode.DIRECT)
        assert mock_download.call_count == 1


@responses.activate
def test_add_metadata_doesnt_update_local_on_rest_failure(
    sample_checkpoint: checkpoint.Checkpoint,
) -> None:
    sample_checkpoint.metadata = {}

    responses.post(
        re.compile(f"{_MASTER}/api/v1/checkpoints/{sample_checkpoint.uuid}.*"), status=400
    )

    try:
        sample_checkpoint.add_metadata({"test": "test"})
        raise AssertionError("Server's 400 should raise an exception")
    except api.errors.APIException:
        assert "test" not in sample_checkpoint.metadata


@responses.activate
def test_remove_metadata_doesnt_update_local_on_rest_failure(
    sample_checkpoint: checkpoint.Checkpoint,
) -> None:
    sample_checkpoint.metadata = {"test": "test"}

    responses.post(
        re.compile(f"{_MASTER}/api/v1/checkpoints/{sample_checkpoint.uuid}.*"), status=400
    )

    try:
        sample_checkpoint.remove_metadata(["test"])
        raise AssertionError("Server's 400 should raise an exception")
    except api.errors.APIException:
        assert "test" in sample_checkpoint.metadata
