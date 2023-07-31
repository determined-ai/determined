import pytest

import tests.command as cmd


@pytest.mark.e2e_slurm
@pytest.mark.e2e_pbs
def test_start_and_verify_hpc_home() -> None:
    """
    Verify that Slurm/PBS jobs retain the user's $HOME directory from the cluster.
    Using a shell we display the value of the HOME variable and verify that
    we don't find /run/determined/workdir in the output which
    is the default for non-HPC jobs.

    For some reason a command (as opposed to a shell) retains the expected
    user $HOME even when the /etc/nsswitch.conf is mis-configured, so this
    test uses a shell.
    """

    with cmd.interactive_command("shell", "start") as shell:
        shell.stdin.write(b"COLUMNS=80 printenv HOME\n")
        # Exit the shell, so we can read output below until EOF
        shell.stdin.write(b"exit\n")
        shell.stdin.close()

        for line in shell.stdout:
            print(f"OUT: {line}")
            if "/run/determined/workdir" in line:
                pytest.fail(f"FAILURE: $HOME in HPC Job is incorrectly set to: {line}")
            if line.startswith("/"):
                print(f"Found HOME={line}\n")
