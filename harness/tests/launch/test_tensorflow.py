import json
import os
from unittest import mock

import determined.launch.tensorflow  # noqa: F401
from determined import launch
from tests.launch import test_util


def test_parse_args() -> None:
    positive_test_cases = {
        "script arg": (29400, ["script", "arg"]),
        "-- script -- arg": (29400, ["script", "--", "arg"]),
        "-- script --port 1": (29400, ["script", "--port", "1"]),
        "--port 1 -- script arg": (1, ["script", "arg"]),
        "script --port 1": (29400, ["script", "--port", "1"]),
    }

    negative_test_cases = {
        "": "empty script",
        "--port 1": "empty script",
        "--port 1 --": "empty script",
        "--asdf 1 script ": "unrecognized arguments",
    }
    test_util.parse_args_check(
        positive_test_cases, negative_test_cases, launch.tensorflow.parse_args
    )


@mock.patch("subprocess.Popen")
@mock.patch("determined.get_cluster_info")
def test_single_node(
    mock_cluster_info: mock.MagicMock,
    mock_subprocess: mock.MagicMock,
) -> None:
    cluster_info = test_util.make_mock_cluster_info(["0.0.0.0"], 0, 1)
    mock_cluster_info.return_value = cluster_info
    script = ["python3", "train.py"]

    mock_exit_code = 99
    mock_proc = mock.MagicMock()
    mock_proc.wait.return_value = mock_exit_code

    mock_subprocess.return_value = mock_proc

    assert launch.tensorflow.main(88, script) == mock_exit_code

    launch_cmd = script

    # No TF_CONFIG or log wrapper for single node trainings.
    env = {**os.environ, "DET_CHIEF_IP": "0.0.0.0"}
    mock_subprocess.assert_called_once_with(launch_cmd, env=env)


@mock.patch("subprocess.Popen")
@mock.patch("determined.get_cluster_info")
def test_multi_node(
    mock_cluster_info: mock.MagicMock,
    mock_subprocess: mock.MagicMock,
) -> None:
    cluster_info = test_util.make_mock_cluster_info(["0.0.0.0", "0.0.0.1"], 1, 2)
    mock_cluster_info.return_value = cluster_info
    port = 88
    script = ["python3", "train.py"]

    mock_exit_code = 99
    mock_proc = mock.MagicMock()
    mock_proc.wait.return_value = mock_exit_code

    mock_subprocess.return_value = mock_proc

    assert launch.tensorflow.main(port, script) == mock_exit_code

    launch_cmd = launch.tensorflow.create_log_wrapper(1) + script

    env = {**os.environ, "DET_CHIEF_IP": "0.0.0.0"}
    env["TF_CONFIG"] = json.dumps(
        {
            "cluster": {"worker": [f"0.0.0.0:{port}", f"0.0.0.1:{port}"]},
            "task": {"type": "worker", "index": 1},
        }
    )
    mock_subprocess.assert_called_once_with(launch_cmd, env=env)
