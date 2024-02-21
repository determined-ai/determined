import contextlib
import copy
import pathlib
import subprocess
import tempfile
import textwrap
from typing import Any, Dict, List, Optional

import docker
import docker.errors
import pytest

from determined.common import api, util
from tests import api_utils
from tests import command as cmd
from tests import config as conf
from tests import detproc, filetree


def _run_cmd(
    sess: api.Session,
    cmd: List[str],
    *,
    expect_success: bool,
    config: Optional[Dict[str, Any]] = None,
    context: Optional[str] = None,
) -> None:
    """Always expect `det cmd run` to succeed, but the command itself might fail."""

    with contextlib.ExitStack() as es:
        det_cmd = ["det", "cmd", "run"]
        if config is not None:
            tf = es.enter_context(tempfile.NamedTemporaryFile())
            with open(tf.name, "w") as f:
                util.yaml_safe_dump(config, f)
            det_cmd += ["--config-file", tf.name]
        if context:
            det_cmd += ["-c", context]

        output = detproc.check_output(sess, det_cmd + cmd)

        if expect_success:
            assert "resources exited successfully" in output.lower(), output
        else:
            assert "resources failed with non-zero exit code" in output.lower(), output


@pytest.mark.slow
@pytest.mark.e2e_cpu
@pytest.mark.e2e_slurm
@pytest.mark.e2e_pbs
def test_basic_workflows(tmp_path: pathlib.Path) -> None:
    sess = api_utils.user_session()
    with filetree.FileTree(tmp_path, {"hello.py": "print('hello world')"}) as tree:
        _run_cmd(sess, ["python", "hello.py"], context=str(tree), expect_success=True)

    with filetree.FileTree(tmp_path, {"hello.py": "print('hello world')"}) as tree:
        link = tree.joinpath("hello-link.py")
        link.symlink_to(tree.joinpath("hello.py"))
        _run_cmd(sess, ["python", "hello-link.py"], context=str(tree), expect_success=True)

    detproc.check_error(
        sess,
        ["det", "cmd", "run", "--context", "non-existent-path", "true"],
        "non-existent-path' doesn't exist",
    )


@pytest.mark.slow
@pytest.mark.e2e_cpu
def test_large_uploads(tmp_path: pathlib.Path) -> None:
    sess = api_utils.user_session()

    with filetree.FileTree(tmp_path, {"hello.py": "print('hello world')"}) as tree:
        large = tree.joinpath("large-file.bin")
        with large.open("w") as f:
            f.seek(1024 * 1024 * 120)
            f.write("\0")

        # 120MB is too big.
        detproc.check_error(
            sess, ["det", "cmd", "run", "-c", str(tree), "true"], "maximum allowed size"
        )

        # .detignore makes it ok though.
        with tree.joinpath(".detignore").open("w") as f:
            f.write("*.bin\n")
        _run_cmd(sess, ["python", "hello.py"], context=str(tree), expect_success=True)


# TODO(DET-9859) we could move this test to nightly or even per release to save CI cost.
# It takes around 15 seconds.
@pytest.mark.e2e_k8s
def test_context_directory_larger_than_config_map_k8s(tmp_path: pathlib.Path) -> None:
    sess = api_utils.user_session()
    with filetree.FileTree(tmp_path, {"hello.py": "print('hello world')"}) as tree:
        large = tree.joinpath("large-file.bin")
        with large.open("w") as f:
            f.seek(1024 * 1024 * 10)
            f.write("\0")

        _run_cmd(sess, ["python", "hello.py"], context=str(tree), expect_success=True)


@pytest.mark.slow
@pytest.mark.e2e_cpu
def test_configs(tmp_path: pathlib.Path) -> None:
    sess = api_utils.user_session()
    config = {"environment": {"environment_variables": ["TEST=TEST"]}}
    _run_cmd(sess, ["env | grep -q TEST=TEST"], config=config, expect_success=True)


@pytest.mark.slow
@pytest.mark.e2e_cpu
@pytest.mark.e2e_slurm
@pytest.mark.e2e_pbs
def test_environment_variables_command() -> None:
    sess = api_utils.user_session()
    config_str = "environment.environment_variables='THISISTRUE=true','WONTCAUSEPANIC'"
    _run_cmd(sess, ["--config", config_str, "env | grep -q THISISTRUE=true"], expect_success=True)


@pytest.mark.parametrize("actual,expected", [("24576", "24"), ("1.5g", "1572864")])
@pytest.mark.parametrize("use_config_file", [True, False])
@pytest.mark.slow
@pytest.mark.e2e_cpu
def test_shm_size_command(
    tmp_path: pathlib.Path, actual: str, expected: str, use_config_file: bool
) -> None:
    sess = api_utils.user_session()
    script = textwrap.dedent(
        rf"""
        set -e
        set -o pipefail
        df /dev/shm | tail -1 | test "$(awk '{{print $2}}')" = '{expected}'
        """
    )
    if use_config_file:
        config = {"resources": {"shm_size": actual}}
        _run_cmd(sess, ["bash", "-c", script], config=config, expect_success=True)
    else:
        config_str = f"resources.shm_size={actual}"
        _run_cmd(sess, ["--config", config_str, "bash", "-c", script], expect_success=True)


@pytest.mark.slow
@pytest.mark.e2e_cpu
def test_absolute_bind_mount(tmp_path: pathlib.Path) -> None:
    sess = api_utils.user_session()
    config = {"bind_mounts": [{"host_path": "/bin", "container_path": "/foo-bar1"}]}
    _run_cmd(
        sess,
        ["--volume", "/bin:/foo-bar2", "ls", "/foo-bar1", "/foo-bar2"],
        config=config,
        expect_success=True,
    )


@pytest.mark.slow
@pytest.mark.e2e_cpu
def test_relative_bind_mount(tmp_path: pathlib.Path) -> None:
    sess = api_utils.user_session()
    config = {"bind_mounts": [{"host_path": "/bin", "container_path": "foo-bar1"}]}
    _run_cmd(
        sess,
        ["--volume", "/bin:foo-bar2", "ls", "foo-bar1", "foo-bar2"],
        config=config,
        expect_success=True,
    )


@pytest.mark.slow
@pytest.mark.e2e_cpu
@pytest.mark.e2e_slurm
@pytest.mark.e2e_pbs
def test_cmd_kill() -> None:
    """Start a command, extract its task ID, and then kill it."""
    sess = api_utils.user_session()

    with cmd.interactive_command(
        sess, ["command", "run", "echo hello world; sleep infinity"]
    ) as command:
        assert command.task_id is not None
        for line in command.stdout:
            if "hello world" in line:
                # For HPC job, dispatcher does the polling of the job state happens
                # every 10 seconds. For example, it is very likely the current job state is
                # STATE_PULLING when job is actually running on HPC. So instead of checking
                # for STATE_RUNNING, we check for other active states as well.
                assert cmd.get_num_active_commands(sess) == 1
                break


@pytest.mark.slow
@pytest.mark.e2e_cpu
def test_image_pull_after_remove() -> None:
    """
    Remove pulled image and verify that it will be pulled again with auth.
    """
    sess = api_utils.user_session()
    client = docker.from_env()
    try:
        client.images.remove("python:3.8.16")
    except docker.errors.ImageNotFound:
        pass

    _run_cmd(sess, ["--config", "environment.image=python:3.8.16", "true"], expect_success=True)


@pytest.mark.e2e_cpu
def test_outrageous_command_rejected() -> None:
    sess = api_utils.user_session()
    # Specify an outrageous number of slots to be sure that it can't be scheduled.
    detproc.check_error(
        sess,
        [
            "det",
            "-m",
            conf.make_master_url(),
            "cmd",
            "run",
            "--config",
            "resources.slots=10485",
            "sleep infinity",
        ],
        "request unfulfillable",
    )


@pytest.mark.e2e_gpu
@pytest.mark.parametrize("sidecar", [True, False])
@api_utils.skipif_not_k8s()
def test_k8s_mount(sidecar: bool) -> None:
    sess = api_utils.user_session()

    mount_path = "/ci/"

    output = detproc.check_output(
        sess,
        ["det", "cmd", "run", "touch", mount_path],
    )
    assert "No such file or directory" in output, output

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

    _run_cmd(sess, ["touch", mount_path], config=config, expect_success=True)


@pytest.mark.e2e_gpu
@api_utils.skipif_not_k8s()
def test_k8s_init_containers() -> None:
    sess = api_utils.user_session()

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
    _run_cmd(sess, ["echo", "hi"], config=config, expect_success=False)

    config["environment"]["pod_spec"]["spec"]["initContainers"][0]["args"] = ["-c", "exit 0"]
    _run_cmd(sess, ["echo", "hi"], config=config, expect_success=True)


@pytest.mark.e2e_gpu
@api_utils.skipif_not_k8s()
def test_k8s_sidecars() -> None:
    sess = api_utils.user_session()

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

    # Sidecar failure should not affect command failure.
    configs = [set_arg("sleep 1; exit 1"), set_arg("sleep 99999999")]
    for config in configs:
        _run_cmd(sess, ["false"], config=config, expect_success=False)
        _run_cmd(sess, ["sleep", "3"], config=config, expect_success=True)


@pytest.mark.e2e_gpu
@pytest.mark.parametrize("slots", [0, 1])
@api_utils.skipif_not_k8s()
def test_k8s_resource_limits(slots: int) -> None:
    sess = api_utils.user_session()

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

    _run_cmd(sess, ["true"], config=config, expect_success=True)


@pytest.mark.e2e_cpu
@pytest.mark.e2e_slurm
@pytest.mark.e2e_pbs
def test_log_wait_timeout(tmp_path: pathlib.Path, secrets: Dict[str, str]) -> None:
    sess = api_utils.user_session()

    # Start a subshell that prints after 5 and 20 seconds, then exit.
    cmd = 'sh -c "sleep 5; echo after 5; sleep 15; echo after 20" & echo main shell exiting'

    config = {"environment": {"environment_variables": ["DET_LOG_WAIT_TIME=10"]}}
    with tempfile.NamedTemporaryFile() as tf:
        with open(tf.name, "w") as f:
            util.yaml_safe_dump(config, f)

        cli = ["det", "cmd", "run", "--config-file", tf.name, cmd]
        p = detproc.run(sess, cli, stdout=subprocess.PIPE, check=True)
        assert p.stdout is not None
        stdout = p.stdout.decode("utf8")

    # Logs should wait for the main process to die, plus 10 seconds, then shut down.
    # That should capture the "after 5" but not the "after 20".
    # By making the "after 20" occur before the default DET_LOG_WAIT_TIME of 30, we also are testing
    # that the escape hatch keeps working.
    assert "after 5" in stdout, stdout
    assert "after 20" not in stdout, stdout


@pytest.mark.parametrize("task_type", ["notebook", "command", "shell", "tensorboard"])
@pytest.mark.e2e_cpu
def test_log_argument(task_type: str) -> None:
    sess = api_utils.user_session()
    taskid = "28ad1623-dcf0-47d2-9faa-265aaa05b078"
    cmd: List[str] = ["det", task_type, "logs", taskid]
    detproc.check_error(sess, cmd, "not found")
