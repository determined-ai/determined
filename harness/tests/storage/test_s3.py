import io
import os
import uuid
from pathlib import Path
from typing import Dict, List, Optional, Union, Set

import botocore.exceptions
import pytest

from determined.common import storage
from determined.common.storage.s3 import normalize_prefix
from determined.tensorboard.fetchers.s3 import S3Fetcher
from tests.storage import util

from determined import core
from tests import parallel
from unittest import mock
import requests
import logging

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


FILES_NODE0 = {'file1_0', 'file2_0', 'dir1_0/file3_0', 'metadata.json'}
FILES_NODE1 = {'file1_1', 'file2_1', 'dir1_1/file3_1', 'metadata.json'}

@pytest.mark.cloud
def test_live_s3_sharded_upload_download(require_secrets: bool, tmp_path: Path) -> None:

    def clean_up(storage_id: str, storage_manager) -> None:
        """Search s3 directly to ensure that a checkpoint is actually deleted."""
        storage_manager.delete(storage_id)
        storage_prefix = storage_manager.get_storage_prefix(storage_id)
        found = [obj.key for obj in storage_manager.bucket.objects.filter(Prefix=storage_prefix)]
        if found:
            file_list = "    " + "\n    ".join(found)
            raise ValueError(f"found {len(found)} files in bucket after delete:\n{file_list}")

    with parallel.Execution(2) as pex:

        @pex.run
        def do_test() -> None:
            # init storage manager
            tmp_path_storage = tmp_path.joinpath('storage')
            storage_manager = get_live_manager(require_secrets, tmp_path_storage, None)

            # create checkpoint context
            checkpoint_context = create_checkpoint_context(pex, storage_manager)

            # create "local" file structure
            ckpt_dir = os.path.join(tmp_path, f'ckpt_dir_{pex.distributed.rank}')
            create_checkpoint_dir(ckpt_dir, suffix=str(pex.distributed.rank))

            # wait for all ranks to save files
            pex.distributed.allgather(None)

            logging.info(f'Rank {pex.distributed.rank}. '
                         f'Files in ckpt_dir: {[os.path.join(dp, f) for dp, dn, fn in os.walk(ckpt_dir) for f in fn]}')

            if pex.distributed.rank == 0:
                metadata = {"steps_completed": 1}
            else:
                metadata = None

            ##################################
            ### Upload sharded data
            storage_id = checkpoint_context.upload(ckpt_dir, metadata, shard=True)

            ##################################
            ### Test downloading with and w/o selectors
            ##################################
            # 1. No selector: every rank gets all the files + metadata
            download_dir1 = os.path.join(tmp_path, f'test1_download_{pex.distributed.rank}')
            checkpoint_context.download(storage_id, download_dir1)
            validate_checkpoint(download_dir1, expected_files=FILES_NODE0.union(FILES_NODE1))

            ##################################
            # 2. Every rank uses selector: only selected files are downloaded (what about metadata?)
            download_dir2 = os.path.join(tmp_path, f'test2_download_{pex.distributed.rank}')
            if pex.distributed.rank == 0:
                selector = lambda x: x == 'dir1_0/file3_0'
            else:
                selector = lambda x: x == 'file1_1'

            checkpoint_context.download(storage_id, download_dir2, selector=selector)

            if pex.distributed.rank == 0:
                validate_checkpoint(download_dir2, expected_files={'dir1_0/file3_0'})
            else:
                validate_checkpoint(download_dir2, expected_files={'file1_1'})

            ##################################
            # 3. Only rank=0 gets its files
            download_dir3 = os.path.join(tmp_path, f'test3_download_{pex.distributed.rank}')
            if pex.distributed.rank == 0:
                selector = lambda x: x in FILES_NODE0
            else:
                selector = None

            checkpoint_context.download(storage_id, download_dir3, selector=selector)

            if pex.distributed.rank == 0:
                validate_checkpoint(download_dir3, expected_files=FILES_NODE0)
            else:
                validate_checkpoint(download_dir3, expected_files=set())

            if pex.distributed.rank == 0:
                clean_up(storage_id, storage_manager)
            pex.distributed.allgather(None)

            ##################################
            ### Upload sharded data from rank 0 only
            storage_id = checkpoint_context.upload(ckpt_dir if pex.distributed.rank == 0 else None, metadata, shard=True)
            download_dir4 = os.path.join(tmp_path, f'test4_download_{pex.distributed.rank}')
            checkpoint_context.download(storage_id, download_dir4)
            validate_checkpoint(download_dir4, expected_files=FILES_NODE0)

            if pex.distributed.rank == 0:
                clean_up(storage_id, storage_manager)
            pex.distributed.broadcast(None)

            ##################################
            ### Upload sharded data from rank 1 only
            storage_id = checkpoint_context.upload(ckpt_dir if pex.distributed.rank == 1 else None, metadata,
                                                   shard=True)
            download_dir5 = os.path.join(tmp_path, f'test5_download_{pex.distributed.rank}')
            checkpoint_context.download(storage_id, download_dir5)
            validate_checkpoint(download_dir5, expected_files=FILES_NODE1)

            if pex.distributed.rank == 0:
                clean_up(storage_id, storage_manager)
            pex.distributed.broadcast(None)


@pytest.mark.cloud
def test_live_s3_sharded_store_restore(require_secrets: bool, tmp_path: Path) -> None:

    def clean_up(storage_id: str, storage_manager) -> None:
        """Search s3 directly to ensure that a checkpoint is actually deleted."""
        storage_manager.delete(storage_id)
        storage_prefix = storage_manager.get_storage_prefix(storage_id)
        found = [obj.key for obj in storage_manager.bucket.objects.filter(Prefix=storage_prefix)]
        if found:
            file_list = "    " + "\n    ".join(found)
            raise ValueError(f"found {len(found)} files in bucket after delete:\n{file_list}")

    with parallel.Execution(2) as pex:

        @pex.run
        def do_test() -> None:
            # init storage manager
            tmp_path_storage = tmp_path.joinpath(f'storage_{pex.distributed.rank}')
            storage_manager = get_live_manager(require_secrets, tmp_path_storage, None)

            # create checkpoint context
            checkpoint_context = create_checkpoint_context(pex, storage_manager)

            ### Upload sharded data
            if pex.distributed.rank == 0:
                metadata = {"steps_completed": 1}
            else:
                metadata = None

            with checkpoint_context.store_path(metadata, shard=True) as (path, storage_id):
                logging.info(f'storage_id={storage_id}')
                create_checkpoint_dir(path, suffix=str(pex.distributed.rank))

            pex.distributed.broadcast(None)
            ##################################
            ### Test downloading with and w/o selectors
            # 1. No selector: every rank gets all the files + metadata
            with checkpoint_context.restore_path(storage_id) as path:
                logging.info(f'Restore. rank {pex.distributed.rank}: all files under {path}: {os.listdir(path)}')
                validate_checkpoint(path, expected_files=FILES_NODE0.union(FILES_NODE1))

            pex.distributed.broadcast(None)
            ##################################
            # 2. Every rank uses selector: only selected files are downloaded
            if pex.distributed.rank == 0:
                selector = lambda x: x == 'dir1_0/file3_0'
            else:
                selector = lambda x: x == 'file1_1'

            with checkpoint_context.restore_path(storage_id, selector=selector) as path:
                if pex.distributed.rank == 0:
                    validate_checkpoint(path, expected_files={'dir1_0/file3_0'})
                else:
                    validate_checkpoint(path, expected_files={'file1_1'})

            pex.distributed.broadcast(None)
            # ##################################
            # # 3. Only rank=0 gets its files
            if pex.distributed.rank == 0:
                selector = lambda x: x in FILES_NODE0
            else:
                selector = None

            with checkpoint_context.restore_path(storage_id, selector=selector) as path:
                if pex.distributed.rank == 0:
                    validate_checkpoint(path, expected_files=FILES_NODE0)
                else:
                    validate_checkpoint(path, expected_files=set())

            if pex.distributed.rank == 0:
                clean_up(storage_id, storage_manager)
            pex.distributed.broadcast(None)


def validate_checkpoint(ckpt_dir: str, expected_files: Set[str]) -> None:
    files = [os.path.join(dp, f) for dp, dn, fn in os.walk(ckpt_dir) for f in fn]

    # convert absolute path to path relative to the "local" directory
    files = set([os.path.relpath(f, ckpt_dir) for f in files])
    logging.info(f'{ckpt_dir}:{files}')
    assert len(files) == len(expected_files)
    assert files == expected_files


def create_checkpoint_dir(top_dir: str, suffix: str = '') -> None:
    other_dir = os.path.join(top_dir, f"dir1_{suffix}")
    os.makedirs(other_dir)

    open(os.path.join(top_dir, f'file1_{suffix}'), 'w').close()
    open(os.path.join(top_dir, f'file2_{suffix}'), 'w').close()

    open(os.path.join(other_dir, f'file3_{suffix}'), 'w').close()
    logging.info(f'all files under {top_dir}: {os.listdir(top_dir)}')


def create_checkpoint_context(pex, storage_manager):
    session = mock.MagicMock()
    response = requests.Response()
    response.status_code = 200
    session._do_request.return_value = response
    tensorboard_manager = mock.MagicMock()
    checkpoint_context = core.CheckpointContext(
        pex.distributed,
        storage_manager,
        session=session,
        task_id="task-id",
        allocation_id="allocation-id",
        tbd_sync_mode=core.TensorboardMode.AUTO,
        tensorboard_manager=tensorboard_manager,
    )
    return checkpoint_context