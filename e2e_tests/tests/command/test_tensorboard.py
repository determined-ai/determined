from pathlib import Path
from typing import Dict, Optional

import pytest
import yaml

from tests import command as cmd
from tests import config as conf
from tests import experiment as exp
from tests.filetree import FileTree

AWAITING_METRICS = "TensorBoard is awaiting metrics"
SERVICE_READY = "TensorBoard is running at: http"
num_trials = 1
custom_image_for_testing = "custom_alpine"


def shared_fs_config(num_trials: int) -> str:
    return f"""
description: noop_random
checkpoint_storage:
  type: shared_fs
  host_path: /tmp
hyperparameters:
  global_batch_size: 1
searcher:
  metric: validation_error
  smaller_is_better: true
  name: random
  max_trials: {num_trials}
  max_length:
    batches: 100
entrypoint: model_def:NoOpTrial
"""


def s3_config(num_trials: int, secrets: Dict[str, str], prefix: Optional[str] = None) -> str:
    config_dict = {
        "description": "noop_random",
        "checkpoint_storage": {
            "type": "s3",
            "access_key": secrets["INTEGRATIONS_S3_ACCESS_KEY"],
            "secret_key": secrets["INTEGRATIONS_S3_SECRET_KEY"],
            "bucket": secrets["INTEGRATIONS_S3_BUCKET"],
        },
        "hyperparameters": {"global_batch_size": 1},
        "searcher": {
            "metric": "validation_error",
            "smaller_is_better": True,
            "name": "random",
            "max_trials": num_trials,
            "max_length": {"batches": 100},
        },
        "entrypoint": "model_def:NoOpTrial",
    }
    if prefix is not None:
        config_dict["checkpoint_storage"]["prefix"] = prefix  # type: ignore

    return str(yaml.dump(config_dict))


@pytest.mark.slow
@pytest.mark.e2e_cpu
def test_start_tensorboard_for_shared_fs_experiment(tmp_path: Path) -> None:
    """
    Start a random experiment configured with the shared_fs backend, start a
    TensorBoard instance pointed to the experiment, and kill the TensorBoard
    instance.
    """
    with FileTree(tmp_path, {"config.yaml": shared_fs_config(1)}) as tree:
        config_path = tree.joinpath("config.yaml")
        experiment_id = exp.run_basic_test(
            str(config_path), conf.fixtures_path("no_op"), num_trials
        )

    command = ["tensorboard", "start", str(experiment_id), "--no-browser"]
    with cmd.interactive_command(*command) as tensorboard:
        for line in tensorboard.stdout:
            if SERVICE_READY in line:
                break
            if AWAITING_METRICS in line:
                raise AssertionError("Tensorboard did not find metrics")
        else:
            raise AssertionError(f"Did not find {SERVICE_READY} in output")


@pytest.mark.slow
@pytest.mark.e2e_gpu
@pytest.mark.tensorflow2
@pytest.mark.parametrize("prefix", [None, "my/test/prefix"])
def test_start_tensorboard_for_s3_experiment(
    tmp_path: Path, secrets: Dict[str, str], prefix: Optional[str]
) -> None:
    """
    Start a random experiment configured with the s3 backend, start a
    TensorBoard instance pointed to the experiment, and kill the TensorBoard
    instance.
    """
    with FileTree(tmp_path, {"config.yaml": s3_config(1, secrets, prefix)}) as tree:
        config_path = tree.joinpath("config.yaml")
        experiment_id = exp.run_basic_test(
            str(config_path), conf.fixtures_path("no_op"), num_trials
        )

    command = ["tensorboard", "start", str(experiment_id), "--no-browser"]
    with cmd.interactive_command(*command) as tensorboard:
        for line in tensorboard.stdout:
            if SERVICE_READY in line:
                break
        else:
            raise AssertionError(f"Did not find {SERVICE_READY} in output")


@pytest.mark.slow
@pytest.mark.e2e_cpu
@pytest.mark.tensorflow2
def test_start_tensorboard_for_multi_experiment(tmp_path: Path, secrets: Dict[str, str]) -> None:
    """
    Start 3 random experiments configured with the s3 and shared_fs backends,
    start a TensorBoard instance pointed to the experiments and some select
    trials, and kill the TensorBoard instance.
    """
    with FileTree(
        tmp_path,
        {
            "shared_fs_config.yaml": shared_fs_config(1),
            "s3_config.yaml": s3_config(1, secrets),
            "multi_trial_config.yaml": shared_fs_config(3),
        },
    ) as tree:
        shared_conf_path = tree.joinpath("shared_fs_config.yaml")
        shared_fs_exp_id = exp.run_basic_test(
            str(shared_conf_path), conf.fixtures_path("no_op"), num_trials
        )

        s3_conf_path = tree.joinpath("s3_config.yaml")
        s3_exp_id = exp.run_basic_test(str(s3_conf_path), conf.fixtures_path("no_op"), num_trials)

        multi_trial_config = tree.joinpath("multi_trial_config.yaml")
        multi_trial_exp_id = exp.run_basic_test(
            str(multi_trial_config), conf.fixtures_path("no_op"), 3
        )

        trial_ids = [str(t.trial.id) for t in exp.experiment_trials(multi_trial_exp_id)]

    command = [
        "tensorboard",
        "start",
        str(shared_fs_exp_id),
        str(s3_exp_id),
        str(multi_trial_exp_id),
        "-t",
        *trial_ids,
        "--no-browser",
    ]

    with cmd.interactive_command(*command) as tensorboard:
        for line in tensorboard.stdout:
            if SERVICE_READY in line:
                break
        else:
            raise AssertionError(f"Did not find {SERVICE_READY} in output")


@pytest.mark.slow
@pytest.mark.e2e_gpu
@pytest.mark.tensorflow2
@pytest.mark.parametrize("prefix", [None, "my/test/prefix"])
def test_start_tensorboard_with_custom_image(
    tmp_path: Path, secrets: Dict[str, str], prefix: Optional[str]
) -> None:
    """
    Start a random experiment configured with the shared_fs backend, start a
    TensorBoard instance pointed to the experiment, and kill the TensorBoard
    instance.
    """
    with FileTree(tmp_path, {"config.yaml": s3_config(1, secrets, prefix)}) as tree:
        config_path = tree.joinpath("config.yaml")
        experiment_id = exp.run_basic_test(
            str(config_path), conf.fixtures_path("no_op"), num_trials
        )

    command = [
        "tensorboard",
        "start",
        str(experiment_id),
        "--no-browser",
        "--detach",
        "--config",
        f"environment.image={custom_image_for_testing}",
    ]
    with cmd.interactive_command(*command) as tensorboard:
        t_id = tensorboard.task_id
        commandt = ["tensorboard", "config", t_id]
        with cmd.interactive_command(*commandt, task_id=t_id) as tensorboard_config:
            for line in tensorboard_config.stdout:
                if "cpu" in line or "cuda:" in line or "rocm" in line:
                    if custom_image_for_testing in line:
                        break
                    else:
                        raise AssertionError(f"Setting custom image not working properly: {line}")
            else:
                raise AssertionError("Did not find custom image in output")
