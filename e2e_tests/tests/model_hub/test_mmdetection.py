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
        config, conf.TF1_CPU_IMAGE, f"determinedai/model-hub-mmdetection:{git_short_hash}"
    )
    return config


@pytest.mark.model_hub_mmdetection  # type: ignore
def test_maskrcnn_fake_data() -> None:
    example_path = conf.fixtures_path("mmdetection")
    config = conf.load_config(os.path.join(example_path, "fake_data.yaml"))
    config = conf.set_max_length(config, {"batches": 200})
    config = set_docker_image(config)

    exp.run_basic_test_with_temp_config(config, example_path, 1)


@pytest.mark.model_hub_mmdetection  # type: ignore
def test_maskrcnn_distributed_fake() -> None:
    example_path = conf.fixtures_path("mmdetection")
    config = conf.load_config(os.path.join(example_path, "distributed_fake_data.yaml"))
    config = conf.set_max_length(config, {"batches": 200})
    config = set_docker_image(config)

    exp.run_basic_test_with_temp_config(config, example_path, 1)
