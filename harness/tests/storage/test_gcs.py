import os
from pathlib import Path
from typing import Iterator

import google.auth.exceptions
import google.cloud.storage
import pytest

from determined.common import storage
from tests.storage import util


@pytest.fixture
def prep_gcs_test_creds(tmp_path: Path) -> Iterator[None]:
    """
    Check for the environment variable we pass as part of circleci's "storage-unit-tests" context.

    Note that the gcs credentials in the "storage-unit-tests" context are the keyid=c07eed131 key
    to the storage-unit-tests@determined-ai.iam.gserviceaccount.com service account.  The contents
    of the key are at github.com/determined-ai/secrets/gcp/service-accounts/storage-unit-tests.json.

    The service account should only have permission to view the "storage-unit-tests" bucket.
    """

    if "DET_GCS_TEST_CREDS" not in os.environ:
        yield
        return

    # Save the text in a temporary file and set GOOGLE_APPLICATION_CREDENTIALS to be the path.
    creds_path = tmp_path.joinpath("gcs-test-creds.json")
    with creds_path.open("w") as f:
        f.write(os.environ["DET_GCS_TEST_CREDS"])
    os.environ["GOOGLE_APPLICATION_CREDENTIALS"] = str(creds_path)
    try:
        yield
    finally:
        del os.environ["GOOGLE_APPLICATION_CREDENTIALS"]


@pytest.fixture
def live_gcs_manager(
    tmp_path: Path, prep_gcs_test_creds: None, require_secrets: bool
) -> storage.GCSStorageManager:
    """
    Skip when we have no gcs access, unless --require-secrets was set, in which case fail.

    Note that if you normally have GCS access to the bucket in question and you have done the usual
    login with the gcloud cli tool, no environment variables are necessary to run this test locally.
    """

    # Instantiating a google.cloud.storage.Client() takes a few seconds, so we speed up test by
    # reusing the one created for the storage manager.
    try:
        manager = storage.GCSStorageManager(bucket="storage-unit-tests", temp_dir=str(tmp_path))
        blob = manager.bucket.blob("check-access")
        assert blob.download_as_string() == b"yo, you have access"
    except google.auth.exceptions.DefaultCredentialsError:
        # No access detected.
        if require_secrets:
            raise
        pytest.skip("No GCS access")

    return manager


@pytest.mark.cloud  # type: ignore
def test_gcs_lifecycle(live_gcs_manager: storage.GCSStorageManager) -> None:
    def post_delete_cb(storage_id: str) -> None:
        """Search gcs directly to ensure that a checkpoint is actually deleted."""
        found = [blob.name for blob in live_gcs_manager.bucket.list_blobs(prefix=storage_id)]
        if found:
            file_list = "    " + "\n    ".join(found)
            raise ValueError(f"found {len(found)} files in bucket after delete:\n{file_list}")

    util.run_storage_lifecycle_test(live_gcs_manager, post_delete_cb)
