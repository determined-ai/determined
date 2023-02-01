import queue
import pathlib

import pytest

from unittest import mock

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


def test_upload_thread() -> None:
    upload_function = mock.Mock()
    work_queue = queue.Queue(maxsize=10)
    upload_thread = tensorboard.base._TensorboardUploadThread(upload_function, work_queue)

    upload_thread.start()
    work_queue.put(["test_file_path_1", "test_file_path_2"])
    work_queue.put(["test_file_path_3"])
    # Pass in sentinel value to exit thread
    work_queue.put(None)
    upload_thread.join()

    assert upload_function.call_count == 2
