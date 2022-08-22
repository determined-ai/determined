import contextlib
import json
import logging
import os
import pathlib
import shutil
import subprocess
import time
import uuid
from typing import Dict, Generator, Iterator, List, Optional, Tuple, cast

import appdirs
import pexpect
import pytest
from pexpect import spawn

from determined.common import api, constants, yaml
from determined.common.api import authentication, bindings, certs, errors
from tests import command
from tests import config as conf
from tests import experiment as exp
from tests.filetree import FileTree

EXPECT_TIMEOUT = 5
ADMIN_CREDENTIALS = authentication.Credentials("admin", "")
logger = logging.getLogger(__name__)


@pytest.fixture(scope="session")
def clean_auth() -> Iterator[None]:
    """
    clean_auth is a session-level fixture that ensures that we run tests with no preconfigured
    authentication, and that any settings we save during tests are cleaned up afterwards.
    """
    authentication.TokenStore(conf.make_master_url()).delete_token_cache()
    yield None
    authentication.TokenStore(conf.make_master_url()).delete_token_cache()


@contextlib.contextmanager
def logged_in_user(credentials: authentication.Credentials) -> Generator:
    assert log_in_user(credentials) == 0
    yield
    assert log_out_user() == 0


def get_random_string() -> str:
    return str(uuid.uuid4())


def det_spawn(args: List[str], env: Optional[Dict[str, str]] = None) -> spawn:
    args = ["-m", conf.make_master_url()] + args
    return pexpect.spawn("det", args, env=env)


def det_run(args: List[str]) -> str:
    return cast(str, pexpect.run(f"det -m {conf.make_master_url()} {' '.join(args)}").decode())


def log_in_user(credentials: authentication.Credentials) -> int:
    username, password = credentials
    child = det_spawn(["user", "login", username])
    child.setecho(True)
    expected = f"Password for user '{username}':"
    child.expect(expected, timeout=EXPECT_TIMEOUT)
    child.sendline(password)
    child.read()
    child.wait()
    return cast(int, child.exitstatus)


def create_user(n_username: str, admin_credentials: authentication.Credentials) -> None:
    a_username, a_password = admin_credentials

    child = det_spawn(["-u", a_username, "user", "create", n_username])

    expected_password_prompt = f"Password for user '{a_username}':"
    i = child.expect([expected_password_prompt, pexpect.EOF], timeout=EXPECT_TIMEOUT)
    if i == 0:
        child.sendline(a_password)
    child.read()
    child.wait()
    child.close()

    assert child.exitstatus == 0
    # Now we activate the user.
    child = det_spawn(["-u", a_username, "user", "activate", n_username])
    child.expect(pexpect.EOF, timeout=EXPECT_TIMEOUT)
    child.read()
    child.wait()
    child.close()
    assert child.exitstatus == 0


def change_user_password(
    target_username: str, target_password: str, admin_credentials: authentication.Credentials
) -> int:
    a_username, a_password = admin_credentials

    child = det_spawn(["-u", a_username, "user", "change-password", target_username])
    expected_pword_prompt = f"Password for user '{a_username}':"
    expected_new_pword_prompt = f"New password for user '{target_username}':"
    confirm_pword_prompt = "Confirm password:"

    i = child.expect([expected_pword_prompt, expected_new_pword_prompt], timeout=EXPECT_TIMEOUT)
    if i == 0:
        child.sendline(a_password)
        child.expect(expected_new_pword_prompt, timeout=EXPECT_TIMEOUT)

    child.sendline(target_password)
    child.expect(confirm_pword_prompt, timeout=EXPECT_TIMEOUT)
    child.sendline(target_password)

    child.read()
    child.wait()
    child.close()
    return cast(int, child.exitstatus)


def create_test_user(
    admin_credentials: authentication.Credentials, add_password: bool = False
) -> authentication.Credentials:
    username = get_random_string()
    create_user(username, admin_credentials)

    password = ""
    if add_password:
        password = get_random_string()
        assert change_user_password(username, password, admin_credentials) == 0

    return authentication.Credentials(username, password)


def log_out_user(username: Optional[str] = None) -> int:
    if username is not None:
        args = ["-u", username, "user", "logout"]
    else:
        args = ["user", "logout"]

    child = det_spawn(args)
    child.read()
    child.wait()
    return cast(int, child.exitstatus)


def activate_deactivate_user(
    active: bool, target_user: str, admin_credentials: Tuple[str, str]
) -> int:
    a_username, a_password = admin_credentials

    child = det_spawn(
        ["-u", a_username, "user", "activate" if active else "deactivate", target_user]
    )
    expected_password_prompt = f"Password for user '{a_username}':"
    i = child.expect([expected_password_prompt, pexpect.EOF], timeout=EXPECT_TIMEOUT)
    if i == 0:
        child.sendline(a_password)

    child.read()
    child.wait()
    child.close()
    return cast(int, child.exitstatus)


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
def test_post_user_api(clean_auth: None) -> None:
    master_url = conf.make_master_url()
    authentication.cli_auth = authentication.Authentication(
        conf.make_master_url(), requested_user="admin", password="", try_reauth=True
    )
    new_username = get_random_string()

    user = bindings.v1User(active=True, admin=False, username=new_username)
    body = bindings.v1PostUserRequest(password="", user=user)
    resp = bindings.post_PostUser(
        api.Session(master_url, "admin", authentication.cli_auth, None), body=body
    )
    assert resp.to_json()["user"]["username"] == new_username


@pytest.mark.e2e_cpu
def test_logout(clean_auth: None) -> None:
    # Tests fallback to default determined user
    creds = create_test_user(ADMIN_CREDENTIALS, True)

    # Set Determined password to something in order to disable auto-login.
    password = get_random_string()
    assert change_user_password(constants.DEFAULT_DETERMINED_USER, password, ADMIN_CREDENTIALS) == 0

    # Log in as new user.
    with logged_in_user(creds):
        # Now we should be able to list experiments.
        child = det_spawn(["e", "list"])
        child.read()
        child.wait()
        child.close()
        assert child.status == 0

        # Exiting the logged_in_user context logs out and asserts that the exit code is 0.

    # Now trying to list experiments should result in an error.
    child = det_spawn(["e", "list"])
    child.expect(".*Unauthenticated.*", timeout=EXPECT_TIMEOUT)
    child.read()
    child.wait()
    assert child.exitstatus != 0

    # Log in as determined.
    log_in_user(authentication.Credentials(constants.DEFAULT_DETERMINED_USER, password))

    # Log back in as new user.
    log_in_user(creds)

    # Now log out as determined.
    assert log_out_user(constants.DEFAULT_DETERMINED_USER) == 0

    # Should still be able to list experiments because new user is logged in.
    child = det_spawn(["e", "list"])
    child.read()
    child.wait()
    child.close()
    assert child.status == 0

    # Change Determined passwordback to "".
    change_user_password(constants.DEFAULT_DETERMINED_USER, "", ADMIN_CREDENTIALS)
    # Clean up.


@pytest.mark.e2e_cpu
@pytest.mark.e2e_cpu_postgres
def test_activate_deactivate(clean_auth: None) -> None:
    creds = create_test_user(ADMIN_CREDENTIALS, True)

    # Make sure we can log in as the user.
    assert log_in_user(creds) == 0

    # Log out.
    assert log_out_user() == 0

    # Deactivate user.
    assert activate_deactivate_user(False, creds.username, ("admin", "")) == 0

    # Attempt to log in again.
    assert log_in_user(creds) != 0

    # Activate user.
    assert activate_deactivate_user(True, creds.username, ("admin", "")) == 0

    # Now log in again.
    assert log_in_user(creds) == 0


@pytest.mark.e2e_cpu
@pytest.mark.e2e_cpu_postgres
def test_change_password(clean_auth: None) -> None:
    # Create a user without a password.
    creds = create_test_user(ADMIN_CREDENTIALS, False)

    # Attempt to log in.
    assert log_in_user(creds) == 0

    # Log out.
    assert log_out_user() == 0

    newPassword = get_random_string()
    assert change_user_password(creds.username, newPassword, ADMIN_CREDENTIALS) == 0

    assert log_in_user(authentication.Credentials(creds.username, newPassword)) == 0


@pytest.mark.e2e_cpu
@pytest.mark.e2e_cpu_postgres
@pytest.mark.e2e_cpu_cross_version
def test_experiment_creation_and_listing(clean_auth: None) -> None:
    # Create 2 users.
    creds1 = create_test_user(ADMIN_CREDENTIALS, True)

    creds2 = create_test_user(ADMIN_CREDENTIALS, True)

    # Create an experiment as first user.
    with logged_in_user(creds1):
        experiment_id1 = exp.run_basic_test(
            conf.fixtures_path("no_op/single.yaml"), conf.fixtures_path("no_op"), 1
        )

    # Create another experiment, this time as second user.
    with logged_in_user(creds2):
        experiment_id2 = exp.run_basic_test(
            conf.fixtures_path("no_op/single.yaml"), conf.fixtures_path("no_op"), 1
        )

    with logged_in_user(creds1):
        # Now it should be the other way around.
        output = extract_id_and_owner_from_exp_list(det_run(["e", "list"]))
        assert (experiment_id1, creds1.username) in output
        assert (experiment_id2, creds2.username) not in output

        # Now use the -a flag to list all experiments.  The output should include both experiments.
        output = extract_id_and_owner_from_exp_list(det_run(["e", "list", "-a"]))
        assert (experiment_id1, creds1.username) in output
        assert (experiment_id2, creds2.username) in output

    with logged_in_user(ADMIN_CREDENTIALS):
        # Clean up.
        delete_experiments(experiment_id1, experiment_id2)


@pytest.mark.e2e_cpu
def test_login_wrong_password(clean_auth: None) -> None:
    creds = create_test_user(ADMIN_CREDENTIALS, True)

    passwd_prompt = f"Password for user '{creds.username}':"
    unauth_error = r".*Forbidden\(invalid credentials\).*"

    child = det_spawn(["user", "login", creds.username])
    child.setecho(True)
    child.expect(passwd_prompt, timeout=EXPECT_TIMEOUT)
    child.sendline("this is the wrong password")

    child.expect(unauth_error, timeout=EXPECT_TIMEOUT)
    child.read()
    child.wait()
    child.close()

    assert child.exitstatus != 0


@pytest.mark.e2e_cpu
def test_login_as_non_existent_user(clean_auth: None) -> None:
    username = "doesNotExist"

    passwd_prompt = f"Password for user '{username}':"
    unauth_error = r".*Forbidden\(user not found\).*"

    child = det_spawn(["user", "login", username])
    child.setecho(True)
    child.expect(passwd_prompt, timeout=EXPECT_TIMEOUT)
    child.sendline("secret")

    child.expect(unauth_error, timeout=EXPECT_TIMEOUT)
    child.read()
    child.wait()
    child.close()

    assert child.exitstatus != 0


@pytest.mark.e2e_cpu
def test_login_with_environment_variables(clean_auth: None) -> None:
    creds = create_test_user(ADMIN_CREDENTIALS, True)
    os.environ["DET_USER"] = creds.username
    os.environ["DET_PASS"] = creds.password
    try:
        child = det_spawn(["user", "whoami"])
        child.expect(creds.username)
        child.wait()
        assert child.exitstatus == 0

        # Can still override with -u.
        with logged_in_user(ADMIN_CREDENTIALS):
            child = det_spawn(["-u", ADMIN_CREDENTIALS.username, "user", "whoami"])
            child.expect(ADMIN_CREDENTIALS.username)
            child.wait()
            assert child.exitstatus == 0
    finally:
        del os.environ["DET_USER"]
        del os.environ["DET_PASS"]


@pytest.mark.e2e_cpu
def test_auth_inside_shell() -> None:
    creds = create_test_user(ADMIN_CREDENTIALS, True)

    with logged_in_user(creds):
        # start a shell
        child = det_spawn(["shell", "start"])
        child.setecho(True)
        # shells take time to start; use the default timeout which is longer
        child.expect(r".*Permanently added.+([0-9a-f-]{36}).+known hosts\.")

        shell_id = child.match.group(1).decode("utf-8")

        def check_whoami(expected_username: str) -> None:
            child.sendline("det user whoami")
            child.expect("You are logged in as user \\'(.*)\\'", timeout=EXPECT_TIMEOUT)
            username = child.match.group(1).decode("utf-8")
            logger.debug(f"They are logged in as user {username}")
            assert username == expected_username

        # check the current user
        check_whoami(creds.username)

        # log in as admin
        child.sendline(f"det user login {ADMIN_CREDENTIALS.username}")
        child.expect(f"Password for user '{ADMIN_CREDENTIALS.username}'", timeout=EXPECT_TIMEOUT)
        child.sendline(ADMIN_CREDENTIALS.password)

        # check that whoami responds with the new user
        check_whoami(ADMIN_CREDENTIALS.username)

        # log out
        child.sendline("det user logout")
        child.expect("#", timeout=EXPECT_TIMEOUT)

        # check that we are back to who we were
        check_whoami(creds.username)

        child.sendline("exit")

        child = det_spawn(["shell", "kill", shell_id])
        child.read()
        child.wait()
        assert child.exitstatus == 0


@pytest.mark.e2e_cpu
def test_login_as_non_active_user(clean_auth: None) -> None:
    creds = create_test_user(ADMIN_CREDENTIALS, True)

    passwd_prompt = f"Password for user '{creds.username}':"
    unauth_error = r".*Forbidden\(user not active\)"

    with logged_in_user(ADMIN_CREDENTIALS):
        child = det_spawn(["user", "deactivate", creds.username])
        child.wait()
        assert child.exitstatus == 0

    child = det_spawn(["user", "login", creds.username])
    child.setecho(True)
    child.expect(passwd_prompt, timeout=EXPECT_TIMEOUT)
    child.sendline(creds.password)

    child.expect(unauth_error, timeout=EXPECT_TIMEOUT)
    child.read()
    child.wait()
    child.close()

    assert child.exitstatus != 0


@pytest.mark.e2e_cpu
def test_non_admin_user_link_with_agent_user(clean_auth: None) -> None:
    creds = create_test_user(ADMIN_CREDENTIALS, True)
    unauth_error = r".*Forbidden.*"

    with logged_in_user(creds):
        child = det_spawn(
            [
                "user",
                "link-with-agent-user",
                creds.username,
                "--agent-uid",
                str(1),
                "--agent-gid",
                str(1),
                "--agent-user",
                creds.username,
                "--agent-group",
                creds.username,
            ]
        )
        child.expect(unauth_error, timeout=EXPECT_TIMEOUT)
        child.read()
        child.wait()
        child.close()

        assert child.exitstatus != 0


@pytest.mark.e2e_cpu
def test_non_admin_commands() -> None:
    command = [
        "det",
        "-m",
        conf.make_master_url(),
        "slot",
        "list",
        "--json",
    ]
    output = subprocess.check_output(command).decode()
    slots = json.loads(output)
    assert len(slots) == 1
    slot_id = slots[0]["slot_id"]
    agent_id = slots[0]["agent_id"]

    enable_slots = ["slot", "enable", agent_id, slot_id]
    disable_slots = ["slot", "disable", agent_id, slot_id]
    enable_agents = ["agent", "enable", agent_id]
    disable_agents = ["agent", "disable", agent_id]
    config = ["master", "config"]
    for cmd in [disable_slots, disable_agents, enable_slots, enable_agents, config]:
        child = det_spawn(["-u", constants.DEFAULT_DETERMINED_USER] + cmd)
        child.expect(".*Forbidden.*", timeout=EXPECT_TIMEOUT)
        child.read()
        child.wait()
        child.close()
        assert child.exitstatus != 0


def run_command() -> str:
    child = det_spawn(["cmd", "run", "echo", "hello"])
    child.expect(r"Scheduling.*\(id: (?P<id>.+?)\)")
    command_id = child.match.groupdict().get("id", None)
    assert command_id is not None
    child.read()
    child.wait()
    assert child.exitstatus == 0
    return cast(str, command_id.decode())


def start_notebook() -> str:
    child = det_spawn(["notebook", "start", "-d"])
    notebook_id = cast(str, child.readline().decode().rstrip())
    child.read()
    child.wait()
    assert child.exitstatus == 0

    return notebook_id


def start_tensorboard(experiment_id: int) -> str:
    child = det_spawn(["tensorboard", "start", "-d", str(experiment_id)])
    tensorboard_id = cast(str, child.readline().decode().rstrip())
    child.read()
    child.wait()
    assert child.exitstatus == 0
    return tensorboard_id


def delete_experiments(*experiment_ids: int) -> None:
    eids = set(experiment_ids)
    while eids:
        output = extract_columns(det_run(["e", "list", "-a"]), [0, 4])

        running_ids = {int(o[0]) for o in output if o[1] == "COMPLETED"}
        intersection = eids & running_ids
        if not intersection:
            time.sleep(0.5)
            continue

        experiment_id = intersection.pop()
        child = det_spawn(["e", "delete", "--yes", str(experiment_id)])
        child.read()
        child.wait()
        assert child.exitstatus == 0
        eids.remove(experiment_id)


def kill_notebooks(*notebook_ids: str) -> None:
    nids = set(notebook_ids)
    while nids:
        output = extract_columns(det_run(["notebook", "list", "-a"]), [0, 3])  # id, state

        # Get set of running IDs.
        running_ids = {task_id for task_id, state in output if state == "RUNNING"}

        intersection = running_ids & nids
        if not intersection:
            time.sleep(0.5)
            continue

        notebook_id = intersection.pop()
        child = det_spawn(["notebook", "kill", notebook_id])
        child.read()
        child.wait()
        assert child.exitstatus == 0
        nids.remove(notebook_id)


def kill_tensorboards(*tensorboard_ids: str) -> None:
    tids = set(tensorboard_ids)
    while tids:
        output = extract_columns(det_run(["tensorboard", "list", "-a"]), [0, 3])

        running_ids = {task_id for task_id, state in output if state == "RUNNING"}

        intersection = running_ids & tids
        if not intersection:
            time.sleep(0.5)
            continue

        tensorboard_id = intersection.pop()
        child = det_spawn(["tensorboard", "kill", tensorboard_id])
        child.read()
        child.wait()
        assert child.exitstatus == 0
        tids.remove(tensorboard_id)


@pytest.mark.e2e_cpu
def test_notebook_creation_and_listing(clean_auth: None) -> None:
    creds1 = create_test_user(ADMIN_CREDENTIALS, True)
    creds2 = create_test_user(ADMIN_CREDENTIALS, True)

    with logged_in_user(creds1):
        notebook_id1 = start_notebook()

    with logged_in_user(creds2):
        notebook_id2 = start_notebook()

        # Listing should only give us user 2's experiment.
        output = extract_columns(det_run(["notebook", "list"]), [0, 1])

    with logged_in_user(creds1):
        output = extract_columns(det_run(["notebook", "list"]), [0, 1])
        assert (notebook_id1, creds1.username) in output
        assert (notebook_id2, creds2.username) not in output

        # Now test listing all.
        output = extract_columns(det_run(["notebook", "list", "-a"]), [0, 1])
        assert (notebook_id1, creds1.username) in output
        assert (notebook_id2, creds2.username) in output

    # Clean up, killing experiments.
    kill_notebooks(notebook_id1, notebook_id2)


@pytest.mark.e2e_cpu
def test_tensorboard_creation_and_listing(clean_auth: None) -> None:
    creds1 = create_test_user(ADMIN_CREDENTIALS, True)
    creds2 = create_test_user(ADMIN_CREDENTIALS, True)

    with logged_in_user(creds1):
        # Create an experiment.
        experiment_id1 = exp.run_basic_test(
            conf.fixtures_path("no_op/single.yaml"), conf.fixtures_path("no_op"), 1
        )

    with logged_in_user(creds1):
        tensorboard_id1 = start_tensorboard(experiment_id1)

    with logged_in_user(creds2):
        experiment_id2 = exp.run_basic_test(
            conf.fixtures_path("no_op/single.yaml"), conf.fixtures_path("no_op"), 1
        )

    with logged_in_user(creds2):
        tensorboard_id2 = start_tensorboard(experiment_id2)

    with logged_in_user(creds1):
        output = extract_columns(det_run(["tensorboard", "list"]), [0, 1])
        assert (tensorboard_id1, creds1.username) in output
        assert (tensorboard_id2, creds2.username) not in output

        output = extract_columns(det_run(["tensorboard", "list", "-a"]), [0, 1])
        assert (tensorboard_id1, creds1.username) in output
        assert (tensorboard_id2, creds2.username) in output

    kill_tensorboards(tensorboard_id1, tensorboard_id2)

    with logged_in_user(ADMIN_CREDENTIALS):
        delete_experiments(experiment_id1, experiment_id2)


@pytest.mark.e2e_cpu
def test_command_creation_and_listing(clean_auth: None) -> None:
    creds1 = create_test_user(ADMIN_CREDENTIALS, True)
    creds2 = create_test_user(ADMIN_CREDENTIALS, True)

    with logged_in_user(creds1):
        command_id1 = run_command()

    with logged_in_user(creds2):
        command_id2 = run_command()

    with logged_in_user(creds1):
        output = extract_columns(det_run(["cmd", "list"]), [0, 1])
        assert (command_id1, creds1.username) in output
        assert (command_id2, creds2.username) not in output

        output = extract_columns(det_run(["cmd", "list", "-a"]), [0, 1])
        assert (command_id1, creds1.username) in output
        assert (command_id2, creds2.username) in output


def create_linked_user(uid: int, user: str, gid: int, group: str) -> authentication.Credentials:
    admin_username, *_rest = ADMIN_CREDENTIALS

    user_creds = create_test_user(ADMIN_CREDENTIALS, False)

    child = det_spawn(
        [
            "-u",
            admin_username,
            "user",
            "link-with-agent-user",
            user_creds.username,
            "--agent-uid",
            str(uid),
            "--agent-gid",
            str(gid),
            "--agent-user",
            user,
            "--agent-group",
            group,
        ]
    )
    child.read()
    child.wait()
    child.close()
    assert child.exitstatus == 0

    return user_creds


@pytest.mark.e2e_cpu
def test_link_with_agent_user(clean_auth: None) -> None:
    user = create_linked_user(200, "someuser", 300, "somegroup")

    expected_output = "someuser:200:somegroup:300"
    with logged_in_user(user), command.interactive_command(
        "cmd", "run", "bash", "-c", "echo $(id -u -n):$(id -u):$(id -g -n):$(id -g)"
    ) as cmd:
        for line in cmd.stdout:
            if expected_output in line:
                break
        else:
            raise AssertionError(f"Did not find {expected_output} in output")


@pytest.mark.e2e_cpu
def test_link_with_large_uid(clean_auth: None) -> None:
    user = create_linked_user(2000000000, "someuser", 2000000000, "somegroup")

    expected_output = "someuser:2000000000:somegroup:2000000000"
    with logged_in_user(user), command.interactive_command(
        "cmd", "run", "bash", "-c", "echo $(id -u -n):$(id -u):$(id -g -n):$(id -g)"
    ) as cmd:
        for line in cmd.stdout:
            if expected_output in line:
                break
        else:
            raise AssertionError(f"Did not find {expected_output} in output")


@pytest.mark.e2e_cpu
def test_link_with_existing_agent_user(clean_auth: None) -> None:
    user = create_linked_user(65534, "nobody", 65534, "nogroup")

    expected_output = "nobody:65534:nogroup:65534"
    with logged_in_user(user), command.interactive_command(
        "cmd", "run", "bash", "-c", "echo $(id -u -n):$(id -u):$(id -g -n):$(id -g)"
    ) as cmd:
        for line in cmd.stdout:
            if expected_output in line:
                break
        else:
            raise AssertionError(f"Did not find {expected_output} in output")


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
def test_non_root_experiment(clean_auth: None, tmp_path: pathlib.Path) -> None:
    user = create_linked_user(65534, "nobody", 65534, "nogroup")

    with logged_in_user(user):
        with open(conf.fixtures_path("no_op/model_def.py")) as f:
            model_def_content = f.read()

        with open(conf.fixtures_path("no_op/single-one-short-step.yaml")) as f:
            config = yaml.safe_load(f)

        # Use a user-owned path to ensure shared_fs uses the container_path and not host_path.
        with non_tmp_shared_fs_path() as host_path:
            config["checkpoint_storage"] = {
                "type": "shared_fs",
                "host_path": host_path,
            }

            # Call `det --version` in a startup hook to ensure that det is on the PATH.
            with FileTree(
                tmp_path,
                {
                    "startup-hook.sh": "det --version || exit 77",
                    "const.yaml": yaml.dump(config),  # type: ignore
                    "model_def.py": model_def_content,
                },
            ) as tree:
                exp.run_basic_test(str(tree.joinpath("const.yaml")), str(tree), None)


@pytest.mark.e2e_cpu
def test_link_without_agent_user(clean_auth: None) -> None:
    user = create_test_user(ADMIN_CREDENTIALS, False)

    expected_output = "root:0:root:0"
    with logged_in_user(user), command.interactive_command(
        "cmd", "run", "bash", "-c", "echo $(id -u -n):$(id -u):$(id -g -n):$(id -g)"
    ) as cmd:
        recvd = []
        for line in cmd.stdout:
            if expected_output in line:
                break
            recvd.append(line)
        else:
            output = "".join(recvd)
            raise AssertionError(f"Did not find {expected_output} in output:\n{output}")


@pytest.mark.e2e_cpu
def test_non_root_shell(clean_auth: None, tmp_path: pathlib.Path) -> None:
    user = create_linked_user(1234, "someuser", 1234, "somegroup")

    expected_output = "someuser:1234:somegroup:1234"

    with logged_in_user(user), command.interactive_command("shell", "start") as shell:
        shell.stdin.write(b"echo $(id -u -n):$(id -u):$(id -g -n):$(id -g)\n")
        shell.stdin.close()

        for line in shell.stdout:
            if expected_output in line:
                break
        else:
            raise AssertionError(f"Did not find {expected_output} in output")


@pytest.mark.e2e_cpu
def test_experiment_delete() -> None:
    user = create_test_user(ADMIN_CREDENTIALS)
    non_owner_user = create_test_user(ADMIN_CREDENTIALS)

    with logged_in_user(user):
        experiment_id = exp.run_basic_test(
            conf.fixtures_path("no_op/single.yaml"), conf.fixtures_path("no_op"), 1
        )

    with logged_in_user(non_owner_user):
        # "det experiment delete" call should fail, because the user is not an admin and
        # doesn't own the experiment.
        child = det_spawn(["experiment", "delete", str(experiment_id), "--yes"])
        child.read()
        child.wait()
        assert child.exitstatus > 0

    with logged_in_user(user):
        child = det_spawn(["experiment", "delete", str(experiment_id), "--yes"])
        child.read()
        child.wait()
        assert child.exitstatus == 0

        experiment_delete_deadline = time.time() + 5 * 60
        while 1:
            child = det_spawn(["experiment", "describe", str(experiment_id)])
            child.read()
            child.wait()
            # "det experiment describe" call should fail, because the
            # experiment is no longer in the database.
            if child.exitstatus > 0:
                return
            elif time.time() > experiment_delete_deadline:
                pytest.fail("experiment didn't delete after timeout")


@pytest.mark.e2e_cpu
@pytest.mark.e2e_cpu_postgres
def test_change_displayname(clean_auth: None) -> None:
    u_patch = create_test_user(ADMIN_CREDENTIALS, False)
    original_name = u_patch.username

    master_url = conf.make_master_url()
    certs.cli_cert = certs.default_load(master_url)
    authentication.cli_auth = authentication.Authentication(
        conf.make_master_url(), requested_user=original_name, password="", try_reauth=True
    )
    sess = api.Session(master_url, original_name, authentication.cli_auth, certs.cli_cert)

    # Get API bindings object for the created test user
    all_users = bindings.get_GetUsers(sess).users
    assert all_users is not None
    current_user = list(filter(lambda u: u.username == original_name, all_users))[0]
    assert current_user is not None and current_user.id

    # Rename user using display name
    patch_user = bindings.v1PatchUser(displayName="renamed")
    bindings.patch_PatchUser(sess, body=patch_user, userId=current_user.id)

    modded_user = bindings.get_GetUser(sess, userId=current_user.id).user
    assert modded_user is not None
    assert modded_user.displayName == "renamed"

    # Avoid display name of 'admin'
    patch_user.displayName = "Admin"
    with pytest.raises(errors.APIException):
        bindings.patch_PatchUser(sess, body=patch_user, userId=current_user.id)

    # Clear display name (UI will show username)
    patch_user.displayName = ""
    bindings.patch_PatchUser(sess, body=patch_user, userId=current_user.id)

    modded_user = bindings.get_GetUser(sess, userId=current_user.id).user
    assert modded_user is not None
    assert modded_user.displayName == ""
