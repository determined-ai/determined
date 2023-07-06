import runpy
from typing import Any, Dict, List
from unittest import mock

import pytest

import determined as det
from determined.exec import launch
from tests.launch import test_util


@mock.patch("subprocess.Popen")
def do_test_launch(config: Dict[str, Any], cmd: List[str], mock_popen: mock.MagicMock) -> None:
    mock_proc = mock.MagicMock()
    mock_proc.wait.return_value = 99
    mock_popen.return_value = mock_proc
    assert launch.launch(det.ExperimentConfig(config)) == 99
    mock_popen.assert_called_once_with(cmd, start_new_session=True)


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


@mock.patch("determined.common.storage.validate_config")
def test_launch_script(mock_validate_config: mock.MagicMock) -> None:
    # Use runpy to actually run the whole launch script.
    with test_util.set_resources_id_env_var():
        with test_util.set_mock_cluster_info(["0.0.0.1"], 0, 1) as info:
            # Successful entrypoints exit 0.
            info.trial._config["entrypoint"] = ["true"]
            with pytest.raises(SystemExit) as e:
                runpy.run_module("determined.exec.launch", run_name="__main__", alter_sys=True)
            assert e.value.code == 0, e

            # Failing entrypoints exit 1.
            info.trial._config["entrypoint"] = ["false"]
            with pytest.raises(SystemExit) as e:
                runpy.run_module("determined.exec.launch", run_name="__main__", alter_sys=True)
            assert e.value.code == 1, e
