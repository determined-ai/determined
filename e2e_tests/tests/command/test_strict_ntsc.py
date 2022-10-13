import pytest

from tests import command as cmd
from pathlib import Path
import determined as det
import time
from typing import Any, List


from tests import experiment as exp
from determined.common import api
from determined.common.api import authentication, bindings
from determined.common.api.errors import NotFoundException, APIException
from tests.cluster.test_users import ADMIN_CREDENTIALS, create_test_user, logged_in_user
from tests import config as conf


# TODO task logs
# AND /tasks

def assert_shell_access(creds: authentication.Credentials, shell_id: str, can_access: bool) -> None:
    master_url = conf.make_master_url()
    authentication.cli_auth = authentication.Authentication(
        master_url, creds.username, creds.password, try_reauth=True)    
    sess = api.Session(master_url, None, None, None)

    resp = bindings.get_GetShells(sess)
    found = any([True for shell in resp.shells if shell.id == shell_id])
    assert (found and can_access,
            f"checking if shell is returned in GetShells found: {found} expected: {can_access}")
    
    if can_access:
        assert bindings.get_GetShell(sess, shellId=shell_id) is not None
        req = bindings.v1SetShellPriorityRequest(shellId=shell_id, priority=50)
        assert bindings.post_SetShellPriority(sess, shellId=shell_id, body=req) is not None
        return
    
    with pytest.raises(NotFoundException):
        bindings.get_GetShell(sess, shellId=shell_id)
    with pytest.raises(NotFoundException):        
        bindings.post_KillShell(sess, shellId=shell_id)
    with pytest.raises(NotFoundException):
        req = bindings.v1SetShellPriorityRequest(shellId=shell_id, priority=50)
        bindings.post_SetShellPriority(sess, shellId=shell_id, body=req)        

        
def assert_notebook_access(creds: authentication.Credentials, notebook_id: str,
                           can_access: bool) -> None:
    master_url = conf.make_master_url()
    authentication.cli_auth = authentication.Authentication(
        master_url, creds.username, creds.password, try_reauth=True)    
    sess = api.Session(master_url, None, None, None)

    resp = bindings.get_GetNotebooks(sess)
    found = any([True for notebook in resp.notebooks if notebook.id == notebook_id])
    assert (found and can_access,
            f"checking if notebook is returned in GetNotebooks found: {found} expected: {can_access}")

    if can_access:
        assert bindings.get_GetNotebook(sess, notebookId=notebook_id) is not None
        req = bindings.v1IdleNotebookRequest(idle=False, notebookId=notebook_id)
        bindings.put_IdleNotebook(sess, notebookId=notebook_id, body=req)

        req = bindings.v1SetNotebookPriorityRequest(notebookId=notebook_id, priority=50)
        assert bindings.post_SetNotebookPriority(sess, notebookId=notebook_id, body=req) is not None

        with api.ws(master_url, f"notebooks/{notebook_id}/events") as ws:
            for msg in ws:
                if msg["service_ready_event"]:
                    assert api.get(
                        master_url, f"/proxy/{notebook_id}/", auth=authentication.cli_auth) is not None
                    break
        return

    with pytest.raises(NotFoundException):
        bindings.get_GetNotebook(sess, notebookId=notebook_id)    
    with pytest.raises(NotFoundException):
        req = bindings.v1IdleNotebookRequest(idle=False, notebookId=notebook_id)
        bindings.put_IdleNotebook(sess, notebookId=notebook_id, body=req)
    with pytest.raises(NotFoundException):
        bindings.post_KillNotebook(sess, notebookId=notebook_id)        
    with pytest.raises(NotFoundException):        
        req = bindings.v1SetNotebookPriorityRequest(notebookId=notebook_id, priority=50)
        bindings.post_SetNotebookPriority(sess, notebookId=notebook_id, body=req) 
    with pytest.raises(NotFoundException):
        api.get(master_url, f"/proxy/{notebook_id}/", auth=authentication.cli_auth)
        
        

def assert_command_access(creds: authentication.Credentials, command_id: str,
                           can_access: bool) -> None:
    master_url = conf.make_master_url()
    authentication.cli_auth = authentication.Authentication(
        master_url, creds.username, creds.password, try_reauth=True)    
    sess = api.Session(master_url, None, None, None)

    resp = bindings.get_GetCommands(sess)
    found = any([True for command in resp.commands if command.id == command_id])
    assert (found and can_access,
            f"checking if command is returned in GetCommands found: {found} expected: {can_access}")

    if can_access:
        assert bindings.get_GetCommand(sess, commandId=command_id) is not None

        req = bindings.v1SetCommandPriorityRequest(commandId=command_id, priority=50)
        assert bindings.post_SetCommandPriority(sess, commandId=command_id, body=req) is not None
        return

    
    with pytest.raises(NotFoundException):
        bindings.get_GetCommand(sess, commandId=command_id)    
    with pytest.raises(NotFoundException):
        bindings.post_KillCommand(sess, commandId=command_id)        
    with pytest.raises(NotFoundException):        
        req = bindings.v1SetCommandPriorityRequest(commandId=command_id, priority=50)
        bindings.post_SetCommandPriority(sess, commandId=command_id, body=req) 

def assert_tensorboard_access(creds: authentication.Credentials, tensorboard_id: str,
                           can_access: bool) -> None:
    master_url = conf.make_master_url()
    authentication.cli_auth = authentication.Authentication(
        master_url, creds.username, creds.password, try_reauth=True)    
    sess = api.Session(master_url, None, None, None)

    resp = bindings.get_GetTensorboards(sess)
    found = any([True for tensorboard in resp.tensorboards if tensorboard.id == tensorboard_id])
    assert (found and can_access,
            f"checking if tensorboard is returned in GetTensorboards found: " +
            "{found} expected: {can_access}")

    
    # TODO proxy test.
    
    if can_access:
        assert bindings.get_GetTensorboard(sess, tensorboardId=tensorboard_id) is not None

        req = bindings.v1SetTensorboardPriorityRequest(tensorboardId=tensorboard_id, priority=50)
        assert bindings.post_SetTensorboardPriority(
            sess, tensorboardId=tensorboard_id, body=req) is not None

        with api.ws(master_url, f"tensorboard/{tensorboard_id}/events") as ws:
            for msg in ws:
                if msg["service_ready_event"]:
                    assert api.get(
                        master_url, f"/proxy/{tensorboard_id}/", auth=authentication.cli_auth) is not None
                    break
        
        return

    with pytest.raises(NotFoundException):
        bindings.get_GetTensorboard(sess, tensorboardId=tensorboard_id)    
    with pytest.raises(NotFoundException):
        bindings.post_KillTensorboard(sess, tensorboardId=tensorboard_id)        
    with pytest.raises(NotFoundException):        
        req = bindings.v1SetTensorboardPriorityRequest(tensorboardId=tensorboard_id, priority=50)
        bindings.post_SetTensorboardPriority(sess, tensorboardId=tensorboard_id, body=req) 
    with pytest.raises(NotFoundException):
        api.get(master_url, f"/proxy/{tensorboard_id}/", auth=authentication.cli_auth)
    
        
def strict_task_test(start_command: List[str], start_message: str, assert_access_func: Any) -> None:
    user_a = create_test_user(ADMIN_CREDENTIALS)
    user_b = create_test_user(ADMIN_CREDENTIALS)    
    
    with logged_in_user(user_a):
        with cmd.interactive_command(*start_command) as task:
            for line in task.stdout:
                if start_message in line:
                    break
            else:
                pytest.fail(f"Did not find expected input '{start_message}' in task stdout.")        
                
            assert_access_func(user_a, task.task_id, True)
            assert_access_func(user_b, task.task_id, False)
            assert_access_func(ADMIN_CREDENTIALS, task.task_id, True)

# TODO mark
# stress_test
@pytest.mark.stress_test            
def test_strict_shell():
    strict_task_test(["shell", "start"], "has started...", assert_shell_access)


# TODO mark
# stress_test
@pytest.mark.stress_test            
def test_strict_notebook():
    strict_task_test(["notebook", "start", "--no-browser"],
                     "has started...", assert_notebook_access)    

# TODO mark
# stress_test
@pytest.mark.stress_test            
def test_strict_command():
    strict_task_test(["command", "run", "sleep", "300"], "have started", assert_command_access)

# TODO mark
# stress_test
@pytest.mark.stress_test            
def test_strict_tensorboard():
    exp_id = exp.run_basic_test(
        conf.fixtures_path("no_op/single-one-short-step.yaml"),
        conf.fixtures_path("no_op"),
        1,
    )
    strict_task_test(["tensorboard", "start", str(exp_id), "--no-browser"],
                     "has started", assert_tensorboard_access)

    
    
            
