import os
import runpy
import signal
import time
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
    mock_popen.assert_called_once_with(cmd)


def test_launch_trial() -> None:
    entrypoint = "model_def:TrialClass"
    config = {"entrypoint": entrypoint}
    cmd = ["python3", "-m", "determined.launch.torch_distributed", "--", "--trial", entrypoint]
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
    with test_util.set_env_vars({"DET_RESOURCES_ID": "resourcesId"}):
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


@mock.patch("subprocess.Popen")
@mock.patch("determined.common.api.authentication.login_from_task")
def test_launch_catches_slurm_preemption(
    mock_login_from_task: mock.MagicMock,
    mock_popen: mock.MagicMock,
) -> None:
    """
    Determined's slurm preemption logic involves catching the SIGTERM sent by slurm, reporting it
    to the master, then the python code will detect that through the normal should_preempt() logic.

    Here, we need to test that the launch script actually catches SIGTERM and makes the right API
    call to the master.
    """

    # Send a SIGTERM to this process during the Popen.wait().
    def wait_side_effect() -> int:
        os.kill(os.getpid(), signal.SIGTERM)
        # Make any syscall at all to ensure the signal handler has a good chance to fire.
        time.sleep(0.000001)
        # SIGTERM handler should have fired, make sure the right API call was made.
        mock_login_from_task.return_value.post.assert_called_once_with(
            "/api/v1/allocations/allocationId/signals/pending_preemption"
        )
        # Return a unique value to make sure our mocks are wired up right.
        return 789

    mock_popen.return_value.wait.side_effect = wait_side_effect

    old_handler = signal.getsignal(signal.SIGTERM)
    try:
        with test_util.set_env_vars({"DET_RESOURCES_TYPE": "slurm-job"}):
            with test_util.set_mock_cluster_info(["0.0.0.1"], 0, 1):
                config = {"entrypoint": "yo"}
                assert launch.launch(det.ExperimentConfig(config)) == 789
    finally:
        signal.signal(signal.SIGTERM, old_handler)
