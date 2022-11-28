import contextlib
import pathlib
from typing import Any, Callable, Dict, Iterator, List, Optional
from unittest import mock

import pytest
import requests

from determined import core
from tests import parallel


def make_mock_storage_manager(basedir: pathlib.Path) -> Any:
    @contextlib.contextmanager
    def store_path(dst: str) -> Iterator[pathlib.Path]:
        path = basedir.joinpath("store-path")
        path.mkdir(exist_ok=True)
        yield pathlib.Path(path)

    @contextlib.contextmanager
    def restore_path(
        storage_id: str, selector: Optional[Callable[[str], bool]] = None
    ) -> Iterator[pathlib.Path]:
        path = basedir.joinpath("restore-path")
        path.mkdir(exist_ok=True)
        yield pathlib.Path(path)

    storage_manager = mock.MagicMock()
    storage_manager.store_path = mock.MagicMock(side_effect=store_path)
    storage_manager.restore_path = mock.MagicMock(side_effect=restore_path)
    storage_manager._list_directory = mock.MagicMock(return_value={"one": 1, "two": 2})

    return storage_manager


@pytest.mark.parametrize(
    "mode",
    [
        core.DownloadMode.LocalWorkersShareDownload,
        core.DownloadMode.NoSharedDownload,
    ],
    ids=lambda x: f"mode={x.name}",
)
@pytest.mark.parametrize("dummy", [False, True], ids=lambda x: f"dummy:{x}")
def test_checkpoint_context(dummy: bool, mode: core.DownloadMode, tmp_path: pathlib.Path) -> None:
    ckpt_dir = tmp_path.joinpath("ckpt-dir")
    ckpt_dir.mkdir(exist_ok=True)
    with parallel.Execution(2) as pex:

        @pex.run
        def do_test() -> None:
            storage_manager = make_mock_storage_manager(tmp_path)
            if not dummy:
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
            else:
                checkpoint_context = core.DummyCheckpointContext(pex.distributed, storage_manager)

            # Test upload.
            with parallel.raises_when(
                pex.distributed.rank == 1,
                RuntimeError,
                match="upload.*non-chief",
            ):
                checkpoint_context.upload(ckpt_dir, metadata={"steps_completed": 1})
            if pex.rank == 0:
                storage_manager.upload.assert_called_once()
                storage_manager.upload.reset_mock()
                storage_manager._list_directory.assert_called_once()
                storage_manager._list_directory.reset_mock()
                if not dummy:
                    session._do_request.assert_called_once()
                    session._do_request.reset_mock()
            else:
                storage_manager.upload.assert_not_called()
                storage_manager._list_directory.assert_not_called()
                if not dummy:
                    session._do_request.assert_not_called()
                    tensorboard_manager.sync.assert_not_called()

            # Test store_path.
            with parallel.raises_when(
                pex.distributed.rank == 1,
                RuntimeError,
                match=r"\.store_path.*non-chief",
            ):
                with checkpoint_context.store_path(metadata={"steps_completed": 1}) as _:
                    pass
            if pex.rank == 0:
                storage_manager.store_path.assert_called_once()
                storage_manager.store_path.reset_mock()
                storage_manager._list_directory.assert_called_once()
                storage_manager._list_directory.reset_mock()
                if not dummy:
                    session._do_request.assert_called_once()
                    session._do_request.reset_mock()
            else:
                storage_manager.store_path.assert_not_called()
                storage_manager._list_directory.assert_not_called()
                if not dummy:
                    session._do_request.assert_not_called()
                    tensorboard_manager.sync.assert_not_called()

            # Test download.
            unique_string = "arbitrary-string"
            if pex.distributed.rank == 0:
                checkpoint_context.download("ckpt-uuid", ckpt_dir, mode)
                if mode == core.DownloadMode.NoSharedDownload:
                    # Send broadcast after download.
                    _ = pex.distributed.broadcast_local(unique_string)
            else:
                if mode == core.DownloadMode.NoSharedDownload:
                    # Receive broadcast before download, to ensure the download is not synchronized.
                    recvd = pex.distributed.broadcast_local(unique_string)
                    assert recvd == unique_string, recvd
                checkpoint_context.download("ckpt-uuid", ckpt_dir, mode)
            storage_manager.download.assert_called_once()
            storage_manager.download.reset_mock()

            # Test restore_path.
            if pex.distributed.rank == 0:
                with checkpoint_context.restore_path("ckpt-uuid", mode) as _:
                    pass
                if mode == core.DownloadMode.NoSharedDownload:
                    _ = pex.distributed.broadcast_local(unique_string)
            else:
                if mode == core.DownloadMode.NoSharedDownload:
                    recvd = pex.distributed.broadcast_local(unique_string)
                    assert recvd == unique_string, recvd
                with checkpoint_context.restore_path("ckpt-uuid", mode) as _:
                    pass
            storage_manager.restore_path.assert_called_once()
            storage_manager.restore_path.reset_mock()


@pytest.mark.parametrize(
    "resources,expected_merged,expected_conflicts",
    [
        ([{"file0": 0}, {"file1": 0}], {"file0": 0, "file1": 0}, {}),
        ([{"file0": 0}, {"file0": 0}], {"file0": 0}, {"file0": [0, 1]}),
        ([{"dir1/": 0}, {"dir1/": 0}], {"dir1/": 0}, {}),
        ([{"file1/": 0}, {"file1": 0}], {"file1/": 0, "file1": 0}, {"file1": [0, 1]}),
        ([{"dir1/file1": 0}, {"file1": 0}], {"dir1/file1": 0, "file1": 0}, {}),
        (
            [{"dir1/file1": 0}, {"dir1/file1/": 0}],
            {"dir1/file1": 0, "dir1/file1/": 0},
            {"dir1/file1": [0, 1]},
        ),
    ],
)
def test_merge_files(
    resources: List[Dict[str, int]],
    expected_merged: Dict[str, int],
    expected_conflicts: Dict[str, List[int]],
) -> None:
    merged, conflicts = core._checkpoint.merge_resources(resources)
    assert conflicts == expected_conflicts
    assert merged == expected_merged


@pytest.mark.parametrize(
    "metadata,expected_merged,expected_conflicts",
    [
        ([{"key1": 0}, {"key2": 0}], {"key1": 0, "key2": 0}, {}),
        ([{"key1": 0}, {"key1": 0}], {"key1": 0}, {"key1": [0, 1]}),
        ([{"key1": 1, "key2": 0}, {"key1": 0}], {"key1": 1, "key2": 0}, {"key1": [0, 1]}),
        ([{"key1": [1]}, {"key1": [0]}], {"key1": [1, 0]}, {}),
        ([{"key1": [1]}, {"key1": 1}], {"key1": [1]}, {"key1": [0, 1]}),
        (
            [{"key1": {"subkey1": 1}}, {"key1": {"subkey2": 2}}],
            {"key1": {"subkey1": 1, "subkey2": 2}},
            {},
        ),
        ([{"key1": {"subkey1": 1}}, {"key1": 2}], {"key1": {"subkey1": 1}}, {"key1": [0, 1]}),
        ([{"key1": {"subkey1": 1}}, {"key1": [2]}], {"key1": {"subkey1": 1}}, {"key1": [0, 1]}),
        (
            [{"key1": {"subkey1": 1}}, {"key1": {"subkey1": 2}}],
            {"key1": {"subkey1": 1}},
            {"key1/subkey1": [0, 1]},
        ),
        (
            [{"key1": {"subkey1": {"subkey2": 1}}}, {"key1": {"subkey2": 2}}],
            {"key1": {"subkey1": {"subkey2": 1}, "subkey2": 2}},
            {},
        ),
        (
            [{"key1": {"subkey1": [1]}}, {"key1": {"subkey1": [2]}}],
            {"key1": {"subkey1": [1, 2]}},
            {},
        ),
    ],
)
def test_merge_metadata(
    metadata: List[Dict[str, Any]],
    expected_merged: Dict[str, Any],
    expected_conflicts: Dict[str, List],
) -> None:
    merged, conflicts = core._checkpoint.merge_metadata(metadata)
    assert conflicts == expected_conflicts
    assert merged == expected_merged
