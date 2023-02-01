import queue
import pathlib

import pytest

from typing import List
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


def test_upload_thread_normal_case() -> None:
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


def test_upload_thread_exception_case() -> None:
    # 1. Set up custom hook to capture threads fail
    #    with exception. This hook is triggered when
    #    a thread exits with exception.
    import threading

    threads_with_exception = []

    def custom_excepthook(args):
        thread_name = args.thread.ident
        print(thread_name)
        threads_with_exception.append(thread_name)

    threading.excepthook = custom_excepthook

    # 2. Define function that throws exception
    def upload_function(paths: List[pathlib.Path]) -> None:
        raise Exception("An exception is raised")

    # 3. Set up a _TensorboardUploadThread instance
    work_queue = queue.Queue(maxsize=10)
    upload_thread = tensorboard.base._TensorboardUploadThread(upload_function, work_queue)

    # 4. start, run, and join the _TensorboardUploadThread instance
    upload_thread.start()
    thread_ident = upload_thread.ident
    work_queue.put(["test_file_path_1", "test_file_path_2"])
    # Pass in sentinel value to exit thread
    work_queue.put(None)
    upload_thread.join()

    # 5. Check that _TensorboardUploadThread instance did not
    #    throw exception
    assert thread_ident not in threads_with_exception
