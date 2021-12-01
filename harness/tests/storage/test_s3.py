import io
from pathlib import Path
from typing import Optional, Union

import botocore.exceptions
import pytest

from determined.common import storage
from determined.common.storage.s3 import normalize_prefix
from tests.storage import util


def get_live_manager(
    require_secrets: bool, tmp_path: Path, prefix: Optional[str]
) -> storage.S3StorageManager:
    """Return a working S3StorageManager connected to a real bucket.

    S3 access may come from the user's normal filesystem authentication of from the
    AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY environment variables.

    Note that we pass these variables as part of circleci's "storage-unit-tests" context.

    The circleci credentials belong to the "storage-unit-tests" user.  The contents of the key are
    at github.com/determined-ai/secrets/aws/access-keys/storage-unit-tests.csv.

    The user only has premissions to read/write the "storage-unit-tests" bucket.
    """

    try:
        # AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY are read from the environment automatically.
        manager = storage.S3StorageManager(
            bucket="storage-unit-tests",
            access_key=None,
            secret_key=None,
            prefix=prefix,
            temp_dir=str(tmp_path),
        )
        out = io.BytesIO()
        manager.bucket.download_fileobj("check-access", out)
        assert out.getvalue() == b"yo, you have access"
    except (botocore.exceptions.NoCredentialsError, botocore.exceptions.PartialCredentialsError):
        # No access detected.
        if require_secrets:
            raise
        pytest.skip("No S3 access")

    return manager


@pytest.mark.cloud
@pytest.mark.parametrize(
    "prefix,expected_storage_prefix,should_fail",
    [
        (None, "", False),
        ("/", "", False),
        ("///", "", False),
        ("foo", "foo", False),
        ("/foo", "foo", False),
        ("/foo/", "foo", False),
        ("./foo/", "foo", False),
        ("/./foo/", "foo", False),
        ("/fo..o/", "fo..o", False),
        ("/foo/..", "-", True),
        ("../foo", "-", True),
        ("fo/../o", "-", True),
        ("..", "-", True),
    ],
)
def test_storage_prefix_normalization(
    require_secrets: bool,
    tmp_path: Path,
    prefix: Union[None, str],
    expected_storage_prefix: str,
    should_fail: bool,
) -> None:
    """Test various inputs for storage prefix are normalized properly."""

    try:
        observed_prefix = normalize_prefix(prefix)
        assert not should_fail and observed_prefix == expected_storage_prefix
    except ValueError as exc:
        assert should_fail and "prefix must not match" in str(exc)


@pytest.mark.cloud
@pytest.mark.parametrize("prefix", [None, "my/test/prefix"])
def test_live_s3_lifecycle(require_secrets: bool, tmp_path: Path, prefix: Optional[str]) -> None:

    live_manager = get_live_manager(require_secrets, tmp_path, prefix)

    def post_delete_cb(storage_id: str) -> None:
        """Search s3 directly to ensure that a checkpoint is actually deleted."""
        storage_prefix = live_manager.get_storage_prefix(storage_id)
        found = [obj.key for obj in live_manager.bucket.objects.filter(Prefix=storage_prefix)]
        if found:
            file_list = "    " + "\n    ".join(found)
            raise ValueError(f"found {len(found)} files in bucket after delete:\n{file_list}")

    util.run_storage_lifecycle_test(live_manager, post_delete_cb)
