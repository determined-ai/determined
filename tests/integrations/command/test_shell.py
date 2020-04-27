from pathlib import Path

import pytest

from tests.integrations import command as cmd


@pytest.mark.slow  # type: ignore
@pytest.mark.e2e_gpu  # type: ignore
def test_start_and_write_to_shell(tmp_path: Path) -> None:
    pytest.skip("This is an extremely flaky test that needs to be rewritten")
    """Start a shell, extract its task ID, and then kill it."""

    with cmd.interactive_command("shell", "start") as shell:
        shell.stdin.write(b"echo hello world")
        shell.stdin.close()

        for line in shell.stdout:
            if "hello world" in line:
                break
        else:
            pytest.fail("Did not find expected input 'hello world' in shell stdout.")
