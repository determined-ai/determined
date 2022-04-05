import contextlib
import io
import os
import sys
import time
from typing import Any, Iterator, List
from unittest import mock

import pytest
from deepspeed.launcher.runner import DEEPSPEED_ENVIRONMENT_NAME

import determined as det
import determined.launch.deepspeed
from determined import constants, launch


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
    for args, exp in positive_test_cases.items():
        assert exp == launch.deepspeed.parse_args(args.split()), f"test case failed, args = {args}"

    negative_test_cases = {
        "--trial my_module:MyTrial script": "extra arguments",
        "": "empty script",
        "--asdf 1 script ": "unrecognized arguments",
    }

    for args, msg in negative_test_cases.items():
        old = sys.stderr
        fake = io.StringIO()
        sys.stderr = fake
        try:
            try:
                launch.deepspeed.parse_args(args.split())
            except SystemExit:
                # This is expected.
                err = fake.getvalue()
                assert msg in err, f"test case failed, args='{args}' msg='{msg}', stderr='{err}'"
                continue
            raise AssertionError(f"negative test case did not fail: args='{args}'")
        finally:
            sys.stderr = old


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
    cluster_info = make_mock_cluster_info(["0.0.0.0", "0.0.0.1"], 0)
    mock_cluster_info.return_value = cluster_info
    mock_start_time = time.time()
    mock_time.return_value = mock_start_time
    script = ["s1", "s2"]
    sshd_cmd = launch.deepspeed.create_sshd_cmd()
    pid_server_cmd = launch.deepspeed.create_pid_server_cmd(
        cluster_info.allocation_id, len(cluster_info.slot_ids)
    )
    deepspeed_cmd = launch.deepspeed.create_run_command(
        cluster_info.container_addrs[0], launch.deepspeed.hostfile_path
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

    with set_resources_id_env_var():
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
    cluster_info = make_mock_cluster_info(["0.0.0.0", "0.0.0.1"], 0)
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
        cluster_info.container_addrs[0], launch.deepspeed.hostfile_path
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

    with set_resources_id_env_var():
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
    cluster_info = make_mock_cluster_info(["0.0.0.0"], 0)
    mock_cluster_info.return_value = cluster_info
    script = ["s1", "s2"]
    pid_server_cmd = launch.deepspeed.create_pid_server_cmd(
        cluster_info.allocation_id, len(cluster_info.slot_ids)
    )
    deepspeed_cmd = launch.deepspeed.create_run_command("localhost", launch.deepspeed.hostfile_path)
    pid_client_cmd = launch.deepspeed.create_pid_client_cmd(cluster_info.allocation_id)
    log_redirect_cmd = launch.deepspeed.create_log_redirect_cmd()
    launch_cmd = pid_server_cmd + deepspeed_cmd + pid_client_cmd + log_redirect_cmd + script

    with set_resources_id_env_var():
        launch.deepspeed.main(script)

    mock_cluster_info.assert_called_once()
    assert os.environ["DET_CHIEF_IP"] == cluster_info.container_addrs[0]
    assert os.environ["USE_DEEPSPEED"] == "1"

    mock_subprocess.assert_called_once_with(launch_cmd)


@mock.patch("subprocess.Popen")
@mock.patch("determined.get_cluster_info")
def test_launch_fail(mock_cluster_info: mock.MagicMock, mock_subprocess: mock.MagicMock) -> None:
    cluster_info = make_mock_cluster_info(["0.0.0.0"], 0)
    mock_cluster_info.return_value = cluster_info
    mock_subprocess.return_value.wait.return_value = 1
    script = ["s1", "s2"]
    pid_server_cmd = launch.deepspeed.create_pid_server_cmd(
        cluster_info.allocation_id, len(cluster_info.slot_ids)
    )
    deepspeed_cmd = launch.deepspeed.create_run_command("localhost", launch.deepspeed.hostfile_path)
    pid_client_cmd = launch.deepspeed.create_pid_client_cmd(cluster_info.allocation_id)
    log_redirect_cmd = launch.deepspeed.create_log_redirect_cmd()
    launch_cmd = pid_server_cmd + deepspeed_cmd + pid_client_cmd + log_redirect_cmd + script

    with set_resources_id_env_var():
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
    cluster_info = make_mock_cluster_info(["0.0.0.0", "0.0.0.1"], 1)
    mock_cluster_info.return_value = cluster_info
    with set_resources_id_env_var():
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


def make_mock_cluster_info(container_addrs: List[str], container_rank: int) -> det.ClusterInfo:
    rendezvous_info_mock = det.RendezvousInfo(
        container_addrs=container_addrs, container_rank=container_rank
    )
    cluster_info_mock = det.ClusterInfo(
        master_url="localhost",
        cluster_id="clusterId",
        agent_id="agentId",
        slot_ids=[0, 1, 2, 3],
        task_id="taskId",
        allocation_id="allocationId",
        session_token="sessionToken",
        task_type="TRIAL",
        rendezvous_info=rendezvous_info_mock,
    )
    return cluster_info_mock


@contextlib.contextmanager
def set_resources_id_env_var() -> Iterator[None]:
    try:
        os.environ["DET_RESOURCES_ID"] = "containerId"
        yield
    finally:
        del os.environ["DET_RESOURCES_ID"]
