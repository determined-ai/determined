import uuid
from typing import Callable, Optional, Sequence, Tuple, TypeVar

import pytest

from determined.common import api
from determined.common.api import authentication, bindings, certs, errors
from tests import config as conf

_cert: Optional[certs.Cert] = None


def cert() -> certs.Cert:
    global _cert
    if _cert is None:
        _cert = certs.default_load(conf.make_master_url())
    return _cert


def make_session(username: str, password: str) -> api.Session:
    master_url = conf.make_master_url()
    # Use login instead of login_with_cache() to not touch auth.json on the filesystem.
    utp = authentication.login(master_url, username, password, cert())
    return api.Session(master_url, utp, cert())


_user_session: Optional[api.Session] = None


def user_session() -> api.Session:
    global _user_session
    if _user_session is None:
        _user_session = make_session("determined", "")
    return _user_session


_admin_session: Optional[api.Session] = None


def admin_session() -> api.Session:
    global _admin_session
    if _admin_session is None:
        _admin_session = make_session("admin", "")
    return _admin_session


def get_random_string() -> str:
    return str(uuid.uuid4())


def create_test_user(
    user: Optional[bindings.v1User] = None,
) -> Tuple[api.Session, str]:
    """
    Returns a tuple of (Session, password).
    """
    session = admin_session()
    username = get_random_string()
    user = user or bindings.v1User(username=username, admin=False, active=True)
    password = get_random_string()
    bindings.post_PostUser(session, body=bindings.v1PostUserRequest(user=user, password=password))
    sess = make_session(username, password)
    return sess, password


def assign_user_role(session: api.Session, user: str, role: str, workspace: Optional[str]) -> None:
    user_assign = api.create_user_assignment_request(
        session, user=user, role=role, workspace=workspace
    )
    req = bindings.v1AssignRolesRequest(userRoleAssignments=user_assign, groupRoleAssignments=[])
    bindings.post_AssignRoles(session, body=req)


def assign_group_role(
    session: api.Session, group: str, role: str, workspace: Optional[str]
) -> None:
    group_assign = api.create_group_assignment_request(
        session, group=group, role=role, workspace=workspace
    )
    req = bindings.v1AssignRolesRequest(userRoleAssignments=[], groupRoleAssignments=group_assign)
    bindings.post_AssignRoles(session, body=req)


def launch_ntsc(
    session: api.Session,
    workspace_id: int,
    typ: api.NTSC_Kind,
    exp_id: Optional[int] = None,
    template: Optional[str] = None,
) -> api.AnyNTSC:
    if typ == api.NTSC_Kind.notebook:
        return bindings.post_LaunchNotebook(
            session,
            body=bindings.v1LaunchNotebookRequest(workspaceId=workspace_id, templateName=template),
        ).notebook
    elif typ == api.NTSC_Kind.tensorboard:
        experiment_ids = [exp_id] if exp_id else []
        return bindings.post_LaunchTensorboard(
            session,
            body=bindings.v1LaunchTensorboardRequest(
                workspaceId=workspace_id, experimentIds=experiment_ids, templateName=template
            ),
        ).tensorboard
    elif typ == api.NTSC_Kind.shell:
        return bindings.post_LaunchShell(
            session,
            body=bindings.v1LaunchShellRequest(workspaceId=workspace_id, templateName=template),
        ).shell
    elif typ == api.NTSC_Kind.command:
        return bindings.post_LaunchCommand(
            session,
            body=bindings.v1LaunchCommandRequest(
                workspaceId=workspace_id,
                config={
                    "entrypoint": ["sleep", "100"],
                },
                templateName=template,
            ),
        ).command
    else:
        raise ValueError("unknown type")


def kill_ntsc(session: api.Session, typ: api.NTSC_Kind, ntsc_id: str) -> None:
    if typ == api.NTSC_Kind.notebook:
        bindings.post_KillNotebook(session, notebookId=ntsc_id)
    elif typ == api.NTSC_Kind.tensorboard:
        bindings.post_KillTensorboard(session, tensorboardId=ntsc_id)
    elif typ == api.NTSC_Kind.shell:
        bindings.post_KillShell(session, shellId=ntsc_id)
    elif typ == api.NTSC_Kind.command:
        bindings.post_KillCommand(session, commandId=ntsc_id)
    else:
        raise ValueError("unknown type")


def set_prio_ntsc(session: api.Session, typ: api.NTSC_Kind, ntsc_id: str, prio: int) -> None:
    if typ == api.NTSC_Kind.notebook:
        bindings.post_SetNotebookPriority(
            session, notebookId=ntsc_id, body=bindings.v1SetNotebookPriorityRequest(priority=prio)
        )
    elif typ == api.NTSC_Kind.tensorboard:
        bindings.post_SetTensorboardPriority(
            session,
            tensorboardId=ntsc_id,
            body=bindings.v1SetTensorboardPriorityRequest(priority=prio),
        )
    elif typ == api.NTSC_Kind.shell:
        bindings.post_SetShellPriority(
            session, shellId=ntsc_id, body=bindings.v1SetShellPriorityRequest(priority=prio)
        )
    elif typ == api.NTSC_Kind.command:
        bindings.post_SetCommandPriority(
            session, commandId=ntsc_id, body=bindings.v1SetCommandPriorityRequest(priority=prio)
        )
    else:
        raise ValueError("unknown type")


def list_ntsc(
    session: api.Session, typ: api.NTSC_Kind, workspace_id: Optional[int] = None
) -> Sequence[api.AnyNTSC]:
    if typ == api.NTSC_Kind.notebook:
        return bindings.get_GetNotebooks(session, workspaceId=workspace_id).notebooks
    elif typ == api.NTSC_Kind.tensorboard:
        return bindings.get_GetTensorboards(session, workspaceId=workspace_id).tensorboards
    elif typ == api.NTSC_Kind.shell:
        return bindings.get_GetShells(session, workspaceId=workspace_id).shells
    elif typ == api.NTSC_Kind.command:
        return bindings.get_GetCommands(session, workspaceId=workspace_id).commands
    else:
        raise ValueError("unknown type")


F = TypeVar("F", bound=Callable)


_is_k8s: Optional[bool] = None


def _get_is_k8s() -> Optional[bool]:
    global _is_k8s

    if _is_k8s is None:
        try:
            admin = admin_session()
            resp = bindings.get_GetMasterConfig(admin)
            _is_k8s = resp.config["resource_manager"]["type"] == "kubernetes"
        except (errors.APIException, errors.MasterNotFoundException):
            pass

    return _is_k8s


def skipif_not_k8s(reason: str = "test is k8s-specific") -> Callable[[F], F]:
    def decorator(f: F) -> F:
        is_k8s = _get_is_k8s()
        if is_k8s is None:
            return f
        if not is_k8s:
            return pytest.mark.skipif(True, reason=reason)(f)  # type: ignore
        return f

    return decorator


_scheduler_type: Optional[bindings.v1SchedulerType] = None


# Queries the determined master for resource pool information to determine if agent is used
# Currently we are assuming that all resource pools are of the same scheduler type
# which is why only the first resource pool's type is checked.
def _get_scheduler_type() -> Optional[bindings.v1SchedulerType]:
    global _scheduler_type
    if _scheduler_type is None:
        try:
            sess = user_session()
            resourcePool = bindings.get_GetResourcePools(sess).resourcePools
            if not resourcePool:
                raise ValueError(
                    "Resource Pool returned no value. Make sure the resource pool is set."
                )
            _scheduler_type = resourcePool[0].schedulerType
        except (errors.APIException, errors.MasterNotFoundException):
            pass
    return _scheduler_type


def skipif_not_hpc(reason: str = "test is hpc-specific") -> Callable[[F], F]:
    def decorator(f: F) -> F:
        st = _get_scheduler_type()
        if st is None:
            return f
        if st not in (bindings.v1SchedulerType.SLURM, bindings.v1SchedulerType.PBS):
            return pytest.mark.skipif(True, reason=reason)(f)  # type: ignore
        return f

    return decorator


def skipif_not_slurm(reason: str = "test is slurm-specific") -> Callable[[F], F]:
    def decorator(f: F) -> F:
        st = _get_scheduler_type()
        if st is None:
            return f
        if st != bindings.v1SchedulerType.SLURM:
            return pytest.mark.skipif(True, reason=reason)(f)  # type: ignore
        return f

    return decorator


def skipif_not_pbs(reason: str = "test is slurm-specific") -> Callable[[F], F]:
    def decorator(f: F) -> F:
        st = _get_scheduler_type()
        if st is None:
            return f
        if st != bindings.v1SchedulerType.PBS:
            return pytest.mark.skipif(True, reason=reason)(f)  # type: ignore
        return f

    return decorator


def is_hpc() -> bool:
    st = _get_scheduler_type()
    if st is None:
        raise RuntimeError("unable to contact master to determine is_hpc()")
    return st in (bindings.v1SchedulerType.SLURM, bindings.v1SchedulerType.PBS)


_is_ee: Optional[bool] = None


def _get_ee() -> Optional[bool]:
    global _is_ee

    if _is_ee is None:
        try:
            sess = api.UnauthSession(conf.make_master_url(), cert())
            info = sess.get("info").json()
            _is_ee = "sso_providers" in info
        except (errors.APIException, errors.MasterNotFoundException):
            pass

    return _is_ee


def skipif_ee(reason: str = "test is oss-specific") -> Callable[[F], F]:
    def decorator(f: F) -> F:
        ee = _get_ee()
        if ee is None:
            return f
        if ee:
            return pytest.mark.skipif(True, reason=reason)(f)  # type: ignore
        return f

    return decorator


def skipif_not_ee(reason: str = "test is ee-specific") -> Callable[[F], F]:
    def decorator(f: F) -> F:
        ee = _get_ee()
        if ee is None:
            return f
        if not ee:
            return pytest.mark.skipif(True, reason=reason)(f)  # type: ignore
        return f

    return decorator


_scim_enabled: Optional[bool] = None


def _get_scim_enabled() -> Optional[bool]:
    global _scim_enabled

    if _scim_enabled is None:
        try:
            sess = api.UnauthSession(conf.make_master_url(), cert())
            info = sess.get("info").json()
            _scim_enabled = bool(info.get("sso_providers") and len(info["sso_providers"]) > 0)
        except (errors.APIException, errors.MasterNotFoundException):
            pass

    return _scim_enabled


def skipif_scim_not_enabled(reason: str = "scim is required for this test") -> Callable[[F], F]:
    def decorator(f: F) -> F:
        se = _get_scim_enabled()
        if se is None:
            return f
        if not se:
            return pytest.mark.skipif(True, reason=reason)(f)  # type: ignore
        return f

    return decorator


_rbac_enabled: Optional[bool] = None


def _get_rbac_enabled() -> Optional[bool]:
    global _rbac_enabled

    if _rbac_enabled is None:
        try:
            sess = api.UnauthSession(conf.make_master_url(), cert())
            _rbac_enabled = bindings.get_GetMaster(sess).rbacEnabled
        except (errors.APIException, errors.MasterNotFoundException):
            pass

    return _rbac_enabled


def skipif_rbac_not_enabled(reason: str = "ee is required for this test") -> Callable[[F], F]:
    def decorator(f: F) -> F:
        re = _get_rbac_enabled()
        if re is None:
            return f
        if not re:
            return pytest.mark.skipif(True, reason=reason)(f)  # type: ignore
        return f

    return decorator


_strict_q: Optional[bool] = None


def _get_strict_q() -> Optional[bool]:
    global _strict_q

    if _strict_q is None:
        try:
            sess = api.UnauthSession(conf.make_master_url(), cert())
            resp = bindings.get_GetMaster(sess)
            _strict_q = resp.rbacEnabled and resp.strictJobQueueControl
        except (errors.APIException, errors.MasterNotFoundException):
            pass

    return _strict_q


def skipif_strict_q_control_not_enabled(
    reason: str = "rbac and strict queue control are required for this test",
) -> Callable[[F], F]:
    def decorator(f: F) -> F:
        sq = _get_strict_q()
        if sq is None:
            return f
        if not sq:
            return pytest.mark.skipif(True, reason=reason)(f)  # type: ignore
        return f

    return decorator
