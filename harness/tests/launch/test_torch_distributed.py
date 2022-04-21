import os

from tests.launch import test_util
import determined.launch.torch_distributed
from determined import constants, launch
from unittest import mock


def test_parse_args() -> None:
    positive_test_cases = {
        "--trial my_module:MyTrial": (
            [],
            ["python3", "-m", "determined.exec.harness", "my_module:MyTrial"]
        ),
        "script arg": ([], ["script", "arg"]),
        "-- script -- arg": ([], ["script", "--", "arg"]),
        "override1 override2 -- script arg": (["override1", "override2"], ["script", "arg"])
    }

    negative_test_cases = {
        "--trial my_module:MyTrial script": "extra arguments",
        "": "empty script",
        "--asdf 1 script ": "unrecognized arguments",
    }
    test_util.test_parse_args(positive_test_cases, negative_test_cases, launch.torch_distributed.parse_args)


@mock.patch("subprocess.Popen")
@mock.patch("determined.get_cluster_info")
def test_launch_single_slot(
        mock_cluster_info: mock.MagicMock,
        mock_subprocess: mock.MagicMock,
) -> None:
    cluster_info = test_util.make_mock_cluster_info(["0.0.0.0"], 0, 1)
    mock_cluster_info.return_value = cluster_info
    script = ["python3", "-m", "determined.exec.harness", "my_module:MyTrial"]
    override_args = ["--max_restarts", "1"]

    with test_util.set_resources_id_env_var():
        launch.torch_distributed.main(override_args, script)

    mock_subprocess.assert_called_once_with(script)

    assert os.environ["USE_TORCH_DISTRIBUTED"] == "True"


@mock.patch("subprocess.Popen")
@mock.patch("determined.get_cluster_info")
def test_launch_distributed(
        mock_cluster_info: mock.MagicMock,
        mock_subprocess: mock.MagicMock,
) -> None:
    cluster_info = test_util.make_mock_cluster_info(["0.0.0.0", "0.0.0.1"], 0, 2)
    mock_cluster_info.return_value = cluster_info
    script = ["python3", "-m", "determined.exec.harness", "my_module:MyTrial"]
    override_args = ["--max_restarts", "1"]

    mock_success_code = 99
    mock_proc = mock.MagicMock()
    mock_proc.wait.return_value = mock_success_code

    mock_subprocess.return_value = mock_proc

    with test_util.set_resources_id_env_var():
        assert launch.torch_distributed.main(override_args, script) == mock_success_code

    launch_cmd = launch.torch_distributed.create_launch_cmd(len(cluster_info.container_addrs),
                                                            len(cluster_info.slot_ids),
                                                            cluster_info.container_rank,
                                                            cluster_info.container_addrs[0],
                                                            override_args)

    launch_cmd += launch.torch_distributed.create_log_redirect_cmd()
    launch_cmd += script

    mock_subprocess.assert_called_once_with(launch_cmd)

    assert os.environ["USE_TORCH_DISTRIBUTED"] == "True"
    assert os.environ["DET_CHIEF_IP"] == cluster_info.container_addrs[0]
