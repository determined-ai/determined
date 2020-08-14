import re
import subprocess
import tempfile
import time
from pathlib import Path
from typing import Any, List

import docker
import docker.errors
import pytest
import yaml

from tests import command as cmd
from tests import config as conf
from tests.filetree import FileTree


@pytest.mark.slow  # type: ignore
@pytest.mark.e2e_cpu  # type: ignore
def test_cold_and_warm_start(tmp_path: Path) -> None:
    for _ in range(3):
        subprocess.check_call(
            ["det", "-m", conf.make_master_url(), "cmd", "run", "echo", "hello", "world"]
        )


def _run_and_return_real_exit_status(args: List[str], **kwargs: Any) -> None:
    """
    Wraps subprocess.check_call and extracts exit status from output.
    """
    # TODO(#2903): remove this once exit status are propagated through cli
    output = subprocess.check_output(args, **kwargs)
    if re.search(b"finished command \\S+ task failed with exit code", output):
        raise subprocess.CalledProcessError(1, " ".join(args), output=output)


def _run_and_verify_exit_code_zero(args: List[str], **kwargs: Any) -> None:
    """Wraps subprocess.check_output and verifies a successful exit code."""
    # TODO(#2903): remove this once exit status are propagated through cli
    output = subprocess.check_output(args, **kwargs)
    assert re.search(b"command exited successfully", output) is not None


def _run_and_verify_failure(args: List[str], message: str, **kwargs: Any) -> None:
    output = subprocess.check_output(args, **kwargs)
    if re.search(message.encode(), output):
        raise subprocess.CalledProcessError(1, " ".join(args), output=output)


@pytest.mark.e2e_cpu  # type: ignore
def test_exit_code_reporting() -> None:
    """
    Confirm that failed commands are not reported as successful, and confirm
    that our test infrastructure is valid.
    """
    with pytest.raises(AssertionError):
        _run_and_verify_exit_code_zero(["det", "-m", conf.make_master_url(), "cmd", "run", "false"])


@pytest.mark.slow  # type: ignore
@pytest.mark.e2e_cpu  # type: ignore
def test_basic_workflows(tmp_path: Path) -> None:
    with FileTree(tmp_path, {"hello.py": "print('hello world')"}) as tree:
        _run_and_verify_exit_code_zero(
            [
                "det",
                "-m",
                conf.make_master_url(),
                "cmd",
                "run",
                "--context",
                str(tree),
                "python",
                "hello.py",
            ]
        )

    with FileTree(tmp_path, {"hello.py": "print('hello world')"}) as tree:
        link = tree.joinpath("hello-link.py")
        link.symlink_to(tree.joinpath("hello.py"))
        _run_and_verify_exit_code_zero(
            [
                "det",
                "-m",
                conf.make_master_url(),
                "cmd",
                "run",
                "--context",
                str(tree),
                "python",
                "hello-link.py",
            ]
        )

    _run_and_verify_exit_code_zero(
        ["det", "-m", conf.make_master_url(), "cmd", "run", "python", "-c", "print('hello world')"]
    )

    with pytest.raises(subprocess.CalledProcessError):
        _run_and_return_real_exit_status(
            [
                "det",
                "-m",
                conf.make_master_url(),
                "cmd",
                "run",
                "--context",
                "non-existent-path-here",
                "python",
                "hello.py",
            ]
        )


@pytest.mark.slow  # type: ignore
@pytest.mark.e2e_cpu  # type: ignore
def test_large_uploads(tmp_path: Path) -> None:
    with pytest.raises(subprocess.CalledProcessError):
        with FileTree(tmp_path, {"hello.py": "print('hello world')"}) as tree:
            large = tree.joinpath("large-file.bin")
            large.touch()
            f = large.open(mode="w")
            f.seek(1024 * 1024 * 120)
            f.write("\0")
            f.close()

            _run_and_return_real_exit_status(
                [
                    "det",
                    "-m",
                    conf.make_master_url(),
                    "cmd",
                    "run",
                    "--context",
                    str(tree),
                    "python",
                    "hello.py",
                ]
            )

    with FileTree(tmp_path, {"hello.py": "print('hello world')", ".detignore": "*.bin"}) as tree:
        large = tree.joinpath("large-file.bin")
        large.touch()
        f = large.open(mode="w")
        f.seek(1024 * 1024 * 120)
        f.write("\0")
        f.close()

        _run_and_verify_exit_code_zero(
            [
                "det",
                "-m",
                conf.make_master_url(),
                "cmd",
                "run",
                "--context",
                str(tree),
                "python",
                "hello.py",
            ]
        )


@pytest.mark.slow  # type: ignore
@pytest.mark.e2e_cpu  # type: ignore
def test_configs(tmp_path: Path) -> None:
    with FileTree(
        tmp_path,
        {
            "config.yaml": """
resources:
  slots: 1
environment:
  environment_variables:
   - TEST=TEST
"""
        },
    ) as tree:
        config_path = tree.joinpath("config.yaml")
        _run_and_verify_exit_code_zero(
            [
                "det",
                "-m",
                conf.make_master_url(),
                "cmd",
                "run",
                "--config-file",
                str(config_path),
                "python",
                "-c",
                """
import os
test = os.environ["TEST"]
if test != "TEST":
    print("{} != {}".format(test, "TEST"))
    sys.exit(1)
""",
            ]
        )


@pytest.mark.slow  # type: ignore
@pytest.mark.e2e_cpu  # type: ignore
def test_singleton_command() -> None:
    _run_and_verify_exit_code_zero(
        ["det", "-m", conf.make_master_url(), "cmd", "run", "echo hello && echo world"]
    )


@pytest.mark.slow  # type: ignore
@pytest.mark.e2e_cpu  # type: ignore
def test_absolute_bind_mount(tmp_path: Path) -> None:
    _run_and_verify_exit_code_zero(
        [
            "det",
            "-m",
            conf.make_master_url(),
            "cmd",
            "run",
            "--volume",
            "/bin:/foo-bar",
            "ls",
            "/foo-bar",
        ]
    )

    with FileTree(
        tmp_path,
        {
            "config.yaml": """
bind_mounts:
- host_path: /bin
  container_path: /foo-bar
"""
        },
    ) as tree:
        config_path = tree.joinpath("config.yaml")
        _run_and_verify_exit_code_zero(
            [
                "det",
                "-m",
                conf.make_master_url(),
                "cmd",
                "run",
                "--volume",
                "/bin:/foo-bar2",
                "--config-file",
                str(config_path),
                "ls",
                "/foo-bar",
                "/foo-bar2",
            ]
        )


@pytest.mark.slow  # type: ignore
@pytest.mark.e2e_cpu  # type: ignore
def test_relative_bind_mount(tmp_path: Path) -> None:
    _run_and_verify_exit_code_zero(
        [
            "det",
            "-m",
            conf.make_master_url(),
            "cmd",
            "run",
            "--volume",
            "/bin:foo-bar",
            "ls",
            "foo-bar",
        ]
    )
    with FileTree(
        tmp_path,
        {
            "config.yaml": """
bind_mounts:
- host_path: /bin
  container_path: foo-bar
"""
        },
    ) as tree:
        config_path = tree.joinpath("config.yaml")
        _run_and_verify_exit_code_zero(
            [
                "det",
                "-m",
                conf.make_master_url(),
                "cmd",
                "run",
                "--volume",
                "/bin:foo-bar2",
                "--config-file",
                str(config_path),
                "ls",
                "foo-bar",
                "foo-bar2",
            ]
        )


@pytest.mark.slow  # type: ignore
@pytest.mark.e2e_cpu  # type: ignore
def test_cmd_kill() -> None:
    """Start a command, extract its task ID, and then kill it."""

    with cmd.interactive_command(
        "command", "run", "echo hello world; echo hello world; sleep infinity"
    ) as command:
        assert command.task_id is not None
        for line in command.stdout:
            if "hello world" in line:
                assert cmd.get_num_running_commands() == 1
                break


@pytest.mark.slow  # type: ignore
@pytest.mark.e2e_cpu  # type: ignore
def test_image_pull_after_remove() -> None:
    """
    Remove pulled image and verify that it will be pulled again with auth.
    """
    client = docker.from_env()
    try:
        client.images.remove("alpine:3.10")
    except docker.errors.ImageNotFound:
        pass

    _run_and_verify_exit_code_zero(
        [
            "det",
            "-m",
            conf.make_master_url(),
            "cmd",
            "run",
            "--config",
            "environment.image=alpine:3.10",
            "sleep 3; echo hello world",
        ]
    )


@pytest.mark.slow  # type: ignore
@pytest.mark.e2e_cpu  # type: ignore
def test_killed_pending_command_terminates() -> None:
    # Specify an outrageous number of slots to be sure that it can't be scheduled.
    with cmd.interactive_command(
        "cmd", "run", "--config", "resources.slots=1048576", "sleep infinity"
    ) as command:
        for _ in range(10):
            assert cmd.get_command(command.task_id)["state"] == "PENDING"
            time.sleep(1)

    # The command is killed when the context is exited; now it should reach TERMINATED soon.
    for _ in range(5):
        if cmd.get_command(command.task_id)["state"] == "TERMINATED":
            break
        time.sleep(1)
    else:
        state = cmd.get_command(command.task_id)["state"]
        raise AssertionError(f"Task was in state {state} rather than TERMINATED")


@pytest.mark.e2e_gpu  # type: ignore
def test_k8_mount(using_k8s: bool) -> None:
    if not using_k8s:
        pytest.skip("only need to run test on kubernetes")

    mount_path = "/ci/"

    with pytest.raises(subprocess.CalledProcessError):
        _run_and_verify_failure(
            ["det", "-m", conf.make_master_url(), "cmd", "run", f"sleep 3; touch {mount_path}"],
            "No such file or directory",
        )

    with tempfile.NamedTemporaryFile() as tf:
        config = {
            "environment": {
                "pod_spec": {
                    "spec": {
                        "containers": [
                            {"volumeMounts": [{"name": "temp1", "mountPath": mount_path}]}
                        ],
                        "volumes": [{"name": "temp1", "emptyDir": {}}],
                    }
                }
            }
        }

        with open(tf.name, "w") as f:
            yaml.dump(config, f)

        _run_and_verify_exit_code_zero(
            [
                "det",
                "-m",
                conf.make_master_url(),
                "cmd",
                "run",
                "--config-file",
                tf.name,
                f"sleep 3; touch {mount_path}",
            ]
        )
