import io
from pathlib import Path

import botocore.exceptions
import pytest

from determined.common import storage
from tests.storage import util


@pytest.fixture
def live_manager(tmp_path: Path, require_secrets: bool) -> storage.S3StorageManager:
    """
    Return a working S3StorageManager connected to a real bucket when we detect that we have access
    to s3.  S3 access may come from the user's normal filesystem authentication of from the
    AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY environment variables.

    Note that we pass these variables as part of circleci's "storage-unit-tests" context.

    The circleci credentials belong to the "storage-unit-tests" user.  The contest of the key are at
    github.com/determined-ai/secrets/aws/access-keys/storage-unit-tests.csv.

    The user only has premissions to read/write the "storage-unit-tests" bucket.
    """

    try:
        # AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY are read from the environment automatically.
        manager = storage.S3StorageManager(
            bucket="storage-unit-tests", access_key=None, secret_key=None, temp_dir=str(tmp_path)
        )
        out = io.BytesIO()
        manager.bucket.download_fileobj("check-access", out)
        assert out.getvalue() == b"yo, you have access"
    except botocore.exceptions.NoCredentialsError:
        # No access detected.
        if require_secrets:
            raise
        pytest.skip("No S3 access")

    return manager


@pytest.mark.cloud  # type: ignore
def test_live_s3_lifecycle(live_manager: storage.S3StorageManager, require_secrets: bool) -> None:
    def post_delete_cb(storage_id: str) -> None:
        """Search s3 directly to ensure that a checkpoint is actually deleted."""
        found = [obj.key for obj in live_manager.bucket.objects.filter(Prefix=storage_id)]
        if found:
            file_list = "    " + "\n    ".join(found)
            raise ValueError(f"found {len(found)} files in bucket after delete:\n{file_list}")

    util.run_storage_lifecycle_test(live_manager, post_delete_cb)
