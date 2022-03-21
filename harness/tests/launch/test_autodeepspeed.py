import contextlib
import os
import time
import unittest.mock as mock
from typing import Any, Iterator, List

import pytest

from determined import ClusterInfo, RendezvousInfo, constants
from determined.launch import autodeepspeed


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
    train_entrypoint = "model_def:TrialClass"
    sshd_cmd = [
        "/usr/sbin/sshd",
        "-p",
        str(constants.DTRAIN_SSH_PORT),
        "-f",
        "/run/determined/ssh/sshd_config",
        "-D",
    ]

    pid_server_cmd = [
        "python3",
        "-m",
        "determined.exec.pid_server",
        "--on-fail",
        "SIGTERM",
        "--on-exit",
        "SIGTERM",
        "--grace-period",
        "5",
        f"/tmp/pid_server-{cluster_info.allocation_id}",
        str(len(cluster_info.slot_ids)),
        "--",
    ]

    deepspeed_cmd = [
        "deepspeed",
        "-H",
        autodeepspeed.hostfile_path,
        "--master_addr",
        cluster_info.container_addrs[0],
        "--no_python",
        "--no_local_rank",
        "--",
    ]

    pid_client_cmd = [
        "python3",
        "-m",
        "determined.exec.pid_client",
        f"/tmp/pid_server-{cluster_info.allocation_id}",
        "--",
    ]

    log_redirect_cmd = [
        "python3",
        "-m",
        "determined.exec.worker_process_wrapper",
        "RANK",
        "--",
    ]

    harness_cmd = [
        "python3",
        "-m",
        "determined.exec.harness",
        "--train-entrypoint",
        train_entrypoint,
    ]

    launch_cmd = pid_server_cmd + deepspeed_cmd + pid_client_cmd + log_redirect_cmd + harness_cmd

    sshd_proc_mock = mock.MagicMock()
    launch_proc_mock = mock.MagicMock()

    def mock_process(cmd: List[str], *args: Any, **kwargs: Any) -> Any:
        if cmd == sshd_cmd:
            return sshd_proc_mock(*args, **kwargs)
        if cmd == launch_cmd:
            return launch_proc_mock(*args, **kwargs)
        return None

    mock_subprocess.side_effect = mock_process

    with set_container_id_env_var():
        autodeepspeed.main(train_entrypoint)

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

    train_entrypoint = "model_def:TrialClass"
    sshd_cmd = [
        "/usr/sbin/sshd",
        "-p",
        str(constants.DTRAIN_SSH_PORT),
        "-f",
        "/run/determined/ssh/sshd_config",
        "-D",
    ]

    pid_server_cmd = [
        "python3",
        "-m",
        "determined.exec.pid_server",
        "--on-fail",
        "SIGTERM",
        "--on-exit",
        "SIGTERM",
        "--grace-period",
        "5",
        f"/tmp/pid_server-{cluster_info.allocation_id}",
        str(len(cluster_info.slot_ids)),
        "--",
    ]

    deepspeed_cmd = [
        "deepspeed",
        "-H",
        autodeepspeed.hostfile_path,
        "--master_addr",
        cluster_info.container_addrs[0],
        "--no_python",
        "--no_local_rank",
        "--",
    ]

    pid_client_cmd = [
        "python3",
        "-m",
        "determined.exec.pid_client",
        f"/tmp/pid_server-{cluster_info.allocation_id}",
        "--",
    ]

    log_redirect_cmd = [
        "python3",
        "-m",
        "determined.exec.worker_process_wrapper",
        "RANK",
        "--",
    ]

    harness_cmd = [
        "python3",
        "-m",
        "determined.exec.harness",
        "--train-entrypoint",
        train_entrypoint,
    ]

    launch_cmd = pid_server_cmd + deepspeed_cmd + pid_client_cmd + log_redirect_cmd + harness_cmd

    sshd_proc_mock = mock.MagicMock()
    launch_proc_mock = mock.MagicMock()

    def mock_process(cmd: List[str], *args: Any, **kwargs: Any) -> Any:
        if cmd == sshd_cmd:
            return sshd_proc_mock(*args, **kwargs)
        if cmd == launch_cmd:
            return launch_proc_mock(*args, **kwargs)
        return None

    mock_subprocess.side_effect = mock_process

    with set_container_id_env_var():
        with pytest.raises(ValueError):
            autodeepspeed.main(train_entrypoint)

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


@mock.patch("subprocess.Popen")
@mock.patch("determined.get_cluster_info")
def test_launch_one_slot(
    mock_cluster_info: mock.MagicMock, mock_subprocess: mock.MagicMock
) -> None:
    cluster_info = make_mock_cluster_info(["0.0.0.0"], 0)
    mock_cluster_info.return_value = cluster_info
    train_entrypoint = "model_def:TrialClass"

    launch_cmd = [
        "python3",
        "-m",
        "determined.exec.pid_server",
        "--on-fail",
        "SIGTERM",
        "--on-exit",
        "SIGTERM",
        "--grace-period",
        "5",
        f"/tmp/pid_server-{cluster_info.allocation_id}",
        str(len(cluster_info.slot_ids)),
        "--",
        "deepspeed",
        "-H",
        autodeepspeed.hostfile_path,
        "--master_addr",
        "localhost",
        "--no_python",
        "--no_local_rank",
        "--",
        "python3",
        "-m",
        "determined.exec.pid_client",
        f"/tmp/pid_server-{cluster_info.allocation_id}",
        "--",
        "python3",
        "-m",
        "determined.exec.worker_process_wrapper",
        "RANK",
        "--",
        "python3",
        "-m",
        "determined.exec.harness",
        "--train-entrypoint",
        train_entrypoint,
    ]

    with set_container_id_env_var():
        autodeepspeed.main(train_entrypoint)

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
    with set_container_id_env_var():
        autodeepspeed.main("model_def:TrialClass")

    mock_cluster_info.assert_called_once()
    assert os.environ["DET_CHIEF_IP"] == cluster_info.container_addrs[0]

    mock_api.assert_called_once()
    expected_cmd = [
        "python3",
        "-m",
        "determined.exec.pid_server",
        "--on-fail",
        "SIGTERM",
        "--on-exit",
        "SIGTERM",
        "--grace-period",
        "3",
        f"/tmp/pid_server-{cluster_info.allocation_id}",
        str(len(cluster_info.slot_ids)),
        "--",
        "/usr/sbin/sshd",
        "-p",
        str(constants.DTRAIN_SSH_PORT),
        "-f",
        "/run/determined/ssh/sshd_config",
        "-D",
    ]
    mock_subprocess.assert_called_once_with(expected_cmd)


def make_mock_cluster_info(container_addrs: List[str], container_rank: int) -> ClusterInfo:
    rendezvous_info_mock = RendezvousInfo(
        container_addrs=container_addrs, container_rank=container_rank
    )
    cluster_info_mock = ClusterInfo(
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
def set_container_id_env_var() -> Iterator[None]:
    try:
        os.environ["DET_CONTAINER_ID"] = "containerId"
        yield
    finally:
        del os.environ["DET_CONTAINER_ID"]
