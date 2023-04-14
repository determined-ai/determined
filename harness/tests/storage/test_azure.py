import io
import os
import tempfile
import uuid
from pathlib import Path
from typing import Dict, List

import pytest

from determined.common import storage
from determined.tensorboard.fetchers.azure import AzureFetcher
from tests.storage import util

CONTAINER_NAME = "storage-unit-tests"
CHECK_ACCESS_KEY = "check-access"
CHECK_KEY_CONTENT = b"yo, you have access"


def get_live_azure_manager(require_secrets: bool, tmp_path: Path) -> storage.AzureStorageManager:
    """Return a working AzureStorageManager connected to a real bucket.

    Check for the environment variable we pass as part of circleci's "storage-unit-tests" context.

    Note that the Azure credentials in the "storage-unit-tests" context are available at the
    following location:

    github.com/determined-ai/secrets/blob/master/azure/connection-strings/storage-unit-tests.txt

    The service account should only have permission to view the "storage-unit-tests" bucket.

    Note: this connection_string can be set via the DET_AZURE_TEST_CREDS environment variable.
    """
    connection_string = os.environ.get("DET_AZURE_TEST_CREDS")

    import azure.core.exceptions

    try:
        manager = storage.AzureStorageManager(CONTAINER_NAME, connection_string)
        with tempfile.TemporaryDirectory() as tmp_dirname:
            tmp_filepath = os.path.join(tmp_dirname, "access.file")
            manager.client.get(CONTAINER_NAME, CHECK_ACCESS_KEY, tmp_filepath)

            with open(tmp_filepath, "rb") as f:
                data = f.read()
                assert data == CHECK_KEY_CONTENT

        return manager

    except (
        ValueError,
        azure.core.exceptions.ClientAuthenticationError,
        azure.core.exceptions.ResourceNotFoundError,
    ):
        if require_secrets:
            raise
        pytest.skip("No Azure access")


@pytest.mark.cloud
def test_live_azure_lifecycle(require_secrets: bool, tmp_path: Path) -> None:
    live_manager = get_live_azure_manager(require_secrets, tmp_path)

    def post_delete_cb(storage_id: str) -> None:
        """Search Azure directly to ensure that a checkpoint is actually deleted."""
        found = [
            blob["name"] for blob in live_manager.client.list_files(CONTAINER_NAME, storage_id)
        ]
        if found:
            file_list = "    " + "\n    ".join(found)
            raise ValueError(f"found {len(found)} files in container after delete:\n{file_list}")

    util.run_storage_lifecycle_test(live_manager, post_delete_cb)


def get_tensorboard_fetcher_azure(
    require_secrets: bool, local_sync_dir: str, paths_to_sync: List[str]
) -> AzureFetcher:
    connection_string = os.environ.get("DET_AZURE_TEST_CREDS")
    storage_config = {"connection_string": connection_string, "container": CONTAINER_NAME}

    import azure.core.exceptions

    try:
        fetcher = AzureFetcher(storage_config, paths_to_sync, local_sync_dir)
        data = io.BytesIO()

        blob_client = fetcher.client.get_blob_client(CONTAINER_NAME, CHECK_ACCESS_KEY)
        blob_client.download_blob().readinto(data)

        data.seek(0)
        assert data.read() == CHECK_KEY_CONTENT

        return fetcher

    except (
        ValueError,
        azure.core.exceptions.ClientAuthenticationError,
        azure.core.exceptions.ResourceNotFoundError,
    ):
        if require_secrets:
            raise
        pytest.skip("No Azure access")


@pytest.mark.cloud
def test_tensorboard_fetcher_azure(require_secrets: bool, tmp_path: Path) -> None:
    local_sync_dir = os.path.join(tmp_path, "sync_dir")
    storage_relpath = os.path.join(local_sync_dir, CONTAINER_NAME)

    # Create two paths as multi-trial sync could happen.
    paths_to_sync = [os.path.join("test_dir", str(uuid.uuid4()), "subdir") for _ in range(2)]

    fetcher = get_tensorboard_fetcher_azure(require_secrets, local_sync_dir, paths_to_sync)

    def put_files(filepath_content: Dict[str, bytes]) -> None:
        for filepath, content in filepath_content.items():
            blob_client = fetcher.client.get_blob_client(CONTAINER_NAME, filepath)
            blob_client.upload_blob(content, overwrite=True)

    def rm_files(filepaths: List[str]) -> None:
        for filepath in filepaths:
            blob_client = fetcher.client.get_blob_client(CONTAINER_NAME, filepath)
            blob_client.delete_blob()

    util.run_tensorboard_fetcher_test(local_sync_dir, fetcher, storage_relpath, put_files, rm_files)
