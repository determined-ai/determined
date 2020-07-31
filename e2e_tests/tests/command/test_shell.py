from pathlib import Path

import pytest

from tests import command as cmd


@pytest.mark.slow  # type: ignore
@pytest.mark.e2e_gpu  # type: ignore
def test_start_and_write_to_shell(tmp_path: Path) -> None:
    with cmd.interactive_command("shell", "start") as shell:
        shell.stdin.write(b"echo hello world\n")
        shell.stdin.close()

        for line in shell.stdout:
            if "hello world" in line:
                break
        else:
            pytest.fail("Did not find expected input 'hello world' in shell stdout.")
