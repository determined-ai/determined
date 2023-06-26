import uuid
from typing import Optional, Sequence

from determined.common import api
from determined.common.api import Session, authentication, bindings, certs
from determined.common.api._util import AnyNTSC, NTSC_Kind
from tests import config as conf

ADMIN_CREDENTIALS = authentication.Credentials("admin", "")


def get_random_string() -> str:
    return str(uuid.uuid4())


def determined_test_session(
    credentials: Optional[authentication.Credentials] = None,
    admin: Optional[bool] = None,
) -> api.Session:
    assert admin is None or credentials is None, "admin and credentials are mutually exclusive"

    if credentials is None:
        if admin:
            credentials = ADMIN_CREDENTIALS
        else:
            credentials = authentication.Credentials("determined", "")

    murl = conf.make_master_url()
    certs.cli_cert = certs.default_load(murl)
    authentication.cli_auth = authentication.Authentication(
        murl, requested_user=credentials.username, password=credentials.password
    )
    return api.Session(murl, credentials.username, authentication.cli_auth, certs.cli_cert)


def create_test_user(
    add_password: bool = False,
    session: Optional[api.Session] = None,
    user: Optional[bindings.v1User] = None,
) -> authentication.Credentials:
    session = session or determined_test_session(admin=True)
    user = user or bindings.v1User(username=get_random_string(), admin=False, active=True)
    password = get_random_string() if add_password else ""
    bindings.post_PostUser(session, body=bindings.v1PostUserRequest(user=user, password=password))
    return authentication.Credentials(user.username, password)


def configure_token_store(credentials: authentication.Credentials) -> None:
    """Authenticate the user for CLI usage with the given credentials."""
    token_store = authentication.TokenStore(conf.make_master_url())
    certs.cli_cert = certs.default_load(conf.make_master_url())
    token = authentication.do_login(
        conf.make_master_url(), credentials.username, credentials.password, certs.cli_cert
    )
    token_store.set_token(credentials.username, token)
    token_store.set_active(credentials.username)


def launch_ntsc(
    session: Session, workspace_id: int, typ: NTSC_Kind, exp_id: Optional[int] = None
) -> str:
    if typ == NTSC_Kind.notebook:
        return bindings.post_LaunchNotebook(
            session, body=bindings.v1LaunchNotebookRequest(workspaceId=workspace_id)
        ).notebook.id
    elif typ == NTSC_Kind.tensorboard:
        experiment_ids = [exp_id] if exp_id else []
        return bindings.post_LaunchTensorboard(
            session,
            body=bindings.v1LaunchTensorboardRequest(
                workspaceId=workspace_id, experimentIds=experiment_ids
            ),
        ).tensorboard.id
    elif typ == NTSC_Kind.shell:
        return bindings.post_LaunchShell(
            session, body=bindings.v1LaunchShellRequest(workspaceId=workspace_id)
        ).shell.id
    elif typ == NTSC_Kind.command:
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


def kill_ntsc(session: Session, typ: NTSC_Kind, ntsc_id: str) -> None:
    if typ == NTSC_Kind.notebook:
        bindings.post_KillNotebook(session, notebookId=ntsc_id)
    elif typ == NTSC_Kind.tensorboard:
        bindings.post_KillTensorboard(session, tensorboardId=ntsc_id)
    elif typ == NTSC_Kind.shell:
        bindings.post_KillShell(session, shellId=ntsc_id)
    elif typ == NTSC_Kind.command:
        bindings.post_KillCommand(session, commandId=ntsc_id)
    else:
        raise ValueError("unknown type")


def set_prio_ntsc(session: Session, typ: NTSC_Kind, ntsc_id: str, prio: int) -> None:
    if typ == NTSC_Kind.notebook:
        bindings.post_SetNotebookPriority(
            session, notebookId=ntsc_id, body=bindings.v1SetNotebookPriorityRequest(priority=prio)
        )
    elif typ == NTSC_Kind.tensorboard:
        bindings.post_SetTensorboardPriority(
            session,
            tensorboardId=ntsc_id,
            body=bindings.v1SetTensorboardPriorityRequest(priority=prio),
        )
    elif typ == NTSC_Kind.shell:
        bindings.post_SetShellPriority(
            session, shellId=ntsc_id, body=bindings.v1SetShellPriorityRequest(priority=prio)
        )
    elif typ == NTSC_Kind.command:
        bindings.post_SetCommandPriority(
            session, commandId=ntsc_id, body=bindings.v1SetCommandPriorityRequest(priority=prio)
        )
    else:
        raise ValueError("unknown type")


def list_ntsc(
    session: Session, typ: NTSC_Kind, workspace_id: Optional[int] = None
) -> Sequence[AnyNTSC]:
    if typ == NTSC_Kind.notebook:
        return bindings.get_GetNotebooks(session, workspaceId=workspace_id).notebooks
    elif typ == NTSC_Kind.tensorboard:
        return bindings.get_GetTensorboards(session, workspaceId=workspace_id).tensorboards
    elif typ == NTSC_Kind.shell:
        return bindings.get_GetShells(session, workspaceId=workspace_id).shells
    elif typ == NTSC_Kind.command:
        return bindings.get_GetCommands(session, workspaceId=workspace_id).commands
    else:
        raise ValueError("unknown type")
