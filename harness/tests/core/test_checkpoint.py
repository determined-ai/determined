import contextlib
import pathlib
from typing import Any, Callable, Dict, Iterator, List, Optional
from unittest import mock

import pytest
import requests

from determined import core
from tests import parallel


def make_mock_storage_manager(
    basedir: pathlib.Path,
    dir_files: Optional[List[str]] = None,
) -> Any:
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

    if dir_files:
        for file in dir_files:
            (basedir / file).touch(exist_ok=True)
    else:
        dir_files = ["one", "two"]

    def pre_store_path(dst: str) -> pathlib.Path:
        path = basedir.joinpath("store-path")
        path.mkdir(exist_ok=True)
        return pathlib.Path(path)

    mock_list_dir = {f: i for i, f in enumerate(dir_files)}

    storage_manager = mock.MagicMock()
    storage_manager.store_path = mock.MagicMock(side_effect=store_path)
    storage_manager.pre_store_path = mock.MagicMock(side_effect=pre_store_path)
    storage_manager.restore_path = mock.MagicMock(side_effect=restore_path)
    storage_manager._list_directory = mock.MagicMock(return_value=mock_list_dir)
    storage_manager.delete = mock.MagicMock()

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
                    storage_backend_id=None,
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

            # Test delete.
            if pex.distributed.rank == 0:
                checkpoint_context.delete("ckpt-uuid")
                if not dummy:
                    session._do_request.assert_called_once()
                    session._do_request.reset_mock()
                storage_manager.delete.assert_called_once()
                storage_manager.delete.reset_mock()


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
        ([{"a": 0}, {"b": 0}], {"a": 0, "b": 0}, {}),
        ([{"a": 0}, {"a": 0}], {"a": 0}, {}),
        ([{"a": 1, "b": 0}, {"a": 0}], {"a": 1, "b": 0}, {"/a": [0, 1]}),
        (
            [{"a": {"c": 1}}, {"a": {"d": 2}}],
            {"a": {"c": 1, "d": 2}},
            {},
        ),
        ([{"a": {"c": 1}}, {"a": 2}], {"a": {"c": 1}}, {"/a": [0, 1]}),
        ([{"a": {"c": 1}}, {"a": [2]}], {"a": {"c": 1}}, {"/a": [0, 1]}),
        (
            [{"a": {"c": 1}}, {"a": {"c": 2}}],
            {"a": {"c": 1}},
            {"/a/c": [0, 1]},
        ),
        (
            [{"a": {"c": {"d": 1}}}, {"a": {"d": 2}}],
            {"a": {"c": {"d": 1}, "d": 2}},
            {},
        ),
        (
            [
                {"a": 1, "d": {"a": 1, "d": {"a": 1}}},
                {"b": 2, "d": {"b": 2, "d": {"a": 1}}},
                {"c": 3, "d": {"c": 3, "d": {"a": 1}}},
            ],
            {"c": 3, "d": {"c": 3, "d": {"a": 1}, "b": 2, "a": 1}, "b": 2, "a": 1},
            {},
        ),
        (
            [
                {"a": 1, "d": {"a": 1, "d": {"a": 1}}},
                {"b": 2, "d": {"b": 2, "d": {"b": 1, "c": 1}}},
                {"c": 3, "d": {"b": 3, "d": {"c": 2, "a": 2}}},
            ],
            {"a": 1, "d": {"a": 1, "d": {"a": 1, "b": 1, "c": 1}, "b": 2}, "b": 2, "c": 3},
            {"/d/b": [1, 2], "/d/d/c": [1, 2], "/d/d/a": [0, 2]},
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


@pytest.mark.parametrize("sharded", [True, False])
def test_checkpoint_upload(sharded: bool, tmp_path: pathlib.Path) -> None:
    ckpt_dir = tmp_path.joinpath("ckpt-dir")
    ckpt_dir.mkdir(exist_ok=True)

    # Create some mock files for each worker to upload, and also identical files across all
    # workers to test the sharded conflict case.
    ckpt_files = {
        0: ["worker0-0", "worker0-1"],
        1: ["worker1-0", "worker1-1"],
    }

    all_workers_files = ["metadata.json", "file2", "file3"]

    with parallel.Execution(2) as pex:

        @pex.run
        def upload_ckpt() -> List[str]:
            storage_manager = make_mock_storage_manager(
                basedir=ckpt_dir, dir_files=ckpt_files[pex.rank] + all_workers_files
            )
            checkpoint_context = core.DummyCheckpointContext(pex.distributed, storage_manager)

            # Upload across all workers, expect exceptions on non-chief workers if shard=False.
            with parallel.raises_when(
                not sharded and pex.distributed.rank != 0,
                RuntimeError,
                match="upload.*non-chief",
            ):
                checkpoint_context.upload(
                    ckpt_dir,
                    metadata={"steps_completed": 1},
                    shard=sharded,
                    # Implement a selector to test file conflicts in sharded case.
                    selector=lambda _: True,
                )

            # When shard=True, every worker will call upload. When shard=False, only the chief
            # worker will upload.
            upload_paths = []
            if sharded or pex.rank == 0:
                storage_manager.upload.assert_called_once()
                upload_paths = storage_manager.upload.call_args.kwargs["paths"]
                storage_manager.upload.reset_mock()
                storage_manager._list_directory.assert_called_once()
                storage_manager._list_directory.reset_mock()
            else:
                storage_manager.upload.assert_not_called()
                storage_manager._list_directory.assert_not_called()
            return upload_paths

    assert len(upload_ckpt) == 2
    assert sorted(upload_ckpt[0]) == sorted(ckpt_files[0] + all_workers_files)

    if sharded:
        # In the sharded case, expect each worker to upload unique files. Files that conflict across
        # workers should only be uploaded by the chief worker.
        assert sorted(upload_ckpt[1]) == sorted(ckpt_files[1])
        assert len(set(upload_ckpt[0]).intersection(ckpt_files[1])) == 0
    else:
        # Only the chief worker should upload files in the non-sharded case.
        assert len(list(upload_ckpt[1])) == 0


@pytest.mark.parametrize("sharded", [True, False])
def test_store_path(sharded: bool, tmp_path: pathlib.Path) -> None:
    ckpt_dir = tmp_path.joinpath("ckpt-dir")
    ckpt_dir.mkdir(exist_ok=True)

    # Create some mock files for each worker to upload, and also identical files across all
    # workers to test the sharded conflict case.
    ckpt_files = {
        0: ["worker0-0", "worker0-1"],
        1: ["worker1-0", "worker1-1"],
    }

    all_workers_files = ["metadata.json", "file2", "file3"]

    with parallel.Execution(2) as pex:

        @pex.run
        def do_store_path() -> None:
            storage_manager = make_mock_storage_manager(
                basedir=ckpt_dir, dir_files=ckpt_files[0] + ckpt_files[1] + all_workers_files
            )

            storage_manager.store_path_is_direct_access = mock.MagicMock(return_value=False)
            checkpoint_context = core.DummyCheckpointContext(pex.distributed, storage_manager)

            # Upload across all workers, expect exceptions on non-chief workers if shard=False.
            with parallel.raises_when(
                not sharded and pex.distributed.rank != 0,
                RuntimeError,
                match=r"\.store_path.*non-chief",
            ):
                with checkpoint_context.store_path(
                    metadata={"steps_completed": 1}, shard=sharded
                ) as (ckpt_path, storage_id):
                    for f in ckpt_files[pex.rank] + all_workers_files:
                        (ckpt_path / f).touch()

            if not sharded:
                if pex.rank == 0:
                    storage_manager.store_path.assert_called_once()
                    storage_manager.store_path.reset_mock()
                    storage_manager._list_directory.assert_called_once()
                    storage_manager._list_directory.reset_mock()
                else:
                    storage_manager.store_path.assert_not_called()
                    storage_manager._list_directory.assert_not_called()
            else:
                # In the sharded case, the chief worker should upload all files written by each
                # worker, merging duplicate, conflicting files.
                storage_manager.pre_store_path.assert_called_once()
                storage_manager.pre_store_path.reset_mock()
                if pex.rank == 0:
                    storage_manager.post_store_path.assert_called_once()
                    _, call_kwargs = storage_manager.post_store_path.call_args_list[0]
                    uploaded_paths = call_kwargs["paths"]
                    assert sorted(uploaded_paths) == sorted(
                        ckpt_files[0] + ckpt_files[1] + all_workers_files
                    )
                    storage_manager.post_store_path.reset_mock()
                    storage_manager._list_directory.assert_called_once()
                    storage_manager._list_directory.reset_mock()
                else:
                    storage_manager.post_store_path.assert_not_called()
                    storage_manager._list_directory.assert_not_called()
