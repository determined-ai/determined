import os
import subprocess
from typing import Dict

import pytest

from tests import config as conf
from tests import experiment as exp


def set_docker_image(config: Dict) -> Dict:
    git_short_hash = (
        subprocess.check_output(["git", "rev-parse", "--short", "HEAD"]).strip().decode("utf-8")
    )

    config = conf.set_image(
        config, conf.TF1_CPU_IMAGE, f"determinedai/model-hub-mmdetection:{git_short_hash}"
    )
    return config


@pytest.mark.model_hub_mmdetection_quarantine
def test_maskrcnn_distributed_fake() -> None:
    example_path = conf.fixtures_path("mmdetection")
    config = conf.load_config(os.path.join(example_path, "distributed_fake_data.yaml"))
    config = conf.set_max_length(config, {"batches": 200})
    config = set_docker_image(config)

    exp.run_basic_test_with_temp_config(config, example_path, 1)


@pytest.mark.model_hub_mmdetection
def test_fasterrcnn_distributed_fake() -> None:
    example_path = conf.fixtures_path("mmdetection")
    config = conf.load_config(os.path.join(example_path, "distributed_fake_data.yaml"))
    config = conf.set_max_length(config, {"batches": 200})
    config = set_docker_image(config)
    config = conf.set_hparam(
        config, "config_file", "/mmdetection/configs/faster_rcnn/faster_rcnn_r50_fpn_1x_coco.py"
    )

    exp.run_basic_test_with_temp_config(config, example_path, 1)


@pytest.mark.model_hub_mmdetection
def test_retinanet_distributed_fake() -> None:
    example_path = conf.fixtures_path("mmdetection")
    config = conf.load_config(os.path.join(example_path, "distributed_fake_data.yaml"))
    config = conf.set_max_length(config, {"batches": 200})
    config = set_docker_image(config)
    config = conf.set_hparam(
        config, "config_file", "/mmdetection/configs/retinanet/retinanet_r50_fpn_1x_coco.py"
    )

    exp.run_basic_test_with_temp_config(config, example_path, 1)


@pytest.mark.model_hub_mmdetection
def test_gfl_distributed_fake() -> None:
    example_path = conf.fixtures_path("mmdetection")
    config = conf.load_config(os.path.join(example_path, "distributed_fake_data.yaml"))
    config = conf.set_max_length(config, {"batches": 200})
    config = set_docker_image(config)
    config = conf.set_hparam(
        config, "config_file", "/mmdetection/configs/gfl/gfl_r50_fpn_1x_coco.py"
    )

    exp.run_basic_test_with_temp_config(config, example_path, 1)


@pytest.mark.model_hub_mmdetection_quarantine
def test_yolo_distributed_fake() -> None:
    example_path = conf.fixtures_path("mmdetection")
    config = conf.load_config(os.path.join(example_path, "distributed_fake_data.yaml"))
    config = conf.set_max_length(config, {"batches": 200})
    config = set_docker_image(config)
    config = conf.set_hparam(
        config, "config_file", "/mmdetection/configs/yolo/yolov3_d53_320_273e_coco.py"
    )

    exp.run_basic_test_with_temp_config(config, example_path, 1)


@pytest.mark.model_hub_mmdetection
def test_detr_distributed_fake() -> None:
    example_path = conf.fixtures_path("mmdetection")
    config = conf.load_config(os.path.join(example_path, "distributed_fake_data.yaml"))
    config = conf.set_max_length(config, {"batches": 200})
    config = set_docker_image(config)
    config = conf.set_hparam(
        config, "config_file", "/mmdetection/configs/detr/detr_r50_8x2_150e_coco.py"
    )

    exp.run_basic_test_with_temp_config(config, example_path, 1)
