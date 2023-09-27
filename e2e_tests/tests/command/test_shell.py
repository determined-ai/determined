from pathlib import Path

import pexpect
import pytest

import determined as det
from determined.common.api import task_is_ready
from tests import command as cmd
from tests.api_utils import determined_test_session
from tests.cluster.test_users import det_spawn
from tests.cluster.utils import wait_for_task_state


@pytest.mark.slow
@pytest.mark.e2e_gpu
def test_start_and_write_to_shell(tmp_path: Path) -> None:
    with cmd.interactive_command("shell", "start") as shell:
        # Call our cli to ensure that PATH and PYTHONUSERBASE are properly set.
        shell.stdin.write(b"COLUMNS=80 det --version\n")
        # Exit the shell, so we can read output below until EOF instead of timeout
        shell.stdin.write(b"exit\n")
        shell.stdin.close()

        lines = ""
        for line in shell.stdout:
            if str(det.__version__) in line:
                break
            lines += "OUTPUT:" + line
        else:
            pytest.fail(
                f"Did not find expected input {det.__version__} in shell stdout." + lines + "\n"
            )


@pytest.mark.e2e_cpu
def test_open_shell() -> None:
    with cmd.interactive_command("shell", "start", "--detach") as shell:
        task_id = shell.task_id
        assert task_id is not None

        wait_for_task_state("shell", task_id, "RUNNING")

        # Shell should fail because it is not ready yet it should not hang.
        child = det_spawn(["shell", "open", task_id])
        child.expect(pexpect.EOF, timeout=5)
        child.wait()
        print(child.exitstatus)
        assert child.exitstatus != 0

        task_is_ready(determined_test_session(), task_id)

        child = det_spawn(["shell", "open", task_id])
        child.setecho(True)
        child.expect(r".*Permanently added.+([0-9a-f-]{36}).+known hosts\.")
        child.sendline("det user whoami")
        child.expect("You are logged in as user \\'(.*)\\'", timeout=10)
        child.sendline("exit")
        child.read()
        child.wait()
        assert child.exitstatus == 0
