import os
from typing import Any, Dict

from ruamel import yaml

MASTER_IP = "localhost"
MASTER_PORT = "8080"
DEFAULT_MAX_WAIT_SECS = 1800
MAX_TASK_SCHEDULED_SECS = 30
MAX_TRIAL_BUILD_SECS = 90


TF1_CPU_IMAGE = "determinedai/environments:py-3.6.9-pytorch-1.4-tf-1.14-cpu-90bf50b"
TF2_CPU_IMAGE = "determinedai/environments:py-3.6.9-pytorch-1.4-tf-2.1-cpu-90bf50b"
TF1_GPU_IMAGE = "determinedai/environments:cuda-10.0-pytorch-1.4-tf-1.14-gpu-90bf50b"
TF2_GPU_IMAGE = "determinedai/environments:cuda-10.1-pytorch-1.4-tf-2.1-gpu-90bf50b"


def fixtures_path(path: str) -> str:
    return os.path.join(os.path.dirname(__file__), "fixtures", path)


def official_examples_path(path: str) -> str:
    return os.path.join(os.path.dirname(__file__), "../../examples/official", path)


def experimental_path(path: str) -> str:
    return os.path.join(os.path.dirname(__file__), "../../examples/experimental", path)


def load_config(config_path: str) -> Any:
    with open(config_path) as f:
        config = yaml.safe_load(f)
    return config


def make_master_url(suffix: str = "") -> str:
    return "http://{}:{}/{}".format(MASTER_IP, MASTER_PORT, suffix)


def set_slots_per_trial(config: Dict[Any, Any], slots: int) -> Dict[Any, Any]:
    config = config.copy()
    config.setdefault("resources", {})
    config["resources"]["slots_per_trial"] = slots
    return config


def set_native_parallel(config: Dict[Any, Any], native_parallel: bool) -> Dict[Any, Any]:
    config = config.copy()
    config.setdefault("resources", {})
    config["resources"]["native_parallel"] = native_parallel
    return config


def set_max_steps(config: Dict[Any, Any], max_steps: int) -> Dict[Any, Any]:
    config = config.copy()
    config.setdefault("resources", {})
    config["searcher"]["max_steps"] = max_steps
    return config


def set_aggregation_frequency(config: Dict[Any, Any], aggregation_frequency: int) -> Dict[Any, Any]:
    config = config.copy()
    config.setdefault("optimizations", {})
    config["optimizations"]["aggregation_frequency"] = aggregation_frequency
    return config


def set_amp_level(config: Dict[Any, Any], amp_level: str) -> Dict[Any, Any]:
    config = config.copy()
    config.setdefault("optimizations", {})
    config["optimizations"]["mixed_precision"] = amp_level
    return config


def set_tensor_auto_tuning(config: Dict[Any, Any], auto_tune: bool) -> Dict[Any, Any]:
    config = config.copy()
    config.setdefault("optimizations", {})
    config["optimizations"]["auto_tune_tensor_fusion"] = auto_tune
    return config


def set_image(config: Dict[Any, Any], cpu_image: str, gpu_image: str) -> Dict[Any, Any]:
    config = config.copy()
    config.setdefault("environment", {})
    config["environment"]["image"] = {"cpu": cpu_image, "gpu": gpu_image}
    return config


def set_tf1_image(config: Dict[Any, Any]) -> Dict[Any, Any]:
    return set_image(config, TF1_CPU_IMAGE, TF1_GPU_IMAGE)


def set_tf2_image(config: Dict[Any, Any]) -> Dict[Any, Any]:
    return set_image(config, TF2_CPU_IMAGE, TF2_GPU_IMAGE)


def set_shared_fs_data_layer(config: Dict[Any, Any]) -> Dict[Any, Any]:
    config = config.copy()
    config["data_layer"] = {}
    config["data_layer"]["type"] = "shared_fs"
    return config


def set_s3_data_layer(config: Dict[Any, Any]) -> Dict[Any, Any]:
    config = config.copy()
    config["data_layer"] = {}
    config["data_layer"]["type"] = "s3"
    config["data_layer"]["bucket"] = "yogadl-test"
    config["data_layer"]["bucket_directory_path"] = "pedl_integration_tests"
    return config
