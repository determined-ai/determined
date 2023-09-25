import copy
import re
import subprocess
import tempfile
import time
from pathlib import Path
from typing import Any, Dict, List, Optional

import docker
import docker.errors
import pytest

from determined.common import yaml
from tests import command as cmd
from tests import config as conf
from tests.filetree import FileTree


@pytest.mark.slow
@pytest.mark.e2e_cpu
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
    assert re.search(b"resources exited successfully", output) is not None, "Output is: {}".format(
        output.decode("utf-8")
    )


def _run_and_verify_failure(args: List[str], message: str, **kwargs: Any) -> None:
    output = subprocess.check_output(args, **kwargs)
    if re.search(message.encode(), output):
        raise subprocess.CalledProcessError(1, " ".join(args), output=output)


def _run_cmd_with_config_expecting_success(
    cmd: str, config: Dict[str, Any], context_path: Optional[str] = None
) -> None:
    with tempfile.NamedTemporaryFile() as tf:
        with open(tf.name, "w") as f:
            yaml.dump(config, f)

        command = ["det", "-m", conf.make_master_url(), "cmd", "run", "--config-file", tf.name]
        if context_path:
            command += ["-c", context_path]
        command.append(cmd)

        _run_and_verify_exit_code_zero(command)


def _run_cmd_with_config_expecting_failure(
    cmd: str, expected_failure: str, config: Dict[str, Any]
) -> None:
    with tempfile.NamedTemporaryFile() as tf:
        with open(tf.name, "w") as f:
            yaml.dump(config, f)

        with pytest.raises(subprocess.CalledProcessError):
            _run_and_verify_failure(
                [
                    "det",
                    "-m",
                    conf.make_master_url(),
                    "cmd",
                    "run",
                    "--config-file",
                    tf.name,
                    cmd,
                ],
                expected_failure,
            )


@pytest.mark.e2e_cpu
def test_exit_code_reporting() -> None:
    """
    Confirm that failed commands are not reported as successful, and confirm
    that our test infrastructure is valid.
    """
    with pytest.raises(AssertionError):
        _run_and_verify_exit_code_zero(["det", "-m", conf.make_master_url(), "cmd", "run", "false"])


@pytest.mark.slow
@pytest.mark.e2e_cpu
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


@pytest.mark.slow
@pytest.mark.e2e_cpu
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


# TODO we could move this test to nightly or even per release to save CI cost.
# It takes around 15 seconds.
@pytest.mark.e2e_k8s
def test_context_directory_larger_than_config_map_k8s(tmp_path: Path) -> None:
    with FileTree(tmp_path, {"hello.py": "print('hello world')"}) as tree:
        large = tree.joinpath("large-file.bin")
        large.touch()
        f = large.open(mode="w")
        f.seek(1024 * 1024 * 10)
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


@pytest.mark.slow
@pytest.mark.e2e_cpu
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


@pytest.mark.slow
@pytest.mark.e2e_cpu
def test_singleton_command() -> None:
    _run_and_verify_exit_code_zero(
        ["det", "-m", conf.make_master_url(), "cmd", "run", "echo hello && echo world"]
    )


@pytest.mark.slow
@pytest.mark.e2e_cpu
def test_environment_variables_command() -> None:
    _run_and_verify_exit_code_zero(
        [
            "det",
            "-m",
            conf.make_master_url(),
            "cmd",
            "run",
            "--config",
            "environment.environment_variables='THISISTRUE=true','WONTCAUSEPANIC'",
            'if [ "$THISISTRUE" != "true" ]; then exit 1; fi',
        ]
    )


@pytest.mark.parametrize("actual,expected", [("24576", "24"), ("1.5g", "1572864")])
@pytest.mark.parametrize("use_config_file", [True, False])
@pytest.mark.slow
@pytest.mark.e2e_cpu
def test_shm_size_command(
    tmp_path: Path, actual: str, expected: str, use_config_file: bool
) -> None:
    with FileTree(
        tmp_path,
        {
            "config.yaml": f"""
resources:
  shm_size: {actual}
"""
        },
    ) as tree:
        config_path = tree.joinpath("config.yaml")
        cmd = ["det", "-m", conf.make_master_url(), "cmd", "run"]
        if use_config_file:
            cmd += ["--config-file", str(config_path)]
        else:
            cmd += ["--config", f"resources.shm_size={actual}"]
        cmd += [
            f"""df /dev/shm && \
df /dev/shm | \
tail -1 | \
[ "$(awk '{{print $2}}')" = '{expected}' ]"""
        ]
        _run_and_verify_exit_code_zero(cmd)


@pytest.mark.slow
@pytest.mark.e2e_cpu
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


@pytest.mark.slow
@pytest.mark.e2e_cpu
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


@pytest.mark.slow
@pytest.mark.e2e_cpu
def test_cmd_kill() -> None:
    """Start a command, extract its task ID, and then kill it."""

    with cmd.interactive_command(
        "command", "run", "echo hello world; echo hello world; sleep infinity"
    ) as command:
        assert command.task_id is not None
        for line in command.stdout:
            if "hello world" in line:
                # For HPC job, dispatcher does the polling of the job state happens
                # every 10 seconds. For example, it is very likely the current job state is
                # STATE_PULLING when job is actually running on HPC. So instead of checking
                # for STATE_RUNNING, we check for other active states as well.
                assert cmd.get_num_active_commands() == 1
                break


@pytest.mark.slow
@pytest.mark.e2e_cpu
def test_image_pull_after_remove() -> None:
    """
    Remove pulled image and verify that it will be pulled again with auth.
    """
    client = docker.from_env()
    try:
        client.images.remove("python:3.8.16")
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
            "environment.image=python:3.8.16",
            "sleep 3; echo hello world",
        ]
    )


@pytest.mark.slow
@pytest.mark.e2e_cpu
def test_killed_pending_command_terminates() -> None:
    # Specify an outrageous number of slots to be sure that it can't be scheduled.
    # NB: slot # higher than postgres smallint (i.e. 32k) is rejected outright.
    with cmd.interactive_command(
        "cmd", "run", "--config", "resources.slots=10485", "sleep infinity"
    ) as command:
        assert command.task_id is not None
        for _ in range(10):
            assert cmd.get_command(command.task_id)["state"] == "STATE_QUEUED"
            time.sleep(1)

    # The command is killed when the context is exited; now it should reach TERMINATED soon.
    for _ in range(5):
        if cmd.get_command(command.task_id)["state"] == "STATE_TERMINATED":
            break
        time.sleep(1)
    else:
        state = cmd.get_command(command.task_id)["state"]
        raise AssertionError(f"Task was in state {state} rather than STATE_TERMINATED")


@pytest.mark.e2e_gpu
@pytest.mark.parametrize("sidecar", [True, False])
def test_k8_mount(using_k8s: bool, sidecar: bool) -> None:
    if not using_k8s:
        pytest.skip("only need to run test on kubernetes")

    mount_path = "/ci/"

    with pytest.raises(subprocess.CalledProcessError):
        _run_and_verify_failure(
            ["det", "-m", conf.make_master_url(), "cmd", "run", f"sleep 3; touch {mount_path}"],
            "No such file or directory",
        )

    config = {
        "environment": {
            "pod_spec": {
                "spec": {
                    "containers": [
                        {
                            "name": "determined-container",
                            "volumeMounts": [{"name": "temp1", "mountPath": mount_path}],
                        }
                    ],
                    "volumes": [{"name": "temp1", "emptyDir": {}}],
                }
            }
        }
    }

    if sidecar:
        sidecar_container = {
            "name": "sidecar",
            "image": conf.TF2_CPU_IMAGE,
            "command": ["/bin/bash"],
            "args": ["-c", "exit 0"],
        }

        # We insert this as the first container, to make sure Determined can handle the case
        # where the `determined-container` is not the first one.
        config["environment"]["pod_spec"]["spec"]["containers"] = [
            sidecar_container,
            config["environment"]["pod_spec"]["spec"]["containers"][0],  # type: ignore
        ]

    _run_cmd_with_config_expecting_success(cmd=f"sleep 3; touch {mount_path}", config=config)


@pytest.mark.e2e_gpu
def test_k8_init_containers(using_k8s: bool) -> None:
    if not using_k8s:
        pytest.skip("only need to run test on kubernetes")

    config = {
        "environment": {
            "pod_spec": {
                "spec": {
                    "initContainers": [
                        {
                            "image": conf.TF1_CPU_IMAGE,
                            "name": "simple-init-container",
                            "command": ["/bin/bash"],
                            "args": ["-c", "exit 1"],
                        }
                    ]
                }
            }
        }
    }

    _run_cmd_with_config_expecting_failure(
        cmd="sleep 3", expected_failure="exit code 1", config=config
    )

    config["environment"]["pod_spec"]["spec"]["initContainers"][0]["args"] = ["-c", "exit 0"]
    _run_cmd_with_config_expecting_success(cmd="sleep 3", config=config)


@pytest.mark.e2e_gpu
def test_k8_sidecars(using_k8s: bool) -> None:
    if not using_k8s:
        pytest.skip("only need to run test on kubernetes")

    base_config = {
        "environment": {
            "pod_spec": {
                "spec": {
                    "containers": [
                        {
                            "image": conf.TF1_CPU_IMAGE,
                            "name": "sidecar",
                            "command": ["/bin/bash"],
                        }
                    ]
                }
            }
        }
    }

    def set_arg(arg: str) -> Dict[str, Any]:
        new_config = copy.deepcopy(base_config)
        new_config["environment"]["pod_spec"]["spec"]["containers"][0]["args"] = ["-c", arg]
        return new_config

    configs = [set_arg("sleep 1; exit 1"), set_arg("sleep 99999999")]
    for config in configs:
        _run_cmd_with_config_expecting_failure(
            cmd="sleep 3; exit 1", expected_failure="exit code 1", config=config
        )

        _run_cmd_with_config_expecting_success(cmd="sleep 3", config=config)


@pytest.mark.e2e_gpu
@pytest.mark.parametrize("slots", [0, 1])
def test_k8_resource_limits(using_k8s: bool, slots: int) -> None:
    if not using_k8s:
        pytest.skip("only need to run test on kubernetes")

    config = {
        "environment": {
            "pod_spec": {
                "spec": {
                    "containers": [
                        {
                            "name": "determined-container",
                            "resources": {
                                "requests": {
                                    "cpu": 0.1,
                                    "memory": "1Gi",
                                },
                                "limits": {
                                    "cpu": 1,
                                    "memory": "1Gi",
                                },
                            },
                        }
                    ],
                }
            }
        },
        "resources": {
            "slots": slots,
        },
    }

    _run_cmd_with_config_expecting_success(cmd="sleep 3; echo hello", config=config)


@pytest.mark.e2e_cpu
def test_log_wait_timeout(tmp_path: Path, secrets: Dict[str, str]) -> None:
    # Start a subshell that prints after 5 and 20 seconds, then exit.
    cmd = 'sh -c "sleep 5; echo after 5; sleep 15; echo after 20" & echo main shell exiting'

    config = {"environment": {"environment_variables": ["DET_LOG_WAIT_TIME=10"]}}
    with tempfile.NamedTemporaryFile() as tf:
        with open(tf.name, "w") as f:
            yaml.dump(config, f)

        cli = ["det", "-m", conf.make_master_url(), "cmd", "run", "--config-file", tf.name, cmd]
        p = subprocess.run(cli, stdout=subprocess.PIPE, check=True)
        assert p.stdout is not None
        stdout = p.stdout.decode("utf8")

    # Logs should wait for the main process to die, plus 10 seconds, then shut down.
    # That should capture the "after 5" but not the "after 60".
    # By making the "after 20" occur before the default DET_LOG_WAIT_TIME of 30, we also are testing
    # that the escape hatch keeps working.
    assert "after 5" in stdout, stdout
    assert "after 20" not in stdout, stdout


@pytest.mark.parametrize("task_type", ["notebook", "command", "shell", "tensorboard"])
@pytest.mark.e2e_cpu
def test_log_argument(task_type: str) -> None:
    taskid = "28ad1623-dcf0-47d2-9faa-265aaa05b078"
    cmd: List[str] = ["det", "-m", conf.make_master_url(), task_type, "logs", taskid]
    p = subprocess.run(cmd, stderr=subprocess.PIPE, check=False)
    assert p.stderr is not None
    assert "not found" in p.stderr.decode("utf8"), p.stderr.decode("utf8")
