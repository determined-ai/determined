"""Tests of advanced modules for a distributed context.

We're particularly interested in correctness of the following tests in a context with multiple
agents.

Since these test advanced models usually maintained by AMLE, these tests are also AMLE-owned.
"""
import os
import tempfile

import pytest
import yaml

from tests import config as conf
from tests import experiment as exp


@pytest.mark.advanced_distributed
@pytest.mark.gpu_required
def test_deepspeed_moe() -> None:
    config = conf.load_config(conf.deepspeed_examples_path("cifar10_moe/moe.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(config, conf.deepspeed_examples_path("cifar10_moe"), 1)


@pytest.mark.advanced_distributed
@pytest.mark.gpu_required
def test_deepspeed_zero() -> None:
    config = conf.load_config(conf.deepspeed_examples_path("cifar10_moe/zero_stages.yaml"))
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_min_validation_period(config, {"batches": 100})

    exp.run_basic_test_with_temp_config(config, conf.deepspeed_examples_path("cifar10_moe"), 1)


@pytest.mark.advanced_distributed
@pytest.mark.gpu_required
def test_deepspeed_pipeline_parallel() -> None:
    config = conf.load_config(conf.deepspeed_examples_path("pipeline_parallelism/distributed.yaml"))
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_min_validation_period(config, {"batches": 100})

    exp.run_basic_test_with_temp_config(
        config, conf.deepspeed_examples_path("pipeline_parallelism"), 1
    )


@pytest.mark.advanced_distributed
@pytest.mark.gpu_required
def test_gpt_neox_zero1() -> None:
    config = conf.load_config(conf.deepspeed_examples_path("gpt_neox/zero1.yaml"))
    config = conf.set_max_length(config, {"batches": 100})
    config = conf.set_min_validation_period(config, {"batches": 100})
    # Changing to satisfy cluter size and gpu mem limitations.
    config = conf.set_slots_per_trial(config, 8)
    config["hyperparameters"]["conf_file"] = ["350M.yml", "determined_cluster.yml"]
    config["hyperparameters"]["overwrite_values"]["train_batch_size"] = 32

    exp.run_basic_test_with_temp_config(config, conf.deepspeed_examples_path("gpt_neox"), 1)


@pytest.mark.advanced_distributed
@pytest.mark.gpu_required
def test_deepspeed_dcgan() -> None:
    config = conf.load_config(conf.deepspeed_examples_path("deepspeed_dcgan/mnist.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(config, conf.deepspeed_examples_path("deepspeed_dcgan"), 1)


@pytest.mark.advanced_distributed
@pytest.mark.gpu_required
def test_deepspeed_cpu_offloading() -> None:
    config = conf.load_config(
        conf.deepspeed_examples_path("cifar10_cpu_offloading/zero_3_cpu_offload.yaml")
    )
    config = conf.set_max_length(config, {"batches": 100})

    exp.run_basic_test_with_temp_config(
        config, conf.deepspeed_examples_path("cifar10_cpu_offloading"), 1
    )


@pytest.mark.advanced_distributed
@pytest.mark.gpu_required
def test_hf_trainer_image_classification_deepspeed_autotuning() -> None:
    test_dir = "hf_image_classification"
    config_path = conf.hf_trainer_examples_path(f"{test_dir}/deepspeed.yaml")
    config = conf.load_config(config_path)
    with tempfile.NamedTemporaryFile() as tf:
        with open(tf.name, "w") as f:
            yaml.dump(config, f)
        _ = exp.run_basic_autotuning_test(
            tf.name,
            conf.hf_trainer_examples_path(test_dir),
            1,
            search_method_name="asha",
        )


@pytest.mark.advanced_distributed
@pytest.mark.gpu_required
def test_hf_trainer_language_modeling_deepspeed_autotuning() -> None:
    test_dir = "hf_language_modeling"
    config_path = conf.hf_trainer_examples_path(f"{test_dir}/deepspeed.yaml")
    config = conf.load_config(config_path)
    with tempfile.NamedTemporaryFile() as tf:
        with open(tf.name, "w") as f:
            yaml.dump(config, f)
        _ = exp.run_basic_autotuning_test(
            tf.name,
            conf.hf_trainer_examples_path(test_dir),
            1,
            search_method_name="binary",
        )


@pytest.mark.advanced_distributed
@pytest.mark.gpu_required
def test_torchvision_core_api_deepspeed_autotuning() -> None:
    test_dir = "torchvision/core_api"
    config_path = conf.deepspeed_autotune_examples_path(f"{test_dir}/deepspeed.yaml")
    config = conf.load_config(config_path)
    with tempfile.NamedTemporaryFile() as tf:
        with open(tf.name, "w") as f:
            yaml.dump(config, f)
        _ = exp.run_basic_autotuning_test(
            tf.name,
            conf.deepspeed_autotune_examples_path(test_dir),
            1,
            search_method_name="asha",
        )


@pytest.mark.advanced_distributed
@pytest.mark.gpu_required
def test_torchvision_deepspeed_trial_deepspeed_autotuning() -> None:
    test_dir = "torchvision/deepspeed_trial"
    config_path = conf.deepspeed_autotune_examples_path(f"{test_dir}/deepspeed.yaml")
    config = conf.load_config(config_path)
    with tempfile.NamedTemporaryFile() as tf:
        with open(tf.name, "w") as f:
            yaml.dump(config, f)
        _ = exp.run_basic_autotuning_test(
            tf.name,
            conf.deepspeed_autotune_examples_path(test_dir),
            1,
            search_method_name="random",
        )


@pytest.mark.advanced_distributed
@pytest.mark.gpu_required
def test_textual_inversion_stable_diffusion_finetune() -> None:
    """Requires downloading weights from Hugging Face via an authorization token. The experiment
    expects the token to be stored in the HF_AUTH_TOKEN environment variable.

    Hugging Face tokens are stored in CircleCI's "hugging-face" context as HF_READ_ONLY_TOKEN and
    HF_READ_WRITE_TOKEN environment variables which are accessible during CI runs.

    The Hugging Face account details can be found at
    github.com/determined-ai/secrets/blob/master/ci/hugging_face.txt
    """
    config = conf.load_config(
        conf.diffusion_examples_path(
            "textual_inversion_stable_diffusion/finetune_const_advanced.yaml"
        )
    )
    config = conf.set_max_length(config, 10)
    try:
        config = conf.set_environment_variables(
            config, [f'HF_AUTH_TOKEN={os.environ["HF_READ_ONLY_TOKEN"]}']
        )
        exp.run_basic_test_with_temp_config(
            config, conf.diffusion_examples_path("textual_inversion_stable_diffusion"), 1
        )
    except KeyError as k:
        if str(k) == "'HF_READ_ONLY_TOKEN'":
            pytest.skip("HF_READ_ONLY_TOKEN CircleCI environment variable missing, skipping test")
        else:
            raise k


@pytest.mark.advanced_distributed
@pytest.mark.gpu_required
def test_textual_inversion_stable_diffusion_generate() -> None:
    """Requires downloading weights from Hugging Face via an authorization token. The experiment
    expects the token to be stored in the HF_AUTH_TOKEN environment variable.

    Hugging Face tokens are stored in CircleCI's "hugging-face" context as HF_READ_ONLY_TOKEN and
    HF_READ_WRITE_TOKEN environment variables which are accessible during CI runs.

    The Hugging Face account details can be found at
    github.com/determined-ai/secrets/blob/master/ci/hugging_face.txt
    """
    config = conf.load_config(
        conf.diffusion_examples_path("textual_inversion_stable_diffusion/generate_grid.yaml")
    )
    # Shorten the Experiment and reduce to two Trials.
    config = conf.set_max_length(config, 2)
    prompt_vals = config["hyperparameters"]["call_kwargs"]["prompt"]["vals"]
    config["hyperparameters"]["call_kwargs"]["guidance_scale"] = 7.5
    while len(prompt_vals) > 1:
        prompt_vals.pop()

    try:
        config = conf.set_environment_variables(
            config, [f'HF_AUTH_TOKEN={os.environ["HF_READ_ONLY_TOKEN"]}']
        )
        exp.run_basic_test_with_temp_config(
            config, conf.diffusion_examples_path("textual_inversion_stable_diffusion"), 2
        )
    except KeyError as k:
        if str(k) == "'HF_READ_ONLY_TOKEN'":
            pytest.skip("HF_READ_ONLY_TOKEN CircleCI environment variable missing, skipping test")
        else:
            raise k
