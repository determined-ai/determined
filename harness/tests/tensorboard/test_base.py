import pathlib
import time
from typing import List
from unittest import mock

import pytest

from determined import tensorboard
from tests.tensorboard import test_util

HOST_PATH = pathlib.Path(__file__).resolve().parent.joinpath("test_tensorboard_host")
STORAGE_PATH = HOST_PATH.joinpath("test_storage_path")
BASE_PATH = pathlib.Path(__file__).resolve().parent.joinpath("fixtures")


def test_getting_manager_instance(tmp_path: pathlib.Path) -> None:
    checkpoint_config = {"type": "shared_fs", "host_path": HOST_PATH}
    env = test_util.get_dummy_env()
    manager = tensorboard.build(
        env.det_cluster_id,
        env.det_experiment_id,
        env.det_trial_id,
        checkpoint_config,
        async_upload=False,
    )
    assert isinstance(manager, tensorboard.SharedFSTensorboardManager)


def test_setting_optional_variable(tmp_path: pathlib.Path) -> None:
    checkpoint_config = {
        "type": "shared_fs",
        "base_path": "test_value",
        "host_path": HOST_PATH,
    }
    env = test_util.get_dummy_env()
    manager = tensorboard.build(
        env.det_cluster_id,
        env.det_experiment_id,
        env.det_trial_id,
        checkpoint_config,
        async_upload=False,
    )
    assert isinstance(manager, tensorboard.SharedFSTensorboardManager)
    assert manager.base_path == pathlib.Path("test_value/tensorboard--0")


def test_build_with_container_path(tmp_path: pathlib.Path) -> None:
    checkpoint_config = {
        "type": "shared_fs",
        "host_path": str(HOST_PATH),
        "storage_path": str(STORAGE_PATH),
    }
    env = test_util.get_dummy_env()
    manager = tensorboard.build(
        env.det_cluster_id,
        env.det_experiment_id,
        env.det_trial_id,
        checkpoint_config,
        container_path=str(tmp_path),
        async_upload=False,
    )
    assert isinstance(manager, tensorboard.SharedFSTensorboardManager)
    assert manager.storage_path == tmp_path.joinpath("test_storage_path")


def test_setting_storage_path(tmp_path: pathlib.Path) -> None:
    checkpoint_config = {
        "type": "shared_fs",
        "host_path": str(HOST_PATH),
        "storage_path": str(STORAGE_PATH),
    }
    env = test_util.get_dummy_env()
    manager = tensorboard.build(
        env.det_cluster_id,
        env.det_experiment_id,
        env.det_trial_id,
        checkpoint_config,
        async_upload=False,
    )
    assert isinstance(manager, tensorboard.SharedFSTensorboardManager)
    assert manager.storage_path == STORAGE_PATH


def test_unknown_type() -> None:
    checkpoint_config = {
        "type": "unknown",
        "host_path": HOST_PATH,
    }
    with pytest.raises(TypeError, match="Unknown storage type: unknown"):
        env = test_util.get_dummy_env()
        tensorboard.build(
            env.det_cluster_id,
            env.det_experiment_id,
            env.det_trial_id,
            checkpoint_config,
            async_upload=False,
        )


def test_missing_type() -> None:
    with pytest.raises(TypeError, match="Missing 'type' parameter"):
        env = test_util.get_dummy_env()
        tensorboard.build(
            env.det_cluster_id, env.det_experiment_id, env.det_trial_id, {}, async_upload=False
        )


def test_illegal_type() -> None:
    checkpoint_config = {"type": 4}
    with pytest.raises(TypeError, match="must be a string"):
        env = test_util.get_dummy_env()
        tensorboard.build(
            env.det_cluster_id,
            env.det_experiment_id,
            env.det_trial_id,
            checkpoint_config,
            async_upload=False,
        )


def test_upload_thread_normal_case() -> None:
    upload_function = mock.Mock()
    upload_thread = tensorboard.base._TensorboardUploadThread(upload_function)

    upload_thread.start()

    path_info_list_1 = []
    path_info_list_2 = []
    path_info_list_1.append(
        tensorboard.base.PathUploadInfo(
            path=pathlib.Path("test_value/file1.json"),
            mangled_relative_path=pathlib.Path("test_value/file1#1.json"),
        )
    )
    path_info_list_2.append(
        tensorboard.base.PathUploadInfo(
            path=pathlib.Path("test_value/file2.json"),
            mangled_relative_path=pathlib.Path("test_value/file2#1.json"),
        )
    )

    upload_thread.upload(path_info_list_1)
    upload_thread.upload(path_info_list_2)

    upload_thread.close()

    assert upload_function.call_count == 2


class MockTensorBoardManager(tensorboard.TensorboardManager):
    def __init__(
        self,
        base_path: pathlib.Path,
        sync_path: pathlib.Path,
        async_upload: bool = True,
        sync_on_close: bool = True,
    ) -> None:
        super().__init__(
            base_path=base_path,
            sync_path=sync_path,
            async_upload=async_upload,
            sync_on_close=sync_on_close,
        )
        # Record timestamps `self._sync_impl` is called for mock tests.
        self._sync_times: List[float] = []

    def _sync_impl(self, path_info_list: List[tensorboard.base.PathUploadInfo]) -> None:
        self._sync_times.append(time.time())

    def delete(self) -> None:
        return


def test_sync_throttles_uploads() -> None:
    """
    Test that `TensorBoardManager.sync()` throttles uploads to a maximum of once per second.
    """
    mock_tensorboard_manager = MockTensorBoardManager(
        base_path=BASE_PATH, sync_path=pathlib.Path("mock-tb-sync-path")
    )

    # Call `sync()` at different time intervals.
    for _ in range(5):
        mock_tensorboard_manager.sync()

    for _ in range(5):
        mock_tensorboard_manager.sync()
        time.sleep(0.5)

    # Verify that the recorded sync times are all at least 1 second apart.
    sync_times = mock_tensorboard_manager._sync_times
    assert len(sync_times) > 1
    sync_intervals = [sync_times[i] - sync_times[i - 1] for i in range(1, len(sync_times))]
    assert all(sync_interval > 1 for sync_interval in sync_intervals)
