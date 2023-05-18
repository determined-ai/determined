import subprocess
from pathlib import Path
from typing import Dict, Optional

import pytest

from determined.common import api, yaml
from tests import command as cmd
from tests import config as conf
from tests import experiment as exp
from tests.api_utils import determined_test_session
from tests.filetree import FileTree

num_trials = 1


def shared_fs_config(num_trials: int) -> str:
    return f"""
name: noop_random
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
        assert tensorboard.task_id is not None
        err = api.task_is_ready(determined_test_session(), tensorboard.task_id)
        assert err is None, err


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
        assert tensorboard.task_id is not None
        err = api.task_is_ready(determined_test_session(), tensorboard.task_id)
        assert err is None, err


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
        assert tensorboard.task_id is not None
        err = api.task_is_ready(determined_test_session(), tensorboard.task_id)
        assert err is None, err


@pytest.mark.e2e_cpu
def test_start_tensorboard_with_custom_image(tmp_path: Path) -> None:
    """
    Start a random experiment, start a TensorBoard instance pointed
    to the experiment with custom image, verify the image has been set.
    """
    experiment_id = exp.run_basic_test(
        conf.fixtures_path("no_op/single-one-short-step.yaml"),
        conf.fixtures_path("no_op"),
        1,
    )
    command = [
        "det",
        "-m",
        conf.make_master_url(),
        "tensorboard",
        "start",
        str(experiment_id),
        "--no-browser",
        "--detach",
        "--config",
        "environment.image=alpine",
    ]
    res = subprocess.run(command, universal_newlines=True, stdout=subprocess.PIPE, check=True)
    t_id = res.stdout.strip("\n")
    command = ["det", "-m", conf.make_master_url(), "tensorboard", "config", t_id]
    res = subprocess.run(command, universal_newlines=True, stdout=subprocess.PIPE, check=True)
    config = yaml.safe_load(res.stdout)
    assert (
        config["environment"]["image"]["cpu"] == "alpine"
        and config["environment"]["image"]["cuda"] == "alpine"
        and config["environment"]["image"]["rocm"] == "alpine"
    ), config


@pytest.mark.e2e_cpu
def test_tensorboard_inherit_image_pull_secrets(tmp_path: Path) -> None:
    """
    Start a random experiment with image_pull_secrets, start a TensorBoard
    instance pointed to the experiment, verify the secrets are inherited.
    """
    exp_secrets = [{"name": "ips"}]
    config_obj = conf.load_config(conf.fixtures_path("no_op/single-one-short-step.yaml"))
    pod = config_obj.setdefault("environment", {}).setdefault("pod_spec", {})
    pod.setdefault("spec", {})["imagePullSecrets"] = [{"name": "ips"}]
    experiment_id = exp.run_basic_test_with_temp_config(config_obj, conf.fixtures_path("no_op"), 1)

    command = [
        "det",
        "-m",
        conf.make_master_url(),
        "tensorboard",
        "start",
        str(experiment_id),
        "--no-browser",
        "--detach",
    ]
    res = subprocess.run(command, universal_newlines=True, stdout=subprocess.PIPE, check=True)
    t_id = res.stdout.strip("\n")
    command = ["det", "-m", conf.make_master_url(), "tensorboard", "config", t_id]
    res = subprocess.run(command, universal_newlines=True, stdout=subprocess.PIPE, check=True)
    config = yaml.safe_load(res.stdout)

    ips = config["environment"]["pod_spec"]["spec"]["imagePullSecrets"]

    assert ips == exp_secrets, (ips, exp_secrets)
