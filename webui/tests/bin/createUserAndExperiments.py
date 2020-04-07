#!/usr/bin/env python

import pathlib
import subprocess

determined_root_dir = pathlib.Path(__file__).absolute().parents[3]
external_examples_dir = determined_root_dir.joinpath("examples", "official")
experiment_dir = external_examples_dir.joinpath("mnist_tf_keras")


for _ in range(4):
    subprocess.run(
        [
            "det",
            "experiment",
            "create",
            str(experiment_dir.joinpath("const.yaml")),
            str(experiment_dir),
        ],
        check=True,
    )


# Create a non-default user for testing purposes. The CLI reads a password using `getpass`, which
# reads from `/dev/tty`; Pexpect is a portable way to simulate the TTY input. (On Linux, `echo |
# setsid -w determined ...` would work (https://unix.stackexchange.com/a/68591),
# but the `setsid` command does not seem to be generally available on MacOS.)
# FIXME The script is blocking here on MacOS. We disable this and its corresponding tests for the
# time being until it's fixed.
# p = pexpect.spawn("det", ["-u", "admin", "u", "create", "hoid"])
# p.expect("Password.*:")
# p.sendline("")
# p.wait()
