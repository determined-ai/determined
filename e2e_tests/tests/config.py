import os
from typing import Any, Dict

from determined_common import yaml

MASTER_SCHEME = "http"
MASTER_IP = "localhost"
MASTER_PORT = "8080"
DET_VERSION = None
DEFAULT_MAX_WAIT_SECS = 1800
MAX_TASK_SCHEDULED_SECS = 30
MAX_TRIAL_BUILD_SECS = 90


TF1_CPU_IMAGE = "determinedai/environments:py-3.6.9-pytorch-1.4-tf-1.15-cpu-0f2001a"
TF2_CPU_IMAGE = "determinedai/environments:py-3.6.9-pytorch-1.4-tf-2.2-cpu-0f2001a"
TF1_GPU_IMAGE = "determinedai/environments:cuda-10.0-pytorch-1.4-tf-1.15-gpu-0f2001a"
TF2_GPU_IMAGE = "determinedai/environments:cuda-10.1-pytorch-1.4-tf-2.2-gpu-0f2001a"


def fixtures_path(path: str) -> str:
    return os.path.join(os.path.dirname(__file__), "fixtures", path)


def tutorials_path(path: str) -> str:
    return os.path.join(os.path.dirname(__file__), "../../examples/tutorials", path)


def cv_examples_path(path: str) -> str:
    return os.path.join(os.path.dirname(__file__), "../../examples/computer_vision", path)


def nlp_examples_path(path: str) -> str:
    return os.path.join(os.path.dirname(__file__), "../../examples/nlp", path)


def nas_examples_path(path: str) -> str:
    return os.path.join(os.path.dirname(__file__), "../../examples/nas", path)


def meta_learning_examples_path(path: str) -> str:
    return os.path.join(os.path.dirname(__file__), "../../examples/meta_learning", path)


def gan_examples_path(path: str) -> str:
    return os.path.join(os.path.dirname(__file__), "../../examples/gan", path)


def decision_trees_examples_path(path: str) -> str:
    return os.path.join(os.path.dirname(__file__), "../../examples/decision_trees", path)


def data_layer_examples_path(path: str) -> str:
    return os.path.join(os.path.dirname(__file__), "../../examples/data_layer", path)


def load_config(config_path: str) -> Any:
    with open(config_path) as f:
        config = yaml.safe_load(f)
    return config


def make_master_url(suffix: str = "") -> str:
    return "{}://{}:{}/{}".format(MASTER_SCHEME, MASTER_IP, MASTER_PORT, suffix)


def set_slots_per_trial(config: Dict[Any, Any], slots: int) -> Dict[Any, Any]:
    config = config.copy()
    config.setdefault("resources", {})
    config["resources"]["slots_per_trial"] = slots
    return config


def set_max_length(config: Dict[Any, Any], max_length: Dict[str, int]) -> Dict[Any, Any]:
    config = config.copy()
    config["searcher"]["max_length"] = max_length
    return config


def set_min_validation_period(
    config: Dict[Any, Any], min_validation_period: Dict[str, int]
) -> Dict[Any, Any]:
    config = config.copy()
    config["min_validation_period"] = min_validation_period
    return config


def set_min_checkpoint_period(
    config: Dict[Any, Any], min_checkpoint_period: Dict[str, int]
) -> Dict[Any, Any]:
    config = config.copy()
    config["min_checkpoint_period"] = min_checkpoint_period
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


def set_random_seed(config: Dict[Any, Any], seed: int) -> Dict[Any, Any]:
    config = config.copy()
    config.setdefault("reproducibility", {})
    config["reproducibility"]["experiment_seed"] = seed
    return config


def set_perform_initial_validation(config: Dict[Any, Any], init_val: bool) -> Dict[Any, Any]:
    config = config.copy()
    config["perform_initial_validation"] = init_val
    return config


def set_pod_spec(config: Dict[Any, Any], pod_spec: Dict[Any, Any]) -> Dict[Any, Any]:
    config = config.copy()
    config.setdefault("environment", {})
    config["environment"]["pod_spec"] = pod_spec
    return config
