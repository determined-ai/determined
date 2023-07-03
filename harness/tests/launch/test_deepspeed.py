import os
import time
from typing import Any, List
from unittest import mock

import pytest
from deepspeed.launcher.runner import DEEPSPEED_ENVIRONMENT_NAME

import determined.launch.deepspeed  # noqa: F401
from determined import constants, launch
from tests.launch import test_util


def test_parse_args() -> None:
    positive_test_cases = {
        "--trial my_module:MyTrial": [
            "python3",
            "-m",
            "determined.exec.harness",
            "my_module:MyTrial",
        ],
        "script arg": ["script", "arg"],
        # The script is allowed to have conflicting args.
        "script --trial": ["script", "--trial"],
        # Scripts which require -- still work.
        "script -- arg": ["script", "--", "arg"],
    }

    negative_test_cases = {
        "--trial my_module:MyTrial script": "extra arguments",
        "": "empty script",
        "--asdf 1 script ": "unrecognized arguments",
    }

    test_util.parse_args_check(
        positive_test_cases, negative_test_cases, launch.deepspeed.parse_args
    )


@mock.patch("subprocess.Popen")
@mock.patch("determined.get_cluster_info")
@mock.patch("determined.util.check_sshd")
@mock.patch("time.time")
def test_launch_multi_slot_chief(
    mock_time: mock.MagicMock,
    mock_check_sshd: mock.MagicMock,
    mock_cluster_info: mock.MagicMock,
    mock_subprocess: mock.MagicMock,
) -> None:
    cluster_info = test_util.make_mock_cluster_info(["0.0.0.0", "0.0.0.1"], 0, 4)
    mock_cluster_info.return_value = cluster_info
    mock_start_time = time.time()
    mock_time.return_value = mock_start_time
    script = ["s1", "s2"]
    sshd_cmd = launch.deepspeed.create_sshd_cmd()
    pid_server_cmd = launch.deepspeed.create_pid_server_cmd(
        cluster_info.allocation_id, len(cluster_info.slot_ids)
    )
    deepspeed_cmd = launch.deepspeed.create_run_command(
        cluster_info.container_addrs[0],
        launch.deepspeed.get_hostfile_path(
            multi_machine=True, allocation_id=cluster_info.allocation_id
        ),
    )
    pid_client_cmd = launch.deepspeed.create_pid_client_cmd(cluster_info.allocation_id)
    log_redirect_cmd = launch.deepspeed.create_log_redirect_cmd()

    launch_cmd = pid_server_cmd + deepspeed_cmd + pid_client_cmd + log_redirect_cmd + script

    sshd_proc_mock = mock.MagicMock()
    launch_proc_mock = mock.MagicMock()

    def mock_process(cmd: List[str], *args: Any, **kwargs: Any) -> Any:
        if cmd == sshd_cmd:
            return sshd_proc_mock(*args, **kwargs)
        if cmd == launch_cmd:
            return launch_proc_mock(*args, **kwargs)
        return None

    mock_subprocess.side_effect = mock_process

    with test_util.set_resources_id_env_var():
        launch.deepspeed.main(script)

    mock_cluster_info.assert_called_once()
    assert os.environ["DET_CHIEF_IP"] == cluster_info.container_addrs[0]
    assert os.environ["USE_DEEPSPEED"] == "1"
    assert os.environ["PDSH_SSH_ARGS"] == (
        "-o PasswordAuthentication=no -o StrictHostKeyChecking=no "
        f"-p {constants.DTRAIN_SSH_PORT} -2 -a -x %h"
    )

    mock_subprocess.assert_has_calls([mock.call(sshd_cmd), mock.call(launch_cmd)])

    assert mock_check_sshd.call_count == len(cluster_info.container_addrs)
    mock_check_sshd.assert_has_calls(
        [
            mock.call(addr, mock_start_time + 20, constants.DTRAIN_SSH_PORT)
            for addr in cluster_info.container_addrs
        ]
    )

    launch_proc_mock().wait.assert_called_once()

    sshd_proc_mock().kill.assert_called_once()
    sshd_proc_mock().wait.assert_called_once()

    # Cleanup deepspeed environment file created in launch.deepspeed.main
    deepspeed_env_path = os.path.join(os.getcwd(), DEEPSPEED_ENVIRONMENT_NAME)
    if os.path.isfile(deepspeed_env_path):
        os.remove(deepspeed_env_path)


@mock.patch("subprocess.Popen")
@mock.patch("determined.get_cluster_info")
@mock.patch("determined.util.check_sshd")
@mock.patch("time.time")
def test_launch_multi_slot_fail(
    mock_time: mock.MagicMock,
    mock_check_sshd: mock.MagicMock,
    mock_cluster_info: mock.MagicMock,
    mock_subprocess: mock.MagicMock,
) -> None:
    cluster_info = test_util.make_mock_cluster_info(["0.0.0.0", "0.0.0.1"], 0, 4)
    mock_cluster_info.return_value = cluster_info
    mock_start_time = time.time()
    mock_time.return_value = mock_start_time
    mock_check_sshd.side_effect = ValueError("no sshd greeting")

    script = ["s1", "s2"]
    sshd_cmd = launch.deepspeed.create_sshd_cmd()
    pid_server_cmd = launch.deepspeed.create_pid_server_cmd(
        cluster_info.allocation_id, len(cluster_info.slot_ids)
    )
    deepspeed_cmd = launch.deepspeed.create_run_command(
        cluster_info.container_addrs[0],
        launch.deepspeed.get_hostfile_path(
            multi_machine=True, allocation_id=cluster_info.allocation_id
        ),
    )
    pid_client_cmd = launch.deepspeed.create_pid_client_cmd(cluster_info.allocation_id)
    log_redirect_cmd = launch.deepspeed.create_log_redirect_cmd()

    launch_cmd = pid_server_cmd + deepspeed_cmd + pid_client_cmd + log_redirect_cmd + script

    sshd_proc_mock = mock.MagicMock()
    launch_proc_mock = mock.MagicMock()

    def mock_process(cmd: List[str], *args: Any, **kwargs: Any) -> Any:
        if cmd == sshd_cmd:
            return sshd_proc_mock(*args, **kwargs)
        if cmd == launch_cmd:
            return launch_proc_mock(*args, **kwargs)
        return None

    mock_subprocess.side_effect = mock_process

    with test_util.set_resources_id_env_var():
        with pytest.raises(ValueError, match="no sshd greeting"):
            launch.deepspeed.main(script)

    mock_cluster_info.assert_called_once()
    assert os.environ["DET_CHIEF_IP"] == cluster_info.container_addrs[0]
    assert os.environ["USE_DEEPSPEED"] == "1"
    assert os.environ["PDSH_SSH_ARGS"] == (
        "-o PasswordAuthentication=no -o StrictHostKeyChecking=no "
        f"-p {constants.DTRAIN_SSH_PORT} -2 -a -x %h"
    )

    mock_subprocess.assert_called_once_with(sshd_cmd)

    mock_check_sshd.assert_called_once_with(
        cluster_info.container_addrs[0], mock_start_time + 20, constants.DTRAIN_SSH_PORT
    )

    sshd_proc_mock().kill.assert_called_once()
    sshd_proc_mock().wait.assert_called_once()

    # Cleanup deepspeed environment file created in launch.deepspeed.main
    deepspeed_env_path = os.path.join(os.getcwd(), DEEPSPEED_ENVIRONMENT_NAME)
    if os.path.isfile(deepspeed_env_path):
        os.remove(deepspeed_env_path)


@mock.patch("subprocess.Popen")
@mock.patch("determined.get_cluster_info")
def test_launch_one_slot(
    mock_cluster_info: mock.MagicMock, mock_subprocess: mock.MagicMock
) -> None:
    cluster_info = test_util.make_mock_cluster_info(["0.0.0.0"], 0, 4)
    mock_cluster_info.return_value = cluster_info
    script = ["s1", "s2"]
    pid_server_cmd = launch.deepspeed.create_pid_server_cmd(
        cluster_info.allocation_id, len(cluster_info.slot_ids)
    )
    deepspeed_cmd = launch.deepspeed.create_run_command(
        "localhost",
        launch.deepspeed.get_hostfile_path(
            multi_machine=False, allocation_id=cluster_info.allocation_id
        ),
    )
    pid_client_cmd = launch.deepspeed.create_pid_client_cmd(cluster_info.allocation_id)
    log_redirect_cmd = launch.deepspeed.create_log_redirect_cmd()
    launch_cmd = pid_server_cmd + deepspeed_cmd + pid_client_cmd + log_redirect_cmd + script

    with test_util.set_resources_id_env_var():
        launch.deepspeed.main(script)

    mock_cluster_info.assert_called_once()
    assert os.environ["DET_CHIEF_IP"] == cluster_info.container_addrs[0]
    assert os.environ["USE_DEEPSPEED"] == "1"

    mock_subprocess.assert_called_once_with(launch_cmd)


@mock.patch("subprocess.Popen")
@mock.patch("determined.get_cluster_info")
def test_launch_fail(mock_cluster_info: mock.MagicMock, mock_subprocess: mock.MagicMock) -> None:
    cluster_info = test_util.make_mock_cluster_info(["0.0.0.0"], 0, 4)
    mock_cluster_info.return_value = cluster_info
    mock_subprocess.return_value.wait.return_value = 1
    script = ["s1", "s2"]
    pid_server_cmd = launch.deepspeed.create_pid_server_cmd(
        cluster_info.allocation_id, len(cluster_info.slot_ids)
    )
    deepspeed_cmd = launch.deepspeed.create_run_command(
        "localhost",
        launch.deepspeed.get_hostfile_path(
            multi_machine=False, allocation_id=cluster_info.allocation_id
        ),
    )
    pid_client_cmd = launch.deepspeed.create_pid_client_cmd(cluster_info.allocation_id)
    log_redirect_cmd = launch.deepspeed.create_log_redirect_cmd()
    launch_cmd = pid_server_cmd + deepspeed_cmd + pid_client_cmd + log_redirect_cmd + script

    with test_util.set_resources_id_env_var():
        assert launch.deepspeed.main(script) == 1

    mock_cluster_info.assert_called_once()
    assert os.environ["DET_CHIEF_IP"] == cluster_info.container_addrs[0]
    assert os.environ["USE_DEEPSPEED"] == "1"

    mock_subprocess.assert_called_once_with(launch_cmd)


@mock.patch("subprocess.Popen")
@mock.patch("determined.get_cluster_info")
@mock.patch("determined.common.api.post")
def test_launch_worker(
    mock_api: mock.MagicMock, mock_cluster_info: mock.MagicMock, mock_subprocess: mock.MagicMock
) -> None:
    cluster_info = test_util.make_mock_cluster_info(["0.0.0.0", "0.0.0.1"], 1, 4)
    mock_cluster_info.return_value = cluster_info
    with test_util.set_resources_id_env_var():
        launch.deepspeed.main(["script"])

    mock_cluster_info.assert_called_once()
    assert os.environ["DET_CHIEF_IP"] == cluster_info.container_addrs[0]

    mock_api.assert_called_once()

    pid_server_cmd = launch.deepspeed.create_pid_server_cmd(
        cluster_info.allocation_id, len(cluster_info.slot_ids)
    )
    sshd_cmd = launch.deepspeed.create_sshd_cmd()

    expected_cmd = pid_server_cmd + sshd_cmd
    mock_subprocess.assert_called_once_with(expected_cmd)


def test_filter_env_vars() -> None:
    env_in = {
        "BASH_FUNC_xyz": "drop",
        "KEEP_BASH_FUNC": "keep",
        "OLDPWD": "drop",
        "HOSTNAME": "drop",
        "CUDA_VISIBLE_DEVICES": "drop",
        "APPTAINER_CUDA_VISIBLE_DEVICES": "drop",
        "SLURM_PROCID": "drop",
        "DET_SLOT_IDS": "drop",
        "DET_AGENT_ID": "drop",
        "RANDOM_USER_VAR": "keep",
    }
    env_out = launch.deepspeed.filter_env_vars(env_in)
    env_exp = {
        "KEEP_BASH_FUNC": "keep",
        "RANDOM_USER_VAR": "keep",
    }
    assert env_out == env_exp
