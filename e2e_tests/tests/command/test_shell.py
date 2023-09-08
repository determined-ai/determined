import pathlib

import pytest

import determined as det
from tests import api_utils
from tests import command as cmd
from tests import detproc


@pytest.mark.slow
@pytest.mark.e2e_gpu
@pytest.mark.e2e_slurm
@pytest.mark.e2e_pbs
def test_start_and_write_to_shell(tmp_path: pathlib.Path) -> None:
    sess = api_utils.user_session()
    with cmd.interactive_command(sess, ["shell", "start"]) as shell:
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
    sess = api_utils.user_session()
    with cmd.interactive_command(sess, ["shell", "start", "--detach"]) as shell:
        assert shell.task_id
        command = ["det", "shell", "open", shell.task_id, "det", "user", "whoami"]
        output = detproc.check_output(sess, command)
        assert "You are logged in as user" in output
