from typing import Any, List

import pytest

from determined.common import api
from determined.common.api import authentication, bindings
from determined.common.api.errors import APIException, BadRequestException, NotFoundException
from tests import command as cmd
from tests import config as conf
from tests import experiment as exp
from tests.cluster.test_users import (
    ADMIN_CREDENTIALS,
    create_test_user,
    log_in_user,
    logged_in_user,
)


def assert_shell_access(creds: authentication.Credentials, shell_id: str, can_access: bool) -> None:
    master_url = conf.make_master_url()
    authentication.cli_auth = authentication.Authentication(
        master_url, creds.username, creds.password
    )
    sess = api.Session(master_url, None, None, None)

    resp = bindings.get_GetShells(sess)
    shell_ids = [shell.id for shell in resp.shells or []]
    if can_access:
        assert shell_id in shell_ids
        bindings.get_GetShell(sess, shellId=shell_id)
        req = bindings.v1SetShellPriorityRequest(shellId=shell_id, priority=50)
        bindings.post_SetShellPriority(sess, shellId=shell_id, body=req)
        api.get(master_url, f"shells/{shell_id}/events", auth=authentication.cli_auth)
        with api.ws(master_url, f"shells/{shell_id}/events") as ws:
            for _ in ws:
                break
        return

    assert shell_id not in shell_ids
    with pytest.raises(NotFoundException):
        bindings.get_GetShell(sess, shellId=shell_id)
    with pytest.raises(NotFoundException):
        bindings.post_KillShell(sess, shellId=shell_id)
    with pytest.raises(NotFoundException):
        req = bindings.v1SetShellPriorityRequest(shellId=shell_id, priority=50)
        bindings.post_SetShellPriority(sess, shellId=shell_id, body=req)
    with pytest.raises(NotFoundException):
        api.get(master_url, f"shells/{shell_id}/events", auth=authentication.cli_auth)
    with pytest.raises(BadRequestException):
        with api.ws(master_url, f"shells/{shell_id}/events") as ws:
            for _ in ws:
                break


def assert_notebook_access(
    creds: authentication.Credentials, notebook_id: str, can_access: bool
) -> None:
    master_url = conf.make_master_url()
    authentication.cli_auth = authentication.Authentication(
        master_url, creds.username, creds.password
    )
    sess = api.Session(master_url, None, None, None)

    resp = bindings.get_GetNotebooks(sess)
    notebook_ids = [notebook.id for notebook in resp.notebooks or []]
    if can_access:
        assert notebook_id in notebook_ids
        assert bindings.get_GetNotebook(sess, notebookId=notebook_id) is not None
        req = bindings.v1IdleNotebookRequest(idle=False, notebookId=notebook_id)
        bindings.put_IdleNotebook(sess, notebookId=notebook_id, body=req)

        pri_req = bindings.v1SetNotebookPriorityRequest(notebookId=notebook_id, priority=50)
        bindings.post_SetNotebookPriority(sess, notebookId=notebook_id, body=pri_req)

        api.get(master_url, f"notebooks/{notebook_id}/events", auth=authentication.cli_auth)
        with api.ws(master_url, f"notebooks/{notebook_id}/events") as ws:
            for _ in ws:
                break
        return

    assert notebook_id not in notebook_ids
    with pytest.raises(NotFoundException):
        bindings.get_GetNotebook(sess, notebookId=notebook_id)
    with pytest.raises(NotFoundException):
        req = bindings.v1IdleNotebookRequest(idle=False, notebookId=notebook_id)
        bindings.put_IdleNotebook(sess, notebookId=notebook_id, body=req)
    with pytest.raises(NotFoundException):
        bindings.post_KillNotebook(sess, notebookId=notebook_id)
    with pytest.raises(NotFoundException):
        pri_req = bindings.v1SetNotebookPriorityRequest(notebookId=notebook_id, priority=50)
        bindings.post_SetNotebookPriority(sess, notebookId=notebook_id, body=pri_req)
    with pytest.raises(NotFoundException):
        api.get(master_url, f"/proxy/{notebook_id}/", auth=authentication.cli_auth)
    with pytest.raises(NotFoundException):
        api.get(master_url, f"notebooks/{notebook_id}/events", auth=authentication.cli_auth)
    with pytest.raises(BadRequestException):
        with api.ws(master_url, f"notebooks/{notebook_id}/events") as ws:
            for _ in ws:
                break


def assert_command_access(
    creds: authentication.Credentials, command_id: str, can_access: bool
) -> None:
    master_url = conf.make_master_url()
    authentication.cli_auth = authentication.Authentication(
        master_url, creds.username, creds.password
    )
    sess = api.Session(master_url, None, None, None)

    resp = bindings.get_GetCommands(sess)
    command_ids = [command.id for command in resp.commands or []]
    if can_access:
        assert command_id in command_ids
        bindings.get_GetCommand(sess, commandId=command_id)

        req = bindings.v1SetCommandPriorityRequest(commandId=command_id, priority=50)
        bindings.post_SetCommandPriority(sess, commandId=command_id, body=req)

        api.get(master_url, f"commands/{command_id}/events", auth=authentication.cli_auth)
        with api.ws(master_url, f"commands/{command_id}/events") as ws:
            for _ in ws:
                break
        return

    assert command_id not in command_ids
    with pytest.raises(NotFoundException):
        bindings.get_GetCommand(sess, commandId=command_id)
    with pytest.raises(NotFoundException):
        bindings.post_KillCommand(sess, commandId=command_id)
    with pytest.raises(NotFoundException):
        req = bindings.v1SetCommandPriorityRequest(commandId=command_id, priority=50)
        bindings.post_SetCommandPriority(sess, commandId=command_id, body=req)
    with pytest.raises(NotFoundException):
        api.get(master_url, f"commands/{command_id}/events", auth=authentication.cli_auth)
    with pytest.raises(BadRequestException):
        with api.ws(master_url, f"commands/{command_id}/events") as ws:
            for _ in ws:
                break


def assert_tensorboard_access(
    creds: authentication.Credentials, tensorboard_id: str, can_access: bool
) -> None:
    master_url = conf.make_master_url()
    authentication.cli_auth = authentication.Authentication(
        master_url, creds.username, creds.password
    )
    sess = api.Session(master_url, None, None, None)

    resp = bindings.get_GetTensorboards(sess)
    tensorboard_ids = [tensorboard.id for tensorboard in resp.tensorboards or []]
    if can_access:
        assert tensorboard_id in tensorboard_ids
        bindings.get_GetTensorboard(sess, tensorboardId=tensorboard_id)

        req = bindings.v1SetTensorboardPriorityRequest(tensorboardId=tensorboard_id, priority=50)
        bindings.post_SetTensorboardPriority(sess, tensorboardId=tensorboard_id, body=req)

        api.get(master_url, f"tensorboard/{tensorboard_id}/events", auth=authentication.cli_auth)
        with api.ws(master_url, f"tensorboard/{tensorboard_id}/events") as ws:
            for _ in ws:
                break
        return

    assert tensorboard_id not in tensorboard_ids
    with pytest.raises(NotFoundException):
        bindings.get_GetTensorboard(sess, tensorboardId=tensorboard_id)
    with pytest.raises(NotFoundException):
        bindings.post_KillTensorboard(sess, tensorboardId=tensorboard_id)
    with pytest.raises(NotFoundException):
        req = bindings.v1SetTensorboardPriorityRequest(tensorboardId=tensorboard_id, priority=50)
        bindings.post_SetTensorboardPriority(sess, tensorboardId=tensorboard_id, body=req)
    with pytest.raises(NotFoundException):
        api.get(master_url, f"/proxy/{tensorboard_id}/", auth=authentication.cli_auth)
    with pytest.raises(NotFoundException):
        api.get(master_url, f"tensorboard/{tensorboard_id}/events", auth=authentication.cli_auth)
    with pytest.raises(BadRequestException):
        with api.ws(master_url, f"tensorboard/{tensorboard_id}/events") as ws:
            for _ in ws:
                break


def assert_access_task(creds: authentication.Credentials, task_id: str, can_access: bool) -> None:
    master_url = conf.make_master_url()
    authentication.cli_auth = authentication.Authentication(
        master_url, creds.username, creds.password
    )
    sess = api.Session(master_url, None, None, None)

    resp = api.get(master_url, "/tasks", auth=authentication.cli_auth).json()
    task_ids = [resp[alloc]["task_id"] for alloc in resp]
    if can_access:
        assert task_id in task_ids
        bindings.get_GetTask(sess, taskId=task_id)
        bindings.get_TaskLogs(sess, taskId=task_id, follow=False)
        bindings.get_TaskLogsFields(sess, taskId=task_id, follow=False)
        return

    assert task_id not in task_ids
    with pytest.raises(NotFoundException):
        bindings.get_GetTask(sess, taskId=task_id)
    with pytest.raises(APIException):
        for _ in bindings.get_TaskLogs(sess, taskId=task_id, follow=False):
            pass
    with pytest.raises(APIException):
        for _ in bindings.get_TaskLogsFields(sess, taskId=task_id, follow=False):
            pass
    with pytest.raises(NotFoundException):
        alloc_id = task_id + ".1"
        req = bindings.v1AllocationReadyRequest(allocationId=alloc_id)
        bindings.post_AllocationReady(sess, allocationId=alloc_id, body=req)


def strict_task_test(start_command: List[str], start_message: str, assert_access_func: Any) -> None:
    log_in_user(ADMIN_CREDENTIALS)
    user_a = create_test_user()
    user_b = create_test_user()

    with logged_in_user(user_a):
        with cmd.interactive_command(*start_command) as task:
            for line in task.stdout:
                if start_message in line:
                    break
            else:
                pytest.fail(f"Did not find expected input '{start_message}' in task stdout.")

            assert task.task_id is not None

            assert_access_task(user_a, task.task_id, True)
            assert_access_task(user_b, task.task_id, False)
            assert_access_task(ADMIN_CREDENTIALS, task.task_id, True)

            assert_access_func(user_a, task.task_id, True)
            assert_access_func(user_b, task.task_id, False)
            assert_access_func(ADMIN_CREDENTIALS, task.task_id, True)


@pytest.mark.test_strict_ntsc
def test_strict_shell() -> None:
    strict_task_test(["shell", "start"], "has started...", assert_shell_access)


@pytest.mark.test_strict_ntsc
def test_strict_notebook() -> None:
    strict_task_test(
        ["notebook", "start", "--no-browser"], "has started...", assert_notebook_access
    )


@pytest.mark.test_strict_ntsc
def test_strict_command() -> None:
    strict_task_test(["command", "run", "sleep", "300"], "have started", assert_command_access)


@pytest.mark.test_strict_ntsc
def test_strict_tensorboard() -> None:
    exp_id = exp.run_basic_test(
        conf.fixtures_path("no_op/single-one-short-step.yaml"),
        conf.fixtures_path("no_op"),
        1,
    )
    strict_task_test(
        ["tensorboard", "start", str(exp_id), "--no-browser"],
        "has started",
        assert_tensorboard_access,
    )
