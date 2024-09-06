import os
import shutil
import tempfile

import pytest

from tests import api_utils
from tests import config as conf
from tests import experiment as exp


@pytest.mark.distributed
def test_mnist_pytorch_distributed() -> None:
    sess = api_utils.user_session()
    config = conf.load_config(conf.tutorials_path("mnist_pytorch/distributed.yaml"))
    assert "--epochs 1" in config["entrypoint"], "update test to match tutorial"
    config["entrypoint"] = config["entrypoint"].replace("--epochs 1", "--batches 64")
    exp.run_basic_test_with_temp_config(sess, config, conf.fixtures_path("mnist_pytorch"), 1)


@pytest.mark.distributed
def test_mnist_pytorch_set_stop_requested_distributed() -> None:
    sess = api_utils.user_session()
    config = conf.load_config(conf.fixtures_path("mnist_pytorch/distributed-stop-requested.yaml"))
    exp.run_basic_test_with_temp_config(sess, config, conf.fixtures_path("mnist_pytorch"), 1)


@pytest.mark.distributed
@pytest.mark.gpu_required
def test_hf_trainer_api_integration() -> None:
    sess = api_utils.user_session()
    test_dir = "hf_image_classification"
    config = conf.load_config(conf.hf_trainer_examples_path(f"{test_dir}/distributed.yaml"))
    exp.run_basic_test_with_temp_config(sess, config, conf.hf_trainer_examples_path(test_dir), 1)


@pytest.mark.deepspeed
@pytest.mark.gpu_required
def test_gpt_neox_zero1() -> None:
    sess = api_utils.user_session()
    config = conf.load_config(conf.deepspeed_examples_path("gpt_neox/zero1.yaml"))
    config["searcher"]["max_length"] = {"batches": 100}
    config = conf.set_min_validation_period(config, {"batches": 100})
    # Changing to satisfy cluter size and gpu mem limitations.
    config = conf.set_slots_per_trial(config, 8)
    config["hyperparameters"]["conf_file"] = ["350M.yml", "determined_cluster.yml"]
    config["hyperparameters"]["overwrite_values"]["train_batch_size"] = 32

    exp.run_basic_test_with_temp_config(sess, config, conf.deepspeed_examples_path("gpt_neox"), 1)


HUGGINGFACE_CONTEXT_ERR_MSG = """
A HF_READ_ONLY_TOKEN environment variable is missing!

If this error was raised in CircleCI, verify that the test was run using the
`hugging-face` context. This context should be injecting the expected
`HF_READ_ONLY_TOKEN`variable.

If this error was raised in another context, set the HF_READ_ONLY_TOKEN env var manually with its
value a valid HF authorization token.
"""


@pytest.mark.distributed
@pytest.mark.gpu_required
def test_textual_inversion_stable_diffusion_finetune() -> None:
    """Requires downloading weights from Hugging Face via an authorization token. The experiment
    expects the token to be stored in the HF_AUTH_TOKEN environment variable.

    Hugging Face tokens are stored in CircleCI's "hugging-face" context as HF_READ_ONLY_TOKEN and
    HF_READ_WRITE_TOKEN environment variables which are accessible during CI runs.

    The Hugging Face account details can be found at
    github.com/determined-ai/secrets/blob/master/ci/hugging_face.txt
    """
    sess = api_utils.user_session()
    config = conf.load_config(
        conf.diffusion_examples_path(
            "textual_inversion_stable_diffusion/finetune_const_advanced.yaml"
        )
    )
    config["searcher"]["max_length"] = 10
    try:
        config = conf.set_environment_variables(
            config, [f'HF_AUTH_TOKEN={os.environ["HF_READ_ONLY_TOKEN"]}']
        )
        exp.run_basic_test_with_temp_config(
            sess, config, conf.diffusion_examples_path("textual_inversion_stable_diffusion"), 1
        )
    except KeyError as k:
        if str(k) == "'HF_READ_ONLY_TOKEN'":
            raise RuntimeError(HUGGINGFACE_CONTEXT_ERR_MSG)
        else:
            raise k


@pytest.mark.distributed
@pytest.mark.gpu_required
def test_textual_inversion_stable_diffusion_generate() -> None:
    """Requires downloading weights from Hugging Face via an authorization token. The experiment
    expects the token to be stored in the HF_AUTH_TOKEN environment variable.

    Hugging Face tokens are stored in CircleCI's "hugging-face" context as HF_READ_ONLY_TOKEN and
    HF_READ_WRITE_TOKEN environment variables which are accessible during CI runs.

    The Hugging Face account details can be found at
    github.com/determined-ai/secrets/blob/master/ci/hugging_face.txt
    """
    sess = api_utils.user_session()
    config = conf.load_config(
        conf.diffusion_examples_path("textual_inversion_stable_diffusion/generate_grid.yaml")
    )
    # Shorten the Experiment and reduce to two Trials.
    config["searcher"]["max_length"] = 2
    prompt_vals = config["hyperparameters"]["call_kwargs"]["prompt"]["vals"]
    config["hyperparameters"]["call_kwargs"]["guidance_scale"] = 7.5
    while len(prompt_vals) > 1:
        prompt_vals.pop()

    try:
        config = conf.set_environment_variables(
            config, [f'HF_AUTH_TOKEN={os.environ["HF_READ_ONLY_TOKEN"]}']
        )
        exp.run_basic_test_with_temp_config(
            sess, config, conf.diffusion_examples_path("textual_inversion_stable_diffusion"), 2
        )
    except KeyError as k:
        if str(k) == "'HF_READ_ONLY_TOKEN'":
            raise RuntimeError(HUGGINGFACE_CONTEXT_ERR_MSG)
        else:
            raise k


@pytest.mark.distributed
@pytest.mark.gpu_required
def test_torch_batch_process_generate_embedding() -> None:
    sess = api_utils.user_session()
    config = conf.load_config(
        conf.features_examples_path("torch_batch_process_embeddings/distributed.yaml")
    )

    with tempfile.TemporaryDirectory() as tmpdir:
        copy_destination = os.path.join(tmpdir, "example")
        shutil.copytree(
            conf.features_examples_path("torch_batch_process_embeddings"),
            copy_destination,
        )
        exp.run_basic_test_with_temp_config(sess, config, copy_destination, 1)
