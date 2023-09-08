import pytest

from determined.cli import ntsc
from determined.common import util
from determined.common.api import bindings
from tests import api_utils
from tests import config as conf
from tests import detproc
from tests.task import task


@pytest.mark.e2e_cpu
def test_create_generic_task() -> None:
    """
    Start a simple task with a context directory called from the task CLI
    """
    sess = api_utils.user_session()
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

    output = detproc.check_output(sess, command)

    id_index = output.find("Created task ")
    task_id = output[id_index + len("Created task ") :].strip()

    task.wait_for_task_state(sess, task_id, bindings.v1GenericTaskState.COMPLETED)


@pytest.mark.e2e_cpu
def test_generic_task_completion() -> None:
    """
    Start a simple task and check for task completion
    """
    sess = api_utils.user_session()

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
    task_resp = bindings.post_CreateGenericTask(sess, body=req)

    # Check for complete state
    task.wait_for_task_state(sess, task_resp.taskId, bindings.v1GenericTaskState.COMPLETED)


@pytest.mark.e2e_cpu
def test_create_generic_task_error() -> None:
    """
    Start a simple task that fails and check for error task state
    """
    sess = api_utils.user_session()

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
    task_resp = bindings.post_CreateGenericTask(sess, body=req)

    # Check for error state
    task.wait_for_task_state(sess, task_resp.taskId, bindings.v1GenericTaskState.ERROR)


@pytest.mark.e2e_cpu
def test_generic_task_config() -> None:
    """
    Start a simple task without a context directory and grab its config
    """
    sess = api_utils.user_session()

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
    task_resp = bindings.post_CreateGenericTask(sess, body=req)

    # Get config
    command = ["det", "-m", conf.make_master_url(), "task", "config", task_resp.taskId]

    output = detproc.check_output(sess, command)

    result_config = util.yaml_safe_load(output)
    expected_config = {"entrypoint": ["echo", "task ran"]}
    assert result_config == expected_config

    task.wait_for_task_state(sess, task_resp.taskId, bindings.v1GenericTaskState.COMPLETED)


@pytest.mark.e2e_cpu
def test_generic_task_create_with_fork() -> None:
    """
    Start a simple task without a context directory and grab its config
    """
    sess = api_utils.user_session()

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
    task_resp = bindings.post_CreateGenericTask(sess, body=req)

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
    fork_task_resp = bindings.post_CreateGenericTask(sess, body=req)

    # Get fork task Config
    command = ["det", "-m", conf.make_master_url(), "task", "config", fork_task_resp.taskId]

    output = detproc.check_output(sess, command)
    result_config = util.yaml_safe_load(output)
    expected_config = {"entrypoint": ["echo", "forked"]}
    assert result_config == expected_config

    task.wait_for_task_state(sess, task_resp.taskId, bindings.v1GenericTaskState.COMPLETED)
    task.wait_for_task_state(sess, fork_task_resp.taskId, bindings.v1GenericTaskState.COMPLETED)


@pytest.mark.e2e_cpu
def test_kill_generic_task() -> None:
    """
    Start a simple task without a context directory and grab its config
    """
    sess = api_utils.user_session()

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
    task_resp = bindings.post_CreateGenericTask(sess, body=req)

    # Kill task
    command = ["det", "-m", conf.make_master_url(), "task", "kill", task_resp.taskId]

    detproc.check_call(sess, command)

    bindings.get_GetTask(sess, taskId=task_resp.taskId)
    task.wait_for_task_state(sess, task_resp.taskId, bindings.v1GenericTaskState.CANCELED)


@pytest.mark.e2e_cpu
def test_pause_and_unpause_generic_task() -> None:
    """
    Start a simple task without a context directory and grab its config
    """
    sess = api_utils.user_session()

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
    task_resp = bindings.post_CreateGenericTask(sess, body=req)

    # Pause task
    command = ["det", "-m", conf.make_master_url(), "task", "pause", task_resp.taskId]

    detproc.check_call(sess, command)

    pause_resp = bindings.get_GetTask(sess, taskId=task_resp.taskId)
    assert pause_resp.task.taskState == bindings.v1GenericTaskState.PAUSED

    # Unpause task
    command = ["det", "-m", conf.make_master_url(), "task", "unpause", task_resp.taskId]

    detproc.check_call(sess, command)

    unpause_resp = bindings.get_GetTask(sess, taskId=task_resp.taskId)
    assert unpause_resp.task.taskState == bindings.v1GenericTaskState.ACTIVE

    task.wait_for_task_state(sess, task_resp.taskId, bindings.v1GenericTaskState.COMPLETED)
