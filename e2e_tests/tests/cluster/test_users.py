import contextlib
import logging
import os
import pathlib
import shutil
import subprocess
import time
from typing import Generator, List, Tuple

import appdirs
import pytest

from determined.common import api
from determined.common.api import bindings, errors
from determined.experimental import client
from tests import api_utils, command
from tests import config as conf
from tests import detproc
from tests.experiment import noop

EXPECT_TIMEOUT = 5
logger = logging.getLogger(__name__)


def activate_deactivate_user(sess: api.Session, active: bool, target_user: str) -> None:
    command = [
        "det",
        "user",
        "activate" if active else "deactivate",
        target_user,
    ]
    detproc.check_output(sess, command)


def extract_columns(output: str, column_indices: List[int]) -> List[Tuple[str, ...]]:
    lines = output.split("\n")
    # Ignore the header.
    lines = lines[2:]
    parsed = []
    for line in lines:
        if not line:
            continue
        columns = line.split("|")
        parsed.append(tuple(columns[i].strip() for i in column_indices))

    return parsed


def extract_id_and_owner_from_exp_list(output: str) -> List[Tuple[int, str]]:
    rows = extract_columns(output, [0, 1])
    return [(int(r[0]), r[1]) for r in rows]


@pytest.mark.e2e_cpu
def test_post_user_api() -> None:
    new_username = "57884ad2-400e-4e6a-a279-e5b943c32c4f"
    # password must contain both upper-case and lower-case letters
    new_password = "262DE646-4549-4504-a3c6-80c04e413329"

    sess = api_utils.admin_session()

    user = bindings.v1User(active=True, admin=False, username=new_username)
    body = bindings.v1PostUserRequest(password=new_password, user=user)
    resp = bindings.post_PostUser(sess, body=body)
    assert resp and resp.user
    assert resp.to_json()["user"]["username"] == new_username
    assert resp.user.agentUserGroup is None

    user = bindings.v1User(
        active=True,
        admin=False,
        username=api_utils.get_random_string(),
        agentUserGroup=bindings.v1AgentUserGroup(
            agentUid=1000, agentGid=1001, agentUser="username", agentGroup="groupname"
        ),
    )
    resp = bindings.post_PostUser(
        sess, body=bindings.v1PostUserRequest(user=user, password=new_password)
    )
    assert resp and resp.user and resp.user.agentUserGroup
    assert resp.user.agentUserGroup.agentUser == "username"
    assert resp.user.agentUserGroup.agentGroup == "groupname"
    assert resp.user.agentUserGroup.agentUid == 1000
    assert resp.user.agentUserGroup.agentGid == 1001

    user = bindings.v1User(
        active=True,
        admin=False,
        username=api_utils.get_random_string(),
        agentUserGroup=bindings.v1AgentUserGroup(
            agentUid=1000,
            agentGid=1001,
        ),
    )

    with pytest.raises(errors.APIException):
        bindings.post_PostUser(
            sess, body=bindings.v1PostUserRequest(user=user, password=new_password)
        )


@pytest.mark.e2e_cpu
def test_create_user_sdk() -> None:
    username = "d62cc542-bf37-44df-9343-407f727387f5"
    # password must contain both upper-case and lower-case letters
    password = "7C663CBB-46b4-4d09-9d01-d32b5cef7141"
    det_obj = client.Determined._from_session(api_utils.admin_session())
    user = det_obj.create_user(username=username, admin=False, password=password)
    assert user.user_id is not None and user.username == username


@pytest.mark.e2e_cpu
def test_logout() -> None:
    # Make sure that a logged out session cannot be reused.
    sess = api_utils.make_session("determined", conf.USER_PASSWORD)

    bindings.post_Logout(sess)
    with pytest.raises(errors.UnauthenticatedException):
        bindings.get_GetMe(sess)


@pytest.mark.e2e_cpu
@pytest.mark.e2e_cpu_postgres
def test_activate_deactivate() -> None:
    sess, password = api_utils.create_test_user()

    # Deactivate user.
    admin = api_utils.admin_session()
    activate_deactivate_user(admin, False, sess.username)

    # Attempt to log in again.
    with pytest.raises(errors.ForbiddenException):
        api_utils.make_session(sess.username, password)

    # Activate user.
    activate_deactivate_user(admin, True, sess.username)

    # Now log in again.
    api_utils.make_session(sess.username, password)

    # SDK testing for activating and deactivating.
    det_obj = client.Determined._from_session(admin)
    user = det_obj.get_user_by_name(user_name=sess.username)
    user.deactivate()
    assert user.active is not True
    with pytest.raises(errors.ForbiddenException):
        api_utils.make_session(sess.username, password)

    user.activate()
    assert user.active is True
    api_utils.make_session(sess.username, password)


@pytest.mark.e2e_cpu
@pytest.mark.e2e_cpu_postgres
def test_change_password() -> None:
    sess, old_password = api_utils.create_test_user()
    d = client.Determined._from_session(api_utils.admin_session())
    userobj = d.get_user_by_name(sess.username)
    userobj.change_password("newPass123!")

    # Old password does not work anymore.
    with pytest.raises(errors.UnauthenticatedException):
        api_utils.make_session(sess.username, old_password)

    # New password does work.
    api_utils.make_session(sess.username, "newPass123!")


@pytest.mark.e2e_cpu
def test_change_own_password() -> None:
    # Create a user without a password.
    sess, old_password = api_utils.create_test_user()

    d = client.Determined._from_session(sess)
    userobj = d.get_user_by_name(sess.username)
    userobj.change_password("newPass123!")

    with pytest.raises(errors.UnauthenticatedException):
        api_utils.make_session(sess.username, old_password)

    api_utils.make_session(sess.username, "newPass123!")


@pytest.mark.e2e_cpu
def test_change_username() -> None:
    admin = api_utils.admin_session()
    sess, _ = api_utils.create_test_user()
    old_username = sess.username
    new_username = "rename-user-64"
    command = ["det", "user", "rename", old_username, new_username]
    detproc.check_call(admin, command)
    d = client.Determined._from_session(admin)
    user = d.get_user_by_name(user_name=new_username)
    assert user.username == new_username

    # Test SDK
    new_username = "rename-user-$64"
    user.rename(new_username)
    user = d.get_user_by_name(user_name=new_username)
    assert user.username == new_username


@pytest.mark.e2e_cpu
@pytest.mark.e2e_cpu_postgres
@pytest.mark.e2e_cpu_cross_version
def test_experiment_creation_and_listing() -> None:
    # Create 2 users.
    sess1, _ = api_utils.create_test_user()
    sess2, _ = api_utils.create_test_user()

    # Create an experiment as each user.
    exp1 = noop.create_experiment(sess1)
    exp2 = noop.create_experiment(sess2)
    # We don't care what happens to them.
    exp1.kill()
    exp2.kill()

    # user 1 can only see user 1 experiment
    output = extract_id_and_owner_from_exp_list(detproc.check_output(sess1, ["det", "e", "list"]))
    assert (exp1.id, sess1.username) in output, output
    assert (exp2.id, sess2.username) not in output, output

    # Now use the -a flag to list all experiments.  The output should include both experiments.
    output = extract_id_and_owner_from_exp_list(
        detproc.check_output(sess1, ["det", "e", "list", "-a"])
    )
    assert (exp1.id, sess1.username) in output, output
    assert (exp2.id, sess2.username) in output, output

    # Clean up.
    delete_experiments(api_utils.admin_session(), exp1.id, exp2.id)


@pytest.mark.e2e_cpu
def test_login_wrong_password() -> None:
    sess, password = api_utils.create_test_user()
    with pytest.raises(errors.UnauthenticatedException):
        api_utils.make_session(sess.username, "wrong" + password)


@pytest.mark.e2e_cpu
def test_login_as_non_existent_user() -> None:
    with pytest.raises(errors.UnauthenticatedException):
        api_utils.make_session("nOtArEaLuSeR", "password")


@pytest.mark.e2e_cpu
def test_login_as_non_active_user() -> None:
    sess, password = api_utils.create_test_user()
    admin = api_utils.admin_session()
    d = client.Determined._from_session(admin)
    userobj = d.get_user_by_name(sess.username)
    userobj.deactivate()

    with pytest.raises(errors.ForbiddenException, match="user is not active"):
        api_utils.make_session(sess.username, password)


@pytest.mark.e2e_cpu
def test_non_admin_user_link_with_agent_user() -> None:
    sess1 = api_utils.user_session()
    sess2, _ = api_utils.create_test_user()

    cmd = [
        "det",
        "user",
        "link-with-agent-user",
        sess2.username,
        "--agent-uid",
        "1",
        "--agent-gid",
        "1",
        "--agent-user",
        sess2.username,
        "--agent-group",
        sess2.username,
    ]

    detproc.check_error(sess1, cmd, "forbidden")


@pytest.mark.e2e_cpu
def test_non_admin_commands() -> None:
    sess = api_utils.user_session()
    command = [
        "det",
        "slot",
        "list",
        "--json",
    ]
    slots = detproc.check_json(sess, command)

    slot_id = slots[0]["slot_id"]
    agent_id = slots[0]["agent_id"]

    enable_slots = ["slot", "enable", agent_id, slot_id]
    disable_slots = ["slot", "disable", agent_id, slot_id]
    enable_agents = ["agent", "enable", agent_id]
    disable_agents = ["agent", "disable", agent_id]
    config = ["master", "config"]
    for cmd in [disable_slots, disable_agents, enable_slots, enable_agents, config]:
        detproc.check_error(sess, ["det", *cmd], "forbidden")


def run_command(session: api.Session) -> str:
    body = bindings.v1LaunchCommandRequest(config={"entrypoint": ["echo", "hello"]})
    cmd = bindings.post_LaunchCommand(session, body=body).command
    return cmd.id


def start_notebook(sess: api.Session) -> str:
    return detproc.check_output(sess, ["det", "notebook", "start", "-d"]).strip()


def start_tensorboard(sess: api.Session, experiment_id: int) -> str:
    cmd = ["det", "tensorboard", "start", "-d", str(experiment_id)]
    return detproc.check_output(sess, cmd).strip()


def delete_experiments(sess: api.Session, *experiment_ids: int) -> None:
    eids = set(experiment_ids)
    while eids:
        output = extract_columns(detproc.check_output(sess, ["det", "e", "list", "-a"]), [0, 4])

        running_ids = {int(o[0]) for o in output if o[1] in ["COMPLETED", "CANCELED"]}
        intersection = eids & running_ids
        if not intersection:
            time.sleep(0.5)
            continue

        experiment_id = intersection.pop()
        detproc.check_output(sess, ["det", "e", "delete", "--yes", str(experiment_id)])
        eids.remove(experiment_id)


def kill_notebooks(sess: api.Session, *notebook_ids: str) -> None:
    nids = set(notebook_ids)
    while nids:
        output = extract_columns(
            detproc.check_output(sess, ["det", "notebook", "list", "-a"]), [0, 3]
        )  # id, state

        # Get set of running IDs.
        running_ids = {task_id for task_id, state in output if state == "RUNNING"}

        intersection = running_ids & nids
        if not intersection:
            time.sleep(0.5)
            continue

        notebook_id = intersection.pop()
        detproc.check_output(sess, ["det", "notebook", "kill", notebook_id])
        nids.remove(notebook_id)


def kill_tensorboards(sess: api.Session, *tensorboard_ids: str) -> None:
    tids = set(tensorboard_ids)
    while tids:
        output = extract_columns(
            detproc.check_output(sess, ["det", "tensorboard", "list", "-a"]), [0, 3]
        )

        running_ids = {task_id for task_id, state in output if state == "RUNNING"}

        intersection = running_ids & tids
        if not intersection:
            time.sleep(0.5)
            continue

        tensorboard_id = intersection.pop()
        detproc.check_output(sess, ["det", "tensorboard", "kill", tensorboard_id])
        tids.remove(tensorboard_id)


@pytest.mark.e2e_cpu
def test_notebook_creation_and_listing() -> None:
    sess1, _ = api_utils.create_test_user()
    sess2, _ = api_utils.create_test_user()

    notebook_id1 = start_notebook(sess1)

    notebook_id2 = start_notebook(sess2)

    # Listing should only give us user 2's experiment.
    output = extract_columns(detproc.check_output(sess2, ["det", "notebook", "list"]), [0, 1])

    output = extract_columns(detproc.check_output(sess1, ["det", "notebook", "list"]), [0, 1])
    assert (notebook_id1, sess1.username) in output
    assert (notebook_id2, sess2.username) not in output

    # Now test listing all.
    output = extract_columns(detproc.check_output(sess1, ["det", "notebook", "list", "-a"]), [0, 1])
    assert (notebook_id1, sess1.username) in output
    assert (notebook_id2, sess2.username) in output

    # Clean up, killing experiments.
    kill_notebooks(api_utils.admin_session(), notebook_id1, notebook_id2)


@pytest.mark.e2e_cpu
def test_tensorboard_creation_and_listing() -> None:
    sess1, _ = api_utils.create_test_user()
    sess2, _ = api_utils.create_test_user()

    # Create an experiment.
    exp1 = noop.create_experiment(sess1, [noop.Report({"x": 1})])
    tensorboard_id1 = start_tensorboard(sess1, exp1.id)

    exp2 = noop.create_experiment(sess2, [noop.Report({"x": 1})])
    tensorboard_id2 = start_tensorboard(sess2, exp2.id)

    output = extract_columns(detproc.check_output(sess1, ["det", "tensorboard", "list"]), [0, 1])
    assert (tensorboard_id1, sess1.username) in output
    assert (tensorboard_id2, sess2.username) not in output

    output = extract_columns(
        detproc.check_output(sess1, ["det", "tensorboard", "list", "-a"]), [0, 1]
    )
    assert (tensorboard_id1, sess1.username) in output
    assert (tensorboard_id2, sess2.username) in output

    admin = api_utils.admin_session()
    kill_tensorboards(admin, tensorboard_id1, tensorboard_id2)
    exp1.kill()
    exp2.kill()


@pytest.mark.e2e_cpu
def test_command_creation_and_listing() -> None:
    sess1, _ = api_utils.create_test_user()
    sess2, _ = api_utils.create_test_user()

    command_id1 = run_command(session=sess1)
    command_id2 = run_command(session=sess2)

    cmds = bindings.get_GetCommands(sess1, users=[sess1.username]).commands
    output = [(cmd.id, cmd.username) for cmd in cmds]
    assert (command_id1, sess1.username) in output
    assert (command_id2, sess2.username) not in output

    cmds = bindings.get_GetCommands(sess1).commands
    output = [(cmd.id, cmd.username) for cmd in cmds]
    assert (command_id1, sess1.username) in output
    assert (command_id2, sess2.username) in output


def create_linked_user_cli(uid: int, user: str, gid: int, group: str) -> api.Session:
    admin = api_utils.admin_session()
    sess, _ = api_utils.create_test_user()

    cmd = [
        "det",
        "user",
        "link-with-agent-user",
        sess.username,
        "--agent-uid",
        str(uid),
        "--agent-gid",
        str(gid),
        "--agent-user",
        user,
        "--agent-group",
        group,
    ]

    detproc.check_call(admin, cmd)

    return sess


def check_link_with_agent_output(sess: api.Session, expected_output: str) -> None:
    assert expected_output in detproc.check_output(
        sess,
        ["det", "cmd", "run", "bash", "-c", "echo $(id -u -n):$(id -u):$(id -g -n):$(id -g)"],
    )


@pytest.mark.e2e_cpu
def test_link_with_agent_user() -> None:
    sess = create_linked_user_cli(200, "someuser", 300, "somegroup")
    expected_output = "someuser:200:somegroup:300"
    check_link_with_agent_output(sess, expected_output)

    sess = api_utils.create_linked_user(210, "anyuser", 310, "anygroup")
    expected_output = "anyuser:210:anygroup:310"
    check_link_with_agent_output(sess, expected_output)


@pytest.mark.e2e_cpu
def test_link_with_large_uid() -> None:
    sess = create_linked_user_cli(2000000000, "someuser", 2000000000, "somegroup")

    expected_output = "someuser:2000000000:somegroup:2000000000"
    check_link_with_agent_output(sess, expected_output)


@pytest.mark.e2e_cpu
def test_link_with_existing_agent_user() -> None:
    sess = create_linked_user_cli(65533, "det-nobody", 65533, "det-nobody")

    expected_output = "det-nobody:65533:det-nobody:65533"
    check_link_with_agent_output(sess, expected_output)


@contextlib.contextmanager
def non_tmp_shared_fs_path() -> Generator:
    """
    Proper checkpoint storage handling for shared_fs involves properly choosing to use the
    container_path instead of the host_path. Issues don't really arise if the container is running
    as root (because root can write to anywhere) or if host_path is in /tmp (because /tmp is world
    writable) so this context manager yields a checkpoint storage config where host_path is a
    user-owned directory.

    Making it a user-owned directory ensures that the test runs without root privileges on
    normal developer machines, and it also ensures that the test would fail if the code was broken.

    Tests should not pollute user directories though, so make sure to clean up the checkpoint
    directory that we use.
    """

    cache_dir = appdirs.user_cache_dir("determined", "determined")
    checkpoint_dir = os.path.join(cache_dir, "e2e_tests")
    os.makedirs(checkpoint_dir, exist_ok=True)
    os.chmod(checkpoint_dir, 0o777)

    try:
        yield checkpoint_dir
    finally:
        shutil.rmtree(checkpoint_dir)


@pytest.mark.e2e_cpu
def test_non_root_experiment(tmp_path: pathlib.Path) -> None:
    sess = create_linked_user_cli(65533, "det-nobody", 65533, "det-nobody")

    # Use a user-owned path to ensure shared_fs uses the container_path and not host_path.
    with non_tmp_shared_fs_path() as host_path:
        config = {
            "checkpoint_storage": {
                "type": "shared_fs",
                "host_path": host_path,
            },
            # Call the `det` cli to make sure det is on the PATH.
            # Call `det user whoami` to make sure we can authenticate with the master.
            "entrypoint": "det user whoami || exit 77",
            "searcher": {
                "name": "single",
                "metric": "x",
                "max_length": 1,
            },
        }
        exp_ref = noop.create_experiment(sess, config=config)
        assert exp_ref.wait(interval=0.01) == client.ExperimentState.COMPLETED


@pytest.mark.e2e_cpu
def test_link_without_agent_user() -> None:
    sess, _ = api_utils.create_test_user()

    check_link_with_agent_output(sess, "root:0:root:0")


@pytest.mark.e2e_cpu
def test_non_root_shell(tmp_path: pathlib.Path) -> None:
    sess = create_linked_user_cli(1234, "someuser", 1234, "somegroup")
    exp = "someuser:1234:somegroup:1234"
    cmd = "echo; echo $(id -u -n):$(id -u):$(id -g -n):$(id -g)"
    with command.interactive_command(sess, ["shell", "start", "--detach"]) as shell:
        assert shell.task_id
        assert exp in detproc.check_output(
            sess, ["det", "shell", "open", shell.task_id, "--", "bash", "-c", cmd]
        )


@pytest.mark.e2e_cpu
def test_experiment_delete() -> None:
    sess = api_utils.user_session()
    other, _ = api_utils.create_test_user()

    exp_ref = noop.create_experiment(sess)
    # We don't care about finishing it, though.
    exp_ref.kill()

    # "det experiment delete" call should fail, because the other user is not an admin and
    # doesn't own the experiment.
    cmd = ["det", "experiment", "delete", str(exp_ref.id), "--yes"]
    detproc.check_error(other, cmd, "forbidden")

    # but the owner can delete it
    detproc.check_output(sess, cmd)

    experiment_delete_deadline = time.time() + 5 * 60
    while True:
        # "det experiment describe" call should fail, because the
        # experiment is no longer in the database.
        p = detproc.run(
            sess,
            ["det", "experiment", "describe", str(exp_ref.id)],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )
        if p.returncode != 0:
            assert p.stderr and b"not found" in p.stderr, p.stderr
            return
        elif time.time() > experiment_delete_deadline:
            pytest.fail("experiment didn't delete after timeout")


@pytest.mark.e2e_cpu
@pytest.mark.e2e_cpu_postgres
def test_change_displayname() -> None:
    sess, _ = api_utils.create_test_user()
    original_name = sess.username

    det_obj = client.Determined._from_session(api_utils.admin_session())
    current_user = det_obj.get_user_by_name(original_name)
    assert current_user is not None and current_user.user_id

    # Rename user using display name
    patch_user = bindings.v1PatchUser(displayName="renamed display-name")
    bindings.patch_PatchUser(sess, body=patch_user, userId=current_user.user_id)

    modded_user = bindings.get_GetUser(sess, userId=current_user.user_id).user
    assert modded_user is not None
    assert modded_user.displayName == "renamed display-name"

    # Rename user display name using SDK
    user = det_obj.get_user_by_id(user_id=current_user.user_id)
    user.change_display_name(display_name="renamedSDK")

    modded_user_sdk = det_obj.get_user_by_id(user_id=current_user.user_id)
    assert modded_user_sdk is not None
    assert modded_user_sdk.display_name == "renamedSDK"

    # Avoid display name of 'admin'
    patch_user.displayName = "Admin"
    with pytest.raises(errors.APIException):
        bindings.patch_PatchUser(sess, body=patch_user, userId=current_user.user_id)

    # Clear display name (UI will show username)
    patch_user.displayName = ""
    bindings.patch_PatchUser(sess, body=patch_user, userId=current_user.user_id)

    modded_user = bindings.get_GetUser(sess, userId=current_user.user_id).user
    assert modded_user is not None
    assert modded_user.displayName == ""


@pytest.mark.e2e_cpu
def test_patch_agentusergroup() -> None:
    sess, _ = api_utils.create_test_user()

    # Patch - normal.
    admin = api_utils.admin_session()
    det_obj = client.Determined._from_session(admin)
    patch_user = bindings.v1PatchUser(
        agentUserGroup=bindings.v1AgentUserGroup(
            agentGid=1000, agentUid=1000, agentUser="username", agentGroup="groupname"
        )
    )
    test_user = det_obj.get_user_by_name(sess.username)
    assert test_user.user_id
    bindings.patch_PatchUser(admin, body=patch_user, userId=test_user.user_id)
    patched_user = bindings.get_GetUser(admin, userId=test_user.user_id).user
    assert patched_user is not None and patched_user.agentUserGroup is not None
    assert patched_user.agentUserGroup.agentUser == "username"
    assert patched_user.agentUserGroup.agentGroup == "groupname"

    # Patch - missing username/groupname.
    patch_user = bindings.v1PatchUser(
        agentUserGroup=bindings.v1AgentUserGroup(agentGid=1000, agentUid=1000)
    )
    test_user = det_obj.get_user_by_name(sess.username)
    assert test_user.user_id
    with pytest.raises(errors.APIException):
        bindings.patch_PatchUser(admin, body=patch_user, userId=test_user.user_id)


@pytest.mark.e2e_cpu
def test_user_edit() -> None:
    admin = api_utils.admin_session()
    sess, _ = api_utils.create_test_user()
    original_name = sess.username

    det_obj = client.Determined._from_session(admin)
    current_user = det_obj.get_user_by_name(original_name)

    new_display_name = api_utils.get_random_string()
    new_username = api_utils.get_random_string()

    assert current_user is not None and current_user.user_id
    command = [
        "det",
        "user",
        "edit",
        original_name,
        "--display-name",
        new_display_name,
        "--username",
        new_username,
        "--active=true",
        "--remote=false",
        "--admin=true",
    ]
    detproc.check_output(admin, command)

    modded_user = bindings.get_GetUser(admin, userId=current_user.user_id).user
    assert modded_user is not None
    assert modded_user.displayName == new_display_name
    assert modded_user.username == new_username
    assert modded_user.active
    assert not modded_user.remote
    assert modded_user.admin


@pytest.mark.e2e_cpu
def test_user_list() -> None:
    admin = api_utils.admin_session()
    sess, _ = api_utils.create_test_user()
    output = detproc.check_output(admin, ["det", "user", "ls"])
    assert sess.username in output

    # Deactivate user
    activate_deactivate_user(admin, active=False, target_user=sess.username)

    # User should no longer appear in list
    output = detproc.check_output(admin, ["det", "user", "ls"])
    assert sess.username not in output

    # User still appears with --all
    output = detproc.check_output(admin, ["det", "user", "ls", "--all"])
    assert sess.username in output
