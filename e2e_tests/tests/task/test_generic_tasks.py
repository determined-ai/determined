import subprocess

import pytest

from determined.cli import ntsc
from determined.common import util
from determined.common.api import bindings
from tests import api_utils
from tests import config as conf
from tests.task import task


@pytest.mark.e2e_cpu
def test_create_generic_task() -> None:
    """
    Start a simple task with a context directory called from the task CLI
    """
    command = [
        "det",
        "-m",
        conf.make_master_url(),
        "task",
        "create",
        conf.fixtures_path("generic_task/test_config.yaml"),
        "--context",
        conf.fixtures_path("generic_task"),
    ]

    subprocess.run(command, universal_newlines=True, stdout=subprocess.PIPE, check=True)


@pytest.mark.e2e_cpu
def test_generic_task_completion() -> None:
    """
    Start a simple task and check for task completion
    """
    test_session = api_utils.determined_test_session()

    with open(conf.fixtures_path("generic_task/test_config.yaml"), "r") as config_file:
        # Create task
        config_text = config_file.read()

    req = bindings.v1CreateGenericTaskRequest(
        config=config_text,
        contextDirectory=[],
        projectId=None,
        forkedFrom=None,
        parentId=None,
        inheritContext=False,
        noPause=False,
    )
    task_resp = bindings.post_CreateGenericTask(test_session, body=req)

    # Check for complete state
    is_valid_state = task.wait_for_task_state(
        test_session, task_resp.taskId, bindings.v1GenericTaskState.COMPLETED, timeout=30
    )
    if not is_valid_state:
        pytest.fail("task failed to complete after 30 seconds")


@pytest.mark.e2e_cpu
def test_create_generic_task_error() -> None:
    """
    Start a simple task that fails and check for error task state
    """
    test_session = api_utils.determined_test_session()

    with open(conf.fixtures_path("generic_task/test_config_error.yaml"), "r") as config_file:
        # Create task
        config_text = config_file.read()

    req = bindings.v1CreateGenericTaskRequest(
        config=config_text,
        contextDirectory=[],
        projectId=None,
        forkedFrom=None,
        parentId=None,
        inheritContext=False,
        noPause=False,
    )
    task_resp = bindings.post_CreateGenericTask(test_session, body=req)

    # Check for error state
    is_valid_state = task.wait_for_task_state(
        test_session, task_resp.taskId, bindings.v1GenericTaskState.ERROR, timeout=30
    )
    if not is_valid_state:
        pytest.fail("task failed to complete after 30 seconds")


@pytest.mark.e2e_cpu
def test_generic_task_config() -> None:
    """
    Start a simple task without a context directory and grab its config
    """
    test_session = api_utils.determined_test_session()

    with open(conf.fixtures_path("generic_task/test_config.yaml"), "r") as config_file:
        # Create task
        config_text = config_file.read()

    req = bindings.v1CreateGenericTaskRequest(
        config=config_text,
        contextDirectory=[],
        projectId=None,
        forkedFrom=None,
        parentId=None,
        inheritContext=False,
        noPause=False,
    )
    task_resp = bindings.post_CreateGenericTask(test_session, body=req)

    # Get config
    command = ["det", "-m", conf.make_master_url(), "task", "config", task_resp.taskId]

    res = subprocess.run(command, universal_newlines=True, stdout=subprocess.PIPE, check=True)

    result_config = util.yaml_safe_load(res.stdout)
    expected_config = {"entrypoint": ["echo", "task ran"]}
    assert result_config == expected_config


@pytest.mark.e2e_cpu
def test_generic_task_create_with_fork() -> None:
    """
    Start a simple task without a context directory and grab its config
    """
    test_session = api_utils.determined_test_session()

    with open(conf.fixtures_path("generic_task/test_config.yaml"), "r") as config_file:
        # Create initial task
        config = ntsc.parse_config(config_file, None, [], [])
    config_text = util.yaml_safe_dump(config)

    req = bindings.v1CreateGenericTaskRequest(
        config=config_text,
        contextDirectory=[],
        projectId=None,
        forkedFrom=None,
        parentId=None,
        inheritContext=False,
        noPause=False,
    )
    task_resp = bindings.post_CreateGenericTask(test_session, body=req)

    # Create fork task
    with open(conf.fixtures_path("generic_task/test_config_fork.yaml"), "r") as fork_config_file:
        config = ntsc.parse_config(fork_config_file, None, [], [])
    config_text = util.yaml_safe_dump(config)

    req = bindings.v1CreateGenericTaskRequest(
        config=config_text,
        contextDirectory=[],
        projectId=None,
        forkedFrom=task_resp.taskId,
        parentId=None,
        inheritContext=False,
        noPause=False,
    )
    fork_task_resp = bindings.post_CreateGenericTask(test_session, body=req)

    # Get fork task Config
    command = ["det", "-m", conf.make_master_url(), "task", "config", fork_task_resp.taskId]

    res = subprocess.run(command, universal_newlines=True, stdout=subprocess.PIPE, check=True)
    result_config = util.yaml_safe_load(res.stdout)
    expected_config = {"entrypoint": ["echo", "forked"]}
    assert result_config == expected_config


@pytest.mark.e2e_cpu
def test_kill_generic_task() -> None:
    """
    Start a simple task without a context directory and grab its config
    """
    test_session = api_utils.determined_test_session()

    with open(conf.fixtures_path("generic_task/test_config.yaml"), "r") as config_file:
        # Create task
        config = ntsc.parse_config(config_file, None, [], [])
    config_text = util.yaml_safe_dump(config)

    req = bindings.v1CreateGenericTaskRequest(
        config=config_text,
        contextDirectory=[],
        projectId=None,
        forkedFrom=None,
        parentId=None,
        inheritContext=False,
        noPause=False,
    )
    task_resp = bindings.post_CreateGenericTask(test_session, body=req)

    # Kill task
    command = ["det", "-m", conf.make_master_url(), "task", "kill", task_resp.taskId]

    subprocess.run(command, universal_newlines=True, stdout=subprocess.PIPE, check=True)

    kill_resp = bindings.get_GetTask(test_session, taskId=task_resp.taskId)
    assert kill_resp.task.taskState == bindings.v1GenericTaskState.CANCELED


@pytest.mark.e2e_cpu
def test_pause_and_unpause_generic_task() -> None:
    """
    Start a simple task without a context directory and grab its config
    """
    test_session = api_utils.determined_test_session()

    with open(conf.fixtures_path("generic_task/test_config_pause.yaml"), "r") as config_file:
        # Create task
        config = ntsc.parse_config(config_file, None, [], [])
    config_text = util.yaml_safe_dump(config)

    req = bindings.v1CreateGenericTaskRequest(
        config=config_text,
        contextDirectory=[],
        projectId=None,
        forkedFrom=None,
        parentId=None,
        inheritContext=False,
        noPause=False,
    )
    task_resp = bindings.post_CreateGenericTask(test_session, body=req)

    # Pause task
    command = ["det", "-m", conf.make_master_url(), "task", "pause", task_resp.taskId]

    subprocess.run(command, universal_newlines=True, stdout=subprocess.PIPE, check=True)

    pause_resp = bindings.get_GetTask(test_session, taskId=task_resp.taskId)
    assert pause_resp.task.taskState == bindings.v1GenericTaskState.PAUSED

    # Unpause task
    command = ["det", "-m", conf.make_master_url(), "task", "unpause", task_resp.taskId]

    subprocess.run(command, universal_newlines=True, stdout=subprocess.PIPE, check=True)

    unpause_resp = bindings.get_GetTask(test_session, taskId=task_resp.taskId)
    assert unpause_resp.task.taskState == bindings.v1GenericTaskState.ACTIVE
