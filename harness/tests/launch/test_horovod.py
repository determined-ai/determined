import os
import time
from unittest import mock

import pytest

import determined.launch.horovod  # noqa: F401
from determined import constants, horovod, launch
from determined.common.api import certs
from tests.launch import test_util


def test_parse_args() -> None:
    positive_test_cases = {
        "--trial my_module:MyTrial": (
            [],
            ["python3", "-m", "determined.exec.harness", "my_module:MyTrial"],
            False,
        ),
        "script arg": ([], ["script", "arg"], False),
        "-- script arg": ([], ["script", "arg"], False),
        "h1 h2 -- script arg": (["h1", "h2"], ["script", "arg"], False),
        # The script is allowed to have conflicting args.
        "--autohorovod script": ([], ["script"], True),
        "script --autohorovod": ([], ["script", "--autohorovod"], False),
        # Scripts which require -- still work if the initial -- is present.
        "-- script -- arg": ([], ["script", "--", "arg"], False),
        "-- --autohorovod script -- arg": ([], ["script", "--", "arg"], True),
    }
    negative_test_cases = {
        "--trial my_module:MyTrial script": "extra arguments",
        "": "empty script",
        "--asdf 1 script ": "unrecognized arguments",
    }

    test_util.parse_args_check(positive_test_cases, negative_test_cases, launch.horovod.parse_args)


@pytest.mark.parametrize("autohorovod", [True, False])
@pytest.mark.parametrize("nnodes", [1, 4])
@pytest.mark.parametrize("nslots", [1, 4])
@mock.patch("subprocess.Popen")
@mock.patch("determined.get_cluster_info")
@mock.patch("determined.util.check_sshd")
@mock.patch("time.time")
def test_horovod_chief(
    mock_time: mock.MagicMock,
    mock_check_sshd: mock.MagicMock,
    mock_cluster_info: mock.MagicMock,
    mock_popen: mock.MagicMock,
    nslots: int,
    nnodes: int,
    autohorovod: bool,
) -> None:
    info = test_util.make_mock_cluster_info(
        ["0.0.0.{i}" for i in range(nnodes)], 0, num_slots=nslots
    )
    experiment_config = info.trial._config
    mock_cluster_info.return_value = info
    mock_start_time = time.time()
    mock_time.return_value = mock_start_time
    hvd_args = ["ds1", "ds2"]
    script = ["s1", "s2"]

    pid_server_cmd = launch.horovod.create_hvd_pid_server_cmd(
        info.allocation_id, len(info.slot_ids)
    )

    hvd_cmd = horovod.create_run_command(
        host_slot_counts=info.container_slot_counts,
        ip_addresses=info.container_addrs,
        inter_node_network_interface=info.trial._inter_node_network_interface,
        optimizations=experiment_config["optimizations"],
        debug=False,
        optional_args=hvd_args,
    )

    worker_wrapper_cmd = launch.horovod.create_worker_wrapper_cmd(info.allocation_id)

    launch_cmd = pid_server_cmd + hvd_cmd + worker_wrapper_cmd + script

    mock_proc = mock.MagicMock()
    mock_proc.wait.return_value = 99

    mock_popen.return_value = mock_proc

    with test_util.set_resources_id_env_var():
        assert launch.horovod.main(hvd_args, script, autohorovod) == 99

    if autohorovod and nnodes == 1 and nslots == 1:
        # Single-slot --autohorovod: we should have just called the script directly.
        mock_popen.assert_has_calls([mock.call(script, start_new_session=True)])
        mock_check_sshd.assert_not_called()
    else:
        # Multi-slot or non --autohorovod: expect a full horovodrun command.
        mock_cluster_info.assert_called_once()
        assert os.environ["DET_CHIEF_IP"] == info.container_addrs[0]
        assert os.environ["USE_HOROVOD"] == "1"

        mock_popen.assert_has_calls([mock.call(launch_cmd, start_new_session=True)])

        assert mock_check_sshd.call_count == len(info.container_addrs[1:])
        mock_check_sshd.assert_has_calls(
            [
                mock.call(addr, mock_start_time + 20, constants.DTRAIN_SSH_PORT)
                for addr in info.container_addrs[1:]
            ]
        )

        mock_proc.wait.assert_called_once()


@mock.patch("subprocess.Popen")
@mock.patch("determined.get_cluster_info")
@mock.patch("determined.common.api.post")
def test_sshd_worker(
    mock_api_post: mock.MagicMock,
    mock_cluster_info: mock.MagicMock,
    mock_popen: mock.MagicMock,
) -> None:
    info = test_util.make_mock_cluster_info(["0.0.0.0", "0.0.0.1"], 1, num_slots=1)
    mock_cluster_info.return_value = info
    hvd_args = ["ds1", "ds2"]
    script = ["s1", "s2"]

    pid_server_cmd, run_sshd_cmd = launch.horovod.create_sshd_worker_cmd(
        info.allocation_id,
        len(info.slot_ids),
    )

    launch_cmd = pid_server_cmd + run_sshd_cmd

    mock_proc = mock.MagicMock()
    mock_proc.wait.return_value = 99

    mock_popen.return_value = mock_proc

    with test_util.set_resources_id_env_var():
        assert launch.horovod.main(hvd_args, script, True) == 99

    mock_cluster_info.assert_called_once()
    assert os.environ["DET_CHIEF_IP"] == info.container_addrs[0]
    assert os.environ["USE_HOROVOD"] == "1"

    mock_popen.assert_has_calls([mock.call(launch_cmd, start_new_session=True)])

    mock_api_post.assert_has_calls(
        [
            mock.call(
                info.master_url,
                path=f"/api/v1/allocations/{info.allocation_id}/resources/resourcesId/daemon",
                cert=certs.cli_cert,
            )
        ]
    )

    mock_proc.wait.assert_called_once()
