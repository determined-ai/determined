import os

import pytest

from tests import config as conf
from tests import experiment as exp


@pytest.mark.model_hub  # type: ignore
@pytest.mark.distributed  # type: ignore
def test_token_classification_ner() -> None:
    example_path = conf.model_hub_examples_path("huggingface/token-classification")
    config = conf.load_config(os.path.join(example_path, "ner_config.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(config, example_path, 1)
