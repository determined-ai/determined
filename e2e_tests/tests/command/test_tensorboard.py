import pathlib
from typing import Dict, Optional

import pytest

from determined.common import api, util
from tests import api_utils
from tests import command as cmd
from tests import config as conf
from tests import detproc
from tests import experiment as exp
from tests import filetree

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

    return str(util.yaml_safe_dump(config_dict))


@pytest.mark.slow
@pytest.mark.e2e_cpu
def test_start_tensorboard_for_shared_fs_experiment(tmp_path: pathlib.Path) -> None:
    """
    Start a random experiment configured with the shared_fs backend, start a
    TensorBoard instance pointed to the experiment, and kill the TensorBoard
    instance.
    """
    sess = api_utils.user_session()
    with filetree.FileTree(tmp_path, {"config.yaml": shared_fs_config(1)}) as tree:
        config_path = tree.joinpath("config.yaml")
        experiment_id = exp.run_basic_test(
            sess, str(config_path), conf.fixtures_path("no_op"), num_trials
        )

    command = ["tensorboard", "start", str(experiment_id), "--no-browser"]
    with cmd.interactive_command(sess, command) as tensorboard:
        assert tensorboard.task_id is not None
        err = api.wait_for_task_ready(sess, tensorboard.task_id)
        assert err is None, err


@pytest.mark.slow
@pytest.mark.e2e_gpu
@pytest.mark.tensorflow2
@pytest.mark.parametrize("prefix", [None, "my/test/prefix"])
def test_start_tensorboard_for_s3_experiment(
    tmp_path: pathlib.Path, secrets: Dict[str, str], prefix: Optional[str]
) -> None:
    """
    Start a random experiment configured with the s3 backend, start a
    TensorBoard instance pointed to the experiment, and kill the TensorBoard
    instance.
    """
    sess = api_utils.user_session()
    with filetree.FileTree(tmp_path, {"config.yaml": s3_config(1, secrets, prefix)}) as tree:
        config_path = tree.joinpath("config.yaml")
        experiment_id = exp.run_basic_test(
            sess, str(config_path), conf.fixtures_path("no_op"), num_trials
        )

    command = ["tensorboard", "start", str(experiment_id), "--no-browser"]
    with cmd.interactive_command(sess, command) as tensorboard:
        assert tensorboard.task_id is not None
        err = api.wait_for_task_ready(sess, tensorboard.task_id)
        assert err is None, err


@pytest.mark.slow
@pytest.mark.e2e_cpu
@pytest.mark.e2e_slurm
@pytest.mark.e2e_pbs
@pytest.mark.tensorflow2
def test_start_tensorboard_for_multi_experiment(
    tmp_path: pathlib.Path, secrets: Dict[str, str]
) -> None:
    """
    Start 3 random experiments configured with the s3 and shared_fs backends,
    start a TensorBoard instance pointed to the experiments and some select
    trials, and kill the TensorBoard instance.
    """
    sess = api_utils.user_session()
    with filetree.FileTree(
        tmp_path,
        {
            "shared_fs_config.yaml": shared_fs_config(1),
            "s3_config.yaml": s3_config(1, secrets),
            "multi_trial_config.yaml": shared_fs_config(3),
        },
    ) as tree:
        shared_conf_path = tree.joinpath("shared_fs_config.yaml")
        shared_fs_exp_id = exp.run_basic_test(
            sess, str(shared_conf_path), conf.fixtures_path("no_op"), num_trials
        )

        s3_conf_path = tree.joinpath("s3_config.yaml")
        s3_exp_id = exp.run_basic_test(
            sess, str(s3_conf_path), conf.fixtures_path("no_op"), num_trials
        )

        multi_trial_config = tree.joinpath("multi_trial_config.yaml")
        multi_trial_exp_id = exp.run_basic_test(
            sess, str(multi_trial_config), conf.fixtures_path("no_op"), 3
        )

        trial_ids = [str(t.trial.id) for t in exp.experiment_trials(sess, multi_trial_exp_id)]

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

    with cmd.interactive_command(sess, command) as tensorboard:
        assert tensorboard.task_id is not None
        err = api.wait_for_task_ready(sess, tensorboard.task_id)
        assert err is None, err


@pytest.mark.e2e_cpu
def test_start_tensorboard_with_custom_image() -> None:
    """
    Start a random experiment, start a TensorBoard instance pointed
    to the experiment with custom image, verify the image has been set.
    """
    sess = api_utils.user_session()
    experiment_id = exp.run_basic_test(
        sess,
        conf.fixtures_path("no_op/single-one-short-step.yaml"),
        conf.fixtures_path("no_op"),
        1,
    )
    command = [
        "det",
        "tensorboard",
        "start",
        str(experiment_id),
        "--no-browser",
        "--detach",
        "--config",
        "environment.image=python:3.8.16",
    ]
    t_id = detproc.check_output(sess, command).strip()
    command = ["det", "tensorboard", "config", t_id]
    res = detproc.check_output(sess, command)
    config = util.yaml_safe_load(res)
    assert (
        config["environment"]["image"]["cpu"] == "python:3.8.16"
        and config["environment"]["image"]["cuda"] == "python:3.8.16"
        and config["environment"]["image"]["rocm"] == "python:3.8.16"
    ), config


@pytest.mark.e2e_cpu
def test_tensorboard_inherit_image_pull_secrets() -> None:
    """
    Start a random experiment with image_pull_secrets, start a TensorBoard
    instance pointed to the experiment, verify the secrets are inherited.
    """
    sess = api_utils.user_session()
    exp_secrets = [{"name": "ips"}]
    config_obj = conf.load_config(conf.fixtures_path("no_op/single-one-short-step.yaml"))
    pod = config_obj.setdefault("environment", {}).setdefault("pod_spec", {})
    pod.setdefault("spec", {})["imagePullSecrets"] = [{"name": "ips"}]
    experiment_id = exp.run_basic_test_with_temp_config(
        sess, config_obj, conf.fixtures_path("no_op"), 1
    )

    command = [
        "det",
        "tensorboard",
        "start",
        str(experiment_id),
        "--no-browser",
        "--detach",
    ]
    t_id = detproc.check_output(sess, command).strip()
    command = ["det", "tensorboard", "config", t_id]
    res = detproc.check_output(sess, command)
    config = util.yaml_safe_load(res)

    ips = config["environment"]["pod_spec"]["spec"]["imagePullSecrets"]

    assert ips == exp_secrets, (ips, exp_secrets)


@pytest.mark.e2e_cpu
def test_delete_tensorboard_for_experiment() -> None:
    """
    Start a random experiment, start a TensorBoard instance pointed to
    the experiment, delete tensorboard and verify deletion.
    """
    sess = api_utils.user_session()
    config_obj = conf.load_config(conf.tutorials_path("mnist_pytorch/const.yaml"))
    experiment_id = exp.run_basic_test_with_temp_config(
        sess, config_obj, conf.tutorials_path("mnist_pytorch"), 1
    )

    command = ["det", "e", "delete-tb-files", str(experiment_id)]
    detproc.check_output(sess, command)

    # Check if Tensorboard files are deleted
    tb_path = sorted(pathlib.Path("/tmp/determined-cp/").glob("*/tensorboard"))[0]
    tb_path = tb_path / "experiment" / str(experiment_id)
    assert not pathlib.Path(tb_path).exists()


@pytest.mark.e2e_cpu
def test_tensorboard_directory_storage(tmp_path: pathlib.Path) -> None:
    sess = api_utils.user_session()
    config_obj = conf.load_config(conf.fixtures_path("no_op/single-one-short-step.yaml"))
    config_obj["checkpoint_storage"] = {
        "type": "directory",
        "container_path": "/tmp/somepath",
    }
    tb_config = {}
    tb_config["bind_mounts"] = config_obj["bind_mounts"] = [
        {
            "host_path": "/tmp/",
            "container_path": "/tmp/somepath",
        }
    ]

    tb_config_path = tmp_path / "tb.yaml"
    with tb_config_path.open("w") as fout:
        util.yaml_safe_dump(tb_config, fout)

    experiment_id = exp.run_basic_test_with_temp_config(
        sess, config_obj, conf.fixtures_path("no_op"), 1
    )

    command = [
        "tensorboard",
        "start",
        str(experiment_id),
        "--no-browser",
        "--config-file",
        str(tb_config_path),
    ]

    with cmd.interactive_command(sess, command) as tensorboard:
        assert tensorboard.task_id is not None
        err = api.wait_for_task_ready(sess, tensorboard.task_id)
        assert err is None, err
