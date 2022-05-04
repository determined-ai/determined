from typing import Any, Dict, List
from unittest import mock

import determined as det
from determined.exec import launch


@mock.patch("subprocess.Popen")
def do_test_launch(config: Dict[str, Any], cmd: List[str], mock_popen: mock.MagicMock) -> None:
    mock_proc = mock.MagicMock()
    mock_proc.wait.return_value = 99
    mock_popen.return_value = mock_proc
    assert launch.launch(det.ExperimentConfig(config)) == 99
    mock_popen.assert_called_once_with(cmd)


def test_launch_trial() -> None:
    entrypoint = "model_def:TrialClass"
    config = {"entrypoint": entrypoint}
    cmd = ["python3", "-m", "determined.launch.horovod", "--autohorovod", "--trial", entrypoint]
    do_test_launch(config, cmd)


def test_launch_string() -> None:
    entrypoint = "a b c"
    config = {"entrypoint": entrypoint}
    cmd = ["sh", "-c", entrypoint]
    do_test_launch(config, cmd)


def test_launch_list() -> None:
    entrypoint = ["a", "b", "c"]
    config = {"entrypoint": entrypoint}
    cmd = [*entrypoint]
    do_test_launch(config, cmd)
