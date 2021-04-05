import os
import subprocess
from typing import Dict

import pytest

from tests import config as conf
from tests import experiment as exp


def set_docker_image(config: Dict) -> Dict:
    git_short_hash = subprocess.check_output(["git", "rev-parse", "--short", "HEAD"]).strip()
    git_short_hash = git_short_hash.decode("utf-8")

    config = conf.set_image(
        config, conf.TF1_CPU_IMAGE, f"determinedai/model-hub-transformers:{git_short_hash}"
    )
    return config


@pytest.mark.model_hub  # type: ignore
@pytest.mark.distributed  # type: ignore
def test_token_classification_ner() -> None:
    example_path = conf.model_hub_examples_path("huggingface/token-classification")
    config = conf.load_config(os.path.join(example_path, "ner_config.yaml"))
    config = conf.set_max_length(config, {"batches": 200})
    config = set_docker_image(config)

    exp.run_basic_test_with_temp_config(config, example_path, 1)


@pytest.mark.model_hub  # type: ignore
@pytest.mark.distributed  # type: ignore
def test_token_classification_ner_amp() -> None:
    example_path = conf.model_hub_examples_path("huggingface/token-classification")
    config = conf.load_config(os.path.join(example_path, "ner_config.yaml"))
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_hparam(config, "use_apex_amp", True)
    config = set_docker_image(config)

    exp.run_basic_test_with_temp_config(config, example_path, 1)
