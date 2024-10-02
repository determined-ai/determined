import functools
import uuid
from typing import Any, Callable, Dict, Optional, Sequence, Tuple, TypeVar

import pytest

from determined.common import api
from determined.common.api import authentication, bindings, certs, errors
from determined.experimental import client
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
    return authentication.login(master_url, username, password, cert())


@functools.lru_cache(maxsize=1)
def user_session() -> api.Session:
    return make_session("determined", conf.USER_PASSWORD)


@functools.lru_cache(maxsize=1)
def admin_session() -> api.Session:
    return make_session("admin", conf.USER_PASSWORD)


def get_random_string() -> str:
    return str(uuid.uuid4())


def create_test_user(
    user: Optional[bindings.v1User] = None,
) -> Tuple[api.Session, str]:
    """
    Returns a tuple of (Session, password).
    """
    session = admin_session()
    user = user or bindings.v1User(username=get_random_string(), admin=False, active=True)
    # password must contain both upper-case and lower-case letters
    password = "AB832ABC-fc98-4c73-a0bd-c4708e6479d9"
    bindings.post_PostUser(session, body=bindings.v1PostUserRequest(user=user, password=password))
    sess = make_session(user.username, password)
    return sess, password


def create_linked_user(uid: int, agent_user: str, gid: int, group: str) -> api.Session:
    sess, _ = create_test_user()
    det_obj = client.Determined._from_session(admin_session())
    user = det_obj.get_user_by_name(user_name=sess.username)
    user.link_with_agent(agent_gid=gid, agent_uid=uid, agent_group=group, agent_user=agent_user)
    return sess


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


@functools.lru_cache(maxsize=1)
def _get_master_config() -> Optional[Dict[str, Any]]:
    try:
        sess = admin_session()
        return bindings.get_GetMasterConfig(sess).config
    except (errors.APIException, errors.MasterNotFoundException):
        return None


@functools.lru_cache(maxsize=1)
def _get_is_k8s() -> Optional[bool]:
    try:
        admin_sess = admin_session()
        resp = bindings.get_GetMasterConfig(admin_sess)
        is_k8s = resp.config["resource_manager"]["type"] == "kubernetes"
        assert isinstance(is_k8s, bool)
        return is_k8s
    except (errors.APIException, errors.MasterNotFoundException):
        return None


def skipif_not_k8s(reason: str = "test is k8s-specific") -> Callable[[F], F]:
    def decorator(f: F) -> F:
        is_k8s = _get_is_k8s()
        if is_k8s is None:
            return f
        if not is_k8s:
            return pytest.mark.skipif(True, reason=reason)(f)  # type: ignore
        return f

    return decorator


# Queries the determined master for resource pool information to determine if agent is used
# Currently we are assuming that all resource pools are of the same scheduler type
# which is why only the first resource pool's type is checked.
@functools.lru_cache(maxsize=1)
def _get_scheduler_type() -> Optional[bindings.v1SchedulerType]:
    try:
        sess = user_session()
        resourcePool = bindings.get_GetResourcePools(sess).resourcePools
        if not resourcePool:
            raise ValueError("Resource Pool returned no value. Make sure the resource pool is set.")
        return resourcePool[0].schedulerType
    except (errors.APIException, errors.MasterNotFoundException):
        return None


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


def is_hpc(sess: api.Session) -> bool:
    resourcePool = bindings.get_GetResourcePools(sess).resourcePools
    if not resourcePool:
        raise ValueError("Resource Pool returned no value. Make sure the resource pool is set.")
    st = resourcePool[0].schedulerType
    return st in (bindings.v1SchedulerType.SLURM, bindings.v1SchedulerType.PBS)


@functools.lru_cache(maxsize=1)
def _get_ee() -> Optional[bool]:
    sess = api.UnauthSession(conf.make_master_url(), cert(), max_retries=0)
    try:
        info = sess.get("info").json()
        return "sso_providers" in info
    except (errors.APIException, errors.MasterNotFoundException):
        return None


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


@functools.lru_cache(maxsize=1)
def _get_scim_enabled() -> Optional[bool]:
    sess = api.UnauthSession(conf.make_master_url(), cert(), max_retries=0)
    try:
        info = sess.get("info").json()
        return bool(info.get("sso_providers") and len(info["sso_providers"]) > 0)
    except (errors.APIException, errors.MasterNotFoundException):
        return None


def skipif_scim_not_enabled(reason: str = "scim is required for this test") -> Callable[[F], F]:
    def decorator(f: F) -> F:
        se = _get_scim_enabled()
        if se is None:
            return f
        if not se:
            return pytest.mark.skipif(True, reason=reason)(f)  # type: ignore
        return f

    return decorator


@functools.lru_cache(maxsize=1)
def _get_rbac_enabled() -> Optional[bool]:
    sess = api.UnauthSession(conf.make_master_url(), cert(), max_retries=0)
    try:
        return bindings.get_GetMaster(sess).rbacEnabled
    except (errors.APIException, errors.MasterNotFoundException):
        return None


def skipif_rbac_not_enabled(reason: str = "ee is required for this test") -> Callable[[F], F]:
    def decorator(f: F) -> F:
        re = _get_rbac_enabled()
        if re is None:
            return f
        if not re:
            return pytest.mark.skipif(True, reason=reason)(f)  # type: ignore
        return f

    return decorator


def skipif_unexpected_master_config(
    expected: Callable[[dict], bool], reason: str = "unexpected master config"
) -> Callable[[F], F]:
    def decorator(f: F) -> F:
        mc = _get_master_config()
        if mc is None:
            return f
        if not expected(mc):
            return pytest.mark.skipif(True, reason=reason)(f)  # type: ignore
        return f

    return decorator


def skipif_missing_startup_hook(
    reason: str = "tcd startup hook is required for this test",
) -> Callable[[F], F]:
    """skip if the backing cluster can be missing TCD startup hooks for some of the workloads"""

    def check_startup_hook(mc: dict) -> bool:
        assert isinstance(mc, dict)

        def has_hook(conf_with_tcd: dict) -> bool:
            hook = (conf_with_tcd.get("task_container_defaults") or {}).get("startup_hook")
            return isinstance(hook, str) and hook != ""

        if has_hook(mc):
            return True
        pools = mc.get("resource_pools") or []
        assert isinstance(pools, list)
        return len(pools) > 0 and all(has_hook(pool) for pool in pools)

    return skipif_unexpected_master_config(check_startup_hook, reason=reason)


@functools.lru_cache(maxsize=1)
def _get_strict_q() -> Optional[bool]:
    sess = api.UnauthSession(conf.make_master_url(), cert(), max_retries=0)
    try:
        resp = bindings.get_GetMaster(sess)
        return resp.rbacEnabled and resp.strictJobQueueControl
    except (errors.APIException, errors.MasterNotFoundException):
        return None


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


@functools.lru_cache(maxsize=1)
def _get_streaming_updates_enabled() -> Optional[bool]:
    config = _get_master_config()
    if config is not None:
        streaming_updates_enabled = "streaming_updates" in config["feature_switches"]
        return streaming_updates_enabled
    return None


def skipif_streaming_updates_not_enabled(
    reason: str = "streaming updates is required for this test",
) -> Callable[[F], F]:
    def decorator(f: F) -> F:
        enabled = _get_streaming_updates_enabled()
        # enabled is None when there is an APIException or a MasterNotFoundException.
        # So we want to run the test to an error, but if we skip it, no error will show up.
        if enabled is None:
            return f
        if not enabled:
            return pytest.mark.skipif(True, reason=reason)(f)  # type: ignore
        return f

    return decorator
