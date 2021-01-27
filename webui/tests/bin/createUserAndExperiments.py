#!/usr/bin/env python

import pathlib
import pexpect
import subprocess

USER_WITH_PASSWORD_USERNAME = "user-w-pw"
USER_WITH_PASSWORD_PASSWORD = "special-pw"
USER_WITHOUT_PASSWORD_USERNAME = "user-wo-pw"

determined_root_dir = pathlib.Path(__file__).absolute().parents[3]
experiments_dir = determined_root_dir.joinpath("webui", "tests", "fixtures", "experiments")
noop_dir = determined_root_dir.joinpath("e2e_tests", "tests", "fixtures", "no_op")
noop_config = "single-very-many-long-steps.yaml"


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


def login_as(username, password=""):
    print(f"logging in as {username} ... ", end="")
    p = pexpect.spawn("det", ["user", "login", username])
    p.expect("Password.*:")
    p.sendline(password)
    wait_for_process(p)
    print("done")


def create_user(username, password=""):
    print(f"creating user {username} ... ", end="")
    p = pexpect.spawn("det", ["-u", "admin", "user", "create", username])
    p.expect(pexpect.EOF)
    wait_for_process(p)
    print("done")

    if password != "":
        print(f"update password for {username} ... ", end="")
        p = pexpect.spawn("det", ["-u", "admin", "user", "change-password", username])
        p.expect("New password.*:")
        print(f"password: {password}")
        p.sendline(password)
        p.expect("Confirm password:")
        p.sendline(password)
        wait_for_process(p)
        print("done")


# Create experiments
def createExperiment(directory, config_file, count):
    cmd = [
        "det",
        "experiment",
        "create",
        str(directory.joinpath(config_file)),
        str(directory),
    ]

    procs = [subprocess.Popen(cmd) for _ in range(count)]
    for p in procs:
        p.wait()


def main():
    print("setting up users...")

    # First login as admin to avoid having to authenticate downstream
    login_as("admin")

    # Create a non-default user without a password
    create_user(USER_WITHOUT_PASSWORD_USERNAME)

    # Create a non-default user with a password
    create_user(USER_WITH_PASSWORD_USERNAME, USER_WITH_PASSWORD_PASSWORD)

    # Login as non-default user with password
    login_as(USER_WITH_PASSWORD_USERNAME, USER_WITH_PASSWORD_PASSWORD)

    print("creating experiments..")
    createExperiment(noop_dir, noop_config, 4)
    createExperiment(experiments_dir.joinpath("no-op-metrics"), experiments_dir.joinpath("no-op-metrics", "single.yaml"), 4)


if __name__ == "__main__":
    main()
