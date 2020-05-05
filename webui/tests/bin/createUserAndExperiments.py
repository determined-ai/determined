#!/usr/bin/env python

import pathlib
import pexpect
import subprocess

test_username = "userwithpw"
test_password = "specialpw"


def wait_for_process(process):
    # Avoid hang on macOS
    # https://github.com/pytest-dev/pytest/issues/2022
    while True:
        try:
            process.read_nonblocking()
        except Exception:
            break

    if process.isalive():
        process.wait()


# Create a non-default user for testing purposes. The CLI reads a password using `getpass`, which
# reads from `/dev/tty`; Pexpect is a portable way to simulate the TTY input. (On Linux, `echo |
# setsid -w determined ...` would work (https://unix.stackexchange.com/a/68591),
# but the `setsid` command does not seem to be generally available on macOS.)
p = pexpect.spawn("det", ["-u", "admin", "user", "create", test_username])
p.expect("Password.*:")
p.sendline("")
wait_for_process(p)

# Change the password for the newly create user
p = pexpect.spawn("det", ["-u", "admin", "user", "change-password", test_username])
p.expect("New password.*:")
p.sendline(test_password)
p.expect("Confirm password:")
p.sendline(test_password)
wait_for_process(p)

# Log into CLI as the newly created user
p = pexpect.spawn("det", ["user", "login", test_username])
p.expect("Password.*:")
p.sendline(test_password)
wait_for_process(p)

# Create experiments
determined_root_dir = pathlib.Path(__file__).absolute().parents[3]
experiment_dir = determined_root_dir.joinpath("e2e_tests", "tests", "fixtures", "no_op")

for _ in range(4):
    subprocess.run(
        [
            "det",
            "experiment",
            "create",
            str(experiment_dir.joinpath("single-very-many-long-steps.yaml")),
            str(experiment_dir),
        ],
        check=True,
    )
