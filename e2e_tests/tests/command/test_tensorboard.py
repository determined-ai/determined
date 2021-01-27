from pathlib import Path
from typing import Dict

import pytest

from tests import command as cmd
from tests import config as conf
from tests import experiment as exp
from tests.filetree import FileTree

AWAITING_METRICS = "TensorBoard is awaiting metrics"
SERVICE_READY = "TensorBoard is running at: http"
num_trials = 1


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


def s3_config(num_trials: int, secrets: Dict[str, str]) -> str:
    return f"""
description: noop_random
checkpoint_storage:
  type: s3
  access_key: {secrets["INTEGRATIONS_S3_ACCESS_KEY"]}
  secret_key: {secrets["INTEGRATIONS_S3_SECRET_KEY"]}
  bucket: {secrets["INTEGRATIONS_S3_BUCKET"]}
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


@pytest.mark.skip()  # type: ignore
@pytest.mark.slow  # type: ignore
@pytest.mark.e2e_cpu  # type: ignore
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


@pytest.mark.slow  # type: ignore
@pytest.mark.e2e_gpu  # type: ignore
def test_start_tensorboard_for_s3_experiment(tmp_path: Path, secrets: Dict[str, str]) -> None:
    """
    Start a random experiment configured with the s3 backend, start a
    TensorBoard instance pointed to the experiment, and kill the TensorBoard
    instance.
    """
    with FileTree(tmp_path, {"config.yaml": s3_config(1, secrets)}) as tree:
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


@pytest.mark.skip()  # type: ignore
@pytest.mark.slow  # type: ignore
@pytest.mark.e2e_gpu  # type: ignore
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

        trial_ids = [str(t["id"]) for t in exp.experiment_trials(multi_trial_exp_id)]

    command = [
        "tensorboard",
        "start",
        str(shared_fs_exp_id),
        str(s3_exp_id),
        "-t",
        *trial_ids,
        "--no-browser",
    ]

    with cmd.interactive_command(*command) as tensorboard:
        for line in tensorboard.stdout:
            if SERVICE_READY in line:
                break
            if AWAITING_METRICS in line:
                raise AssertionError("Tensorboard did not find metrics")
        else:
            raise AssertionError(f"Did not find {SERVICE_READY} in output")
