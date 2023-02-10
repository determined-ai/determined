import pathlib
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
        env.det_cluster_id, env.det_experiment_id, env.det_trial_id, checkpoint_config
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
        env.det_cluster_id, env.det_experiment_id, env.det_trial_id, checkpoint_config
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
        env.det_cluster_id, env.det_experiment_id, env.det_trial_id, checkpoint_config
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
            env.det_cluster_id, env.det_experiment_id, env.det_trial_id, checkpoint_config
        )


def test_missing_type() -> None:
    with pytest.raises(TypeError, match="Missing 'type' parameter"):
        env = test_util.get_dummy_env()
        tensorboard.build(env.det_cluster_id, env.det_experiment_id, env.det_trial_id, {})


def test_illegal_type() -> None:
    checkpoint_config = {"type": 4}
    with pytest.raises(TypeError, match="must be a string"):
        env = test_util.get_dummy_env()
        tensorboard.build(
            env.det_cluster_id, env.det_experiment_id, env.det_trial_id, checkpoint_config
        )


def test_upload_thread_normal_case() -> None:
    upload_function = mock.Mock()
    upload_thread = tensorboard.base._TensorboardUploadThread(upload_function)

    upload_thread.start()

    path_info_list_1 = []
    path_info_list_2 = []
    path_info_list_1.append(
        tensorboard.base.PathUploadInfo(
            path="test_value/file1.json", mangled_relative_path="test_value/file1#1.json"
        )
    )
    path_info_list_2.append(
        tensorboard.base.PathUploadInfo(
            path="test_value/file2.json", mangled_relative_path="test_value/file2#1.json"
        )
    )

    upload_thread.upload(path_info_list_1)
    upload_thread.upload(path_info_list_2)

    upload_thread.close()

    assert upload_function.call_count == 2
