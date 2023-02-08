import dataclasses
import time
from typing import Callable, List, Optional, Set, Union

from determined.common.api import Session, bindings

NTSC_TYPE = str  # Literal["notebook", "tensorboard", "shell", "command"]
all_ntsc: Set[NTSC_TYPE] = {"notebook", "shell", "command", "tensorboard"}
proxied_ntsc: Set[NTSC_TYPE] = {"notebook", "tensorboard"}


@dataclasses.dataclass
class SharedNTSC:
    """a shared class representing some common attributes among NTSC"""

    id_: str
    typ: str
    state: bindings.taskv1State


def launch_ntsc(session: Session, workspace_id: int, typ: str, exp_id: Optional[int] = None) -> str:
    assert typ in all_ntsc
    if typ == "notebook":
        return bindings.post_LaunchNotebook(
            session, body=bindings.v1LaunchNotebookRequest(workspaceId=workspace_id)
        ).notebook.id
    elif typ == "tensorboard":
        experiment_ids = [exp_id] if exp_id else []
        return bindings.post_LaunchTensorboard(
            session,
            body=bindings.v1LaunchTensorboardRequest(
                workspaceId=workspace_id, experimentIds=experiment_ids
            ),
        ).tensorboard.id
    elif typ == "shell":
        return bindings.post_LaunchShell(
            session, body=bindings.v1LaunchShellRequest(workspaceId=workspace_id)
        ).shell.id
    elif typ == "command":
        return bindings.post_LaunchCommand(
            session,
            body=bindings.v1LaunchCommandRequest(
                workspaceId=workspace_id,
                config={
                    "entrypoint": ["sleep", "100"],
                },
            ),
        ).command.id
    else:
        raise ValueError("unknown type")


def kill_ntsc(session: Session, typ: str, ntsc_id: str) -> None:
    assert typ in all_ntsc
    if typ == "notebook":
        bindings.post_KillNotebook(session, notebookId=ntsc_id)
    elif typ == "tensorboard":
        bindings.post_KillTensorboard(session, tensorboardId=ntsc_id)
    elif typ == "shell":
        bindings.post_KillShell(session, shellId=ntsc_id)
    elif typ == "command":
        bindings.post_KillCommand(session, commandId=ntsc_id)
    else:
        raise ValueError("unknown type")


def set_prio_ntsc(session: Session, typ: str, ntsc_id: str, prio: int) -> None:
    assert typ in all_ntsc
    if typ == "notebook":
        bindings.post_SetNotebookPriority(
            session, notebookId=ntsc_id, body=bindings.v1SetNotebookPriorityRequest(priority=prio)
        )
    elif typ == "tensorboard":
        bindings.post_SetTensorboardPriority(
            session,
            tensorboardId=ntsc_id,
            body=bindings.v1SetTensorboardPriorityRequest(priority=prio),
        )
    elif typ == "shell":
        bindings.post_SetShellPriority(
            session, shellId=ntsc_id, body=bindings.v1SetShellPriorityRequest(priority=prio)
        )
    elif typ == "command":
        bindings.post_SetCommandPriority(
            session, commandId=ntsc_id, body=bindings.v1SetCommandPriorityRequest(priority=prio)
        )
    else:
        raise ValueError("unknown type")


def get_ntsc_details(session: Session, typ: str, ntsc_id: str) -> SharedNTSC:
    assert typ in all_ntsc
    ntsc: Union[bindings.v1Notebook, bindings.v1Tensorboard, bindings.v1Shell, bindings.v1Command]
    if typ == "notebook":
        ntsc = bindings.get_GetNotebook(session, notebookId=ntsc_id).notebook
        return SharedNTSC(id_=ntsc_id, typ=typ, state=ntsc.state)
    elif typ == "tensorboard":
        ntsc = bindings.get_GetTensorboard(session, tensorboardId=ntsc_id).tensorboard
        return SharedNTSC(id_=ntsc_id, typ=typ, state=ntsc.state)
    elif typ == "shell":
        ntsc = bindings.get_GetShell(session, shellId=ntsc_id).shell
        return SharedNTSC(id_=ntsc_id, typ=typ, state=ntsc.state)
    elif typ == "command":
        ntsc = bindings.get_GetCommand(session, commandId=ntsc_id).command
        return SharedNTSC(id_=ntsc_id, typ=typ, state=ntsc.state)
    else:
        raise ValueError("unknown type")


def list_ntsc(session: Session, typ: str, workspace_id: Optional[int] = None) -> List[SharedNTSC]:
    assert typ in all_ntsc
    if typ == "notebook":
        return [
            SharedNTSC(id_=ntsc.id, typ=typ, state=ntsc.state)
            for ntsc in bindings.get_GetNotebooks(session, workspaceId=workspace_id).notebooks
        ]
    elif typ == "tensorboard":
        return [
            SharedNTSC(id_=ntsc.id, typ=typ, state=ntsc.state)
            for ntsc in bindings.get_GetTensorboards(session, workspaceId=workspace_id).tensorboards
        ]
    elif typ == "shell":
        return [
            SharedNTSC(id_=ntsc.id, typ=typ, state=ntsc.state)
            for ntsc in bindings.get_GetShells(session, workspaceId=workspace_id).shells
        ]
    elif typ == "command":
        return [
            SharedNTSC(id_=ntsc.id, typ=typ, state=ntsc.state)
            for ntsc in bindings.get_GetCommands(session, workspaceId=workspace_id).commands
        ]
    else:
        raise ValueError("unknown type")


def wait_for_ntsc_state(
    session: Session,
    typ: str,
    ntsc_id: str,
    predicate: Callable[[bindings.taskv1State], bool],
    timeout: int = 10,
) -> Optional[bindings.taskv1State]:
    """wait for ntsc to reach a state that satisfies the predicate"""
    assert typ in all_ntsc
    start = time.time()
    last_state = None
    while True:
        if time.time() - start > timeout:
            raise Exception(f"timed out waiting for state predicate to pass. reached {last_state}")
        last_state = get_ntsc_details(session, typ, ntsc_id).state
        if predicate(last_state):
            return last_state
        time.sleep(0.5)
