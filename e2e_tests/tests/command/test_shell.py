from pathlib import Path

import pytest

import determined as det
from tests import command as cmd


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
