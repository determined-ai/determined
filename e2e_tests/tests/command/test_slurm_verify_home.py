import logging

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

    foundLineWithUserName = False

    with cmd.interactive_command("shell", "start") as shell:
        # In order to identify whether we are running a Podman container, we will
        # check if we're running as "root", because Podman containers run as
        # "root".  Use "$(whoami)" to report the user we running as, because the
        # "$USER" environment variable will report the launching user.
        shell.stdin.write(b'COLUMNS=80 echo "USER=$(whoami), HOME=$HOME"\n')
        # Exit the shell, so we can read output below until EOF
        shell.stdin.write(b"exit\n")
        shell.stdin.close()

        for line in shell.stdout:
            logging.info(f"OUT: {line}")

            # We're only interested in the line containing "USER=".
            if "USER=" not in line:
                continue

            foundLineWithUserName = True

            if "USER=root" in line:
                # If we're running as "root" inside the container, it implies
                # we're using Podman as the container run type. For Podman, the
                # home directory in "/run/determined/etc/passwd" is based on the
                # Determined "work_dir" setting, which is
                # "/run/determined/workdir" by default.
                if "HOME=/run/determined/workdir" not in line:
                    pytest.fail(
                        "FAILURE: $HOME in HPC Job is set to an "
                        f"unexpected value for Podman: {line}"
                    )
            else:
                # For Singularity and Enroot, the home directory will be the
                # user's home directory on the host, which is different for each
                # user, so we can't check for a specific value. The best we can
                # do is to check that it's not set to the Determined "work_dir".
                if "HOME=/run/determined/workdir" in line:
                    pytest.fail(
                        "FAILURE: $HOME in HPC Job is set to an "
                        f"unexpected value for Singularity/Enroot: {line}"
                    )

    if not foundLineWithUserName:
        pytest.fail("FAILURE: Did not find line containing the username")
