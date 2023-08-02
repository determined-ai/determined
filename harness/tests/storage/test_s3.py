import io
import logging
import os
import uuid
from pathlib import Path
from typing import Dict, List, Optional, Union

import botocore.exceptions
import pytest

from determined.common import storage
from determined.common.storage.s3 import normalize_prefix
from determined.tensorboard.fetchers.s3 import S3Fetcher
from tests import parallel
from tests.storage import util

BUCKET_NAME = "storage-unit-tests"
CHECK_ACCESS_KEY = "check-access"
CHECK_KEY_CONTENT = b"yo, you have access"


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
            bucket=BUCKET_NAME,
            access_key=None,
            secret_key=None,
            prefix=prefix,
            temp_dir=str(tmp_path),
        )
        out = io.BytesIO()
        manager.bucket.download_fileobj(CHECK_ACCESS_KEY, out)
        assert out.getvalue() == CHECK_KEY_CONTENT
        return manager
    except (botocore.exceptions.NoCredentialsError, botocore.exceptions.PartialCredentialsError):
        # No access detected.
        if require_secrets:
            raise
        pytest.skip("No S3 access")


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


def get_tensorboard_fetcher_s3(
    require_secrets: bool, local_sync_dir: str, paths_to_sync: List[str]
) -> S3Fetcher:
    storage_config = {"bucket": BUCKET_NAME}

    try:
        fetcher = S3Fetcher(storage_config, paths_to_sync, local_sync_dir)

        out = io.BytesIO()
        fetcher.client.download_fileobj(BUCKET_NAME, CHECK_ACCESS_KEY, out)
        assert out.getvalue() == CHECK_KEY_CONTENT

        return fetcher

    except (botocore.exceptions.NoCredentialsError, botocore.exceptions.PartialCredentialsError):
        # No access detected.
        if require_secrets:
            raise
        pytest.skip("No S3 access")


@pytest.mark.cloud
def test_tensorboard_fetcher_s3(require_secrets: bool, tmp_path: Path) -> None:
    local_sync_dir = os.path.join(tmp_path, "sync_dir")
    storage_relpath = os.path.join(local_sync_dir, BUCKET_NAME)

    # Create two paths as multi-trial sync could happen.
    paths_to_sync = [os.path.join("test_dir", str(uuid.uuid4()), "subdir") for _ in range(2)]

    fetcher = get_tensorboard_fetcher_s3(require_secrets, local_sync_dir, paths_to_sync)

    def put_files(files: Dict[str, bytes]) -> None:
        for path, filebytes in files.items():
            fetcher.client.put_object(Bucket=BUCKET_NAME, Key=path, Body=filebytes)

    def rm_files(files: List[str]) -> None:
        for path in files:
            fetcher.client.delete_object(Bucket=BUCKET_NAME, Key=path)

    util.run_tensorboard_fetcher_test(local_sync_dir, fetcher, storage_relpath, put_files, rm_files)


def clean_up(storage_id: str, storage_manager: storage.S3StorageManager) -> None:
    """Search s3 directly to ensure that a checkpoint is actually deleted."""
    storage_manager.delete(storage_id, ["**/*"])
    storage_prefix = storage_manager.get_storage_prefix(storage_id)
    found = [obj.key for obj in storage_manager.bucket.objects.filter(Prefix=storage_prefix)]
    if found:
        file_list = "    " + "\n    ".join(found)
        logging.info(f"found {len(found)} files in bucket after delete:\n{file_list}")
        raise ValueError(f"found {len(found)} files in bucket after delete:\n{file_list}")


@pytest.mark.cloud
def test_live_s3_sharded_upload_download(
    require_secrets: bool,
    tmp_path: Path,
) -> None:
    with parallel.Execution(4, local_size=2) as pex:

        @pex.run
        def do_test() -> None:
            tmp_path_storage = tmp_path.joinpath("storage")
            storage_manager = get_live_manager(require_secrets, tmp_path_storage, None)
            util.run_storage_upload_download_sharded_test(pex, storage_manager, tmp_path, clean_up)


@pytest.mark.cloud
def test_live_s3_sharded_store_restore(require_secrets: bool, tmp_path: Path) -> None:
    with parallel.Execution(4, local_size=2) as pex:

        @pex.run
        def do_test() -> None:
            tmp_path_storage = tmp_path.joinpath(f"storage_{pex.distributed.rank}")
            storage_manager = get_live_manager(require_secrets, tmp_path_storage, None)
            util.run_storage_store_restore_sharded_test(pex, storage_manager, clean_up)
