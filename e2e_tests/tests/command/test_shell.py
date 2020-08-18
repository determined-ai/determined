from pathlib import Path

import pytest

import determined as det
from tests import command as cmd


@pytest.mark.slow  # type: ignore
@pytest.mark.e2e_gpu  # type: ignore
def test_start_and_write_to_shell(tmp_path: Path) -> None:
    with cmd.interactive_command("shell", "start") as shell:
        # Call our cli to ensure that PATH and PYTHONUSERBASE are properly set.
        shell.stdin.write(b"det --version\n")
        shell.stdin.close()

        for line in shell.stdout:
            if str(det.__version__) in line:
                break
        else:
            pytest.fail(f"Did not find expected input {det.__version__} in shell stdout.")
