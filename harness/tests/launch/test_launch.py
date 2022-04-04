from unittest import mock

import determined as det
from determined.exec import launch


@mock.patch("subprocess.Popen")
@mock.patch("determined.exec.harness.main")
def test_launch_native(mock_harness: mock.MagicMock, mock_subprocess: mock.MagicMock) -> None:
    """
    entrypoint: None
    Native enabled -> launch harness.py
    """
    native = {
        "internal": {"native": {"command": ["script.py"]}},
        "resources": {"slots_per_trial": 1},
    }

    launch.launch(det.ExperimentConfig(native))
    mock_harness.assert_called_once_with(train_entrypoint=None)
    mock_subprocess.assert_not_called()


@mock.patch("subprocess.Popen")
@mock.patch("determined.exec.harness.main")
def test_launch_native_parallel(
    mock_harness: mock.MagicMock, mock_subprocess: mock.MagicMock
) -> None:
    """
    entrypoint: None
    Native enabled -> launch harness.py
    """
    native = {
        "internal": {"native": {"command": ["script.py"]}},
        "resources": {"slots_per_trial": 4},
    }

    launch.launch(det.ExperimentConfig(native))
    mock_harness.assert_not_called()
    entrypoint_list = ["python3", "-m", "determined.launch.autohorovod", "__NATIVE__"]
    mock_subprocess.assert_called_once_with(entrypoint_list)


@mock.patch("subprocess.Popen")
@mock.patch("determined.exec.harness.main")
def test_launch_legacy(mock_harness: mock.MagicMock, mock_subprocess: mock.MagicMock) -> None:
    """
    entrypoint: "model_def:TrialClass"
    Distributed training -> launch with autohorovod
    Non-distributed training -> launch harness with trial class
    """
    legacy_entrypoint = "model_def:TrialClass"
    legacy_distributed = {"resources": {"slots_per_trial": 4}, "entrypoint": legacy_entrypoint}

    launch.launch(det.ExperimentConfig(legacy_distributed))
    mock_harness.assert_not_called()
    mock_subprocess.assert_called_once_with(
        ["sh", "-c", f"python3 -m determined.launch.autohorovod {legacy_entrypoint}"]
    )
    mock_harness.reset_mock()
    mock_subprocess.reset_mock()

    legacy_non_distributed = {"resources": {"slots_per_trial": 1}, "entrypoint": legacy_entrypoint}

    launch.launch(det.ExperimentConfig(legacy_non_distributed))
    mock_harness.assert_called_once_with(train_entrypoint=legacy_entrypoint)
    mock_subprocess.assert_not_called()
    mock_harness.reset_mock()
    mock_subprocess.reset_mock()


@mock.patch("subprocess.Popen")
@mock.patch("determined.exec.harness.main")
def test_launch(mock_harness: mock.MagicMock, mock_subprocess: mock.MagicMock) -> None:
    """
    entrypoint: "python3 train.py" or ["python3", "train.py"]
    Launch generic shell script
    """
    entrypoint = "python3 train.py"
    experiment_config = {"resources": {"slots_per_trial": 4}, "entrypoint": entrypoint}

    launch.launch(det.ExperimentConfig(experiment_config))
    mock_harness.assert_not_called()
    mock_subprocess.assert_called_once_with(["sh", "-c", entrypoint])
    mock_harness.reset_mock()
    mock_subprocess.reset_mock()

    entrypoint_list = ["python3", "train.py"]
    experiment_config = {"resources": {"slots_per_trial": 4}, "entrypoint": entrypoint_list}
    launch.launch(det.ExperimentConfig(experiment_config))
    mock_harness.assert_not_called()
    mock_subprocess.assert_called_once_with(entrypoint_list)
