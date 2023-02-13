import os
import uuid
from pathlib import Path
from typing import Dict, Iterator, List, Optional

import google.auth.exceptions
import google.cloud.storage
import pytest

from determined import errors
from determined.common import storage
from determined.tensorboard.fetchers.gcs import GCSFetcher
from tests.storage import util

BUCKET_NAME = "storage-unit-tests"
CHECK_ACCESS_KEY = "check-access"
CHECK_KEY_CONTENT = b"yo, you have access"


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


def get_live_gcs_manager(
    tmp_path: Path,
    prefix: Optional[str],
    require_secrets: bool,
) -> storage.GCSStorageManager:
    """
    Skip when we have no gcs access, unless --require-secrets was set, in which case fail.

    Note that if you normally have GCS access to the bucket in question and you have done the usual
    login with the gcloud cli tool, no environment variables are necessary to run this test locally.
    """

    # Instantiating a google.cloud.storage.Client() takes a few seconds, so we speed up test by
    # reusing the one created for the storage manager.
    try:
        manager = storage.GCSStorageManager(
            bucket=BUCKET_NAME,
            prefix=prefix,
            temp_dir=str(tmp_path),
        )
        blob = manager.bucket.blob(CHECK_ACCESS_KEY)
        assert blob.download_as_string() == CHECK_KEY_CONTENT
    except errors.NoDirectStorageAccess as e:
        # No access detected.
        if (not require_secrets) and isinstance(
            e.__cause__, google.auth.exceptions.DefaultCredentialsError
        ):
            pytest.skip("No GCS access")
        raise

    return manager


@pytest.mark.cloud
@pytest.mark.parametrize("prefix", [None, "test/prefix/"])
def test_gcs_lifecycle(
    require_secrets: bool,
    tmp_path: Path,
    prefix: Optional[str],
    prep_gcs_test_creds: None,
) -> None:
    live_gcs_manager = get_live_gcs_manager(tmp_path, prefix, require_secrets)

    def post_delete_cb(storage_id: str) -> None:
        """Search gcs directly to ensure that a checkpoint is actually deleted."""
        storage_prefix = live_gcs_manager.get_storage_prefix(storage_id)
        found = [blob.name for blob in live_gcs_manager.bucket.list_blobs(prefix=storage_prefix)]
        if found:
            file_list = "    " + "\n    ".join(found)
            raise ValueError(f"found {len(found)} files in bucket after delete:\n{file_list}")

    util.run_storage_lifecycle_test(live_gcs_manager, post_delete_cb)


def get_tensorboard_fetcher_gcs(
    require_secrets: bool, local_sync_dir: str, paths_to_sync: List[str]
) -> GCSFetcher:

    storage_config = {"bucket": BUCKET_NAME}

    try:
        fetcher = GCSFetcher(storage_config, paths_to_sync, local_sync_dir)

        blob = fetcher.client.bucket(BUCKET_NAME).blob("check-access")
        assert blob.download_as_string() == CHECK_KEY_CONTENT

        return fetcher

    except google.auth.exceptions.DefaultCredentialsError:
        # No access detected.
        if require_secrets:
            raise
        pytest.skip("No GCS access")


@pytest.mark.cloud
def test_tensorboard_fetcher_gcs(
    require_secrets: bool, tmp_path: Path, prep_gcs_test_creds: None
) -> None:

    local_sync_dir = os.path.join(tmp_path, "sync_dir")
    storage_relpath = os.path.join(local_sync_dir, BUCKET_NAME)

    # Create two paths as multi-trial sync could happen.
    paths_to_sync = [os.path.join("test_dir", str(uuid.uuid4()), "subdir") for _ in range(2)]

    fetcher = get_tensorboard_fetcher_gcs(require_secrets, local_sync_dir, paths_to_sync)

    def put_files(filepath_content: Dict[str, bytes]) -> None:
        for filepath, content in filepath_content.items():
            fetcher.client.bucket(BUCKET_NAME).blob(filepath).upload_from_string(content)

    def rm_files(filepaths: List[str]) -> None:
        for filepath in filepaths:
            fetcher.client.bucket(BUCKET_NAME).blob(filepath).delete()

    util.run_tensorboard_fetcher_test(local_sync_dir, fetcher, storage_relpath, put_files, rm_files)
