import os
import pathlib
from typing import Any, Dict, List, Union

from determined.common import api, util

MASTER_SCHEME = "http"
MASTER_IP = "localhost"
MASTER_PORT = "8080"
DET_VERSION = None
USER_PASSWORD = ""
DEFAULT_MAX_WAIT_SECS = 1800
MAX_TASK_SCHEDULED_SECS = 30
MAX_TRIAL_BUILD_SECS = 90


DEFAULT_TF2_CPU_IMAGE = "determinedai/tensorflow-ngc-dev:0736b6d"
DEFAULT_TF2_GPU_IMAGE = "determinedai/tensorflow-ngc-dev:0736b6d"
DEFAULT_PT_CPU_IMAGE = "determinedai/pytorch-tensorflow-cpu-dev:0736b6d"
DEFAULT_PT_GPU_IMAGE = "determinedai/pytorch-tensorflow-cuda-dev:0736b6d"
DEFAULT_PT2_CPU_IMAGE = "determinedai/pytorch-ngc-dev:0736b6d"
DEFAULT_PT2_GPU_IMAGE = "determinedai/pytorch-ngc-dev:0736b6d"

TF2_CPU_IMAGE = os.environ.get("TF2_CPU_IMAGE") or DEFAULT_TF2_CPU_IMAGE
TF2_GPU_IMAGE = os.environ.get("TF2_GPU_IMAGE") or DEFAULT_TF2_GPU_IMAGE
PT_CPU_IMAGE = os.environ.get("PT_CPU_IMAGE") or DEFAULT_PT_CPU_IMAGE
PT_GPU_IMAGE = os.environ.get("PT_GPU_IMAGE") or DEFAULT_PT_GPU_IMAGE
PT2_CPU_IMAGE = os.environ.get("PT2_CPU_IMAGE") or DEFAULT_PT2_CPU_IMAGE
PT2_GPU_IMAGE = os.environ.get("PT2_GPU_IMAGE") or DEFAULT_PT2_GPU_IMAGE
GPU_ENABLED = os.environ.get("DET_TEST_GPU_ENABLED", "1") not in ("0", "false")

PROJECT_ROOT_PATH = pathlib.Path(__file__).resolve().parents[2]
EXAMPLES_PATH = PROJECT_ROOT_PATH / "examples"

SCIM_USERNAME = "determined"
SCIM_PASSWORD = "password"

ALL_NTSC = set(api.NTSC_Kind.__members__.values())
PROXIED_NTSC = {api.NTSC_Kind.notebook, api.NTSC_Kind.tensorboard}

DEFAULT_RM_CLUSTER_NAME = "defaultrm"
DEFAULT_RM_KUBECONFIG = "/tmp/defaultrm-kubeconf"
ADDITIONAL_RM_CLUSTER_NAME = "additionalrm"
ADDITIONAL_RM_KUBECONFIG = "/tmp/additionalrm-kubeconf"
DEFAULT_KUBECONFIG = "~/.kube/config"
DEFAULT_KUBERNETES_NAMESPACE = "default"


def fixtures_path(path: str) -> str:
    return os.path.join(os.path.dirname(__file__), "fixtures", path)


def tutorials_path(path: str) -> str:
    return os.path.join(os.path.dirname(__file__), "../../examples/tutorials", path)


def cv_examples_path(path: str) -> str:
    return os.path.join(os.path.dirname(__file__), "../../examples/computer_vision", path)


def diffusion_examples_path(path: str) -> str:
    return os.path.join(os.path.dirname(__file__), "../../examples/diffusion", path)


def features_examples_path(path: str) -> str:
    return os.path.join(os.path.dirname(__file__), "../../examples/features", path)


def deepspeed_examples_path(path: str) -> str:
    return os.path.join(os.path.dirname(__file__), "../../examples/deepspeed", path)


def deepspeed_autotune_examples_path(path: str) -> str:
    return os.path.join(os.path.dirname(__file__), "../../examples/deepspeed_autotune", path)


def hf_trainer_examples_path(path: str) -> str:
    return os.path.join(os.path.dirname(__file__), "../../examples/hf_trainer_api", path)


def load_config(config_path: str) -> Any:
    with open(config_path) as f:
        config = util.safe_load_yaml_with_exceptions(f)
    return config


def make_master_url() -> str:
    return api.canonicalize_master_url(f"{MASTER_SCHEME}://{MASTER_IP}:{MASTER_PORT}")


def set_global_batch_size(config: Dict[Any, Any], batch_size: int) -> Dict[Any, Any]:
    config = config.copy()
    config["hyperparameters"]["global_batch_size"] = batch_size
    return config


def set_slots_per_trial(config: Dict[Any, Any], slots: int) -> Dict[Any, Any]:
    config = config.copy()
    config.setdefault("resources", {})
    config["resources"]["slots_per_trial"] = slots
    return config


def set_max_length(
    config: Dict[Any, Any], max_length: Union[Dict[str, int], int]
) -> Dict[Any, Any]:
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


def set_tf2_image(config: Dict[Any, Any]) -> Dict[Any, Any]:
    return set_image(config, TF2_CPU_IMAGE, TF2_GPU_IMAGE)


def set_pt_image(config: Dict[Any, Any]) -> Dict[Any, Any]:
    return set_image(config, PT_CPU_IMAGE, PT_GPU_IMAGE)


def set_pt2_image(config: Dict[Any, Any]) -> Dict[Any, Any]:
    return set_image(config, PT2_CPU_IMAGE, PT2_GPU_IMAGE)


def set_random_seed(config: Dict[Any, Any], seed: int) -> Dict[Any, Any]:
    config = config.copy()
    config.setdefault("reproducibility", {})
    config["reproducibility"]["experiment_seed"] = seed
    return config


def set_hparam(config: Dict[Any, Any], name: str, value: Any) -> Dict[Any, Any]:
    config = config.copy()
    config["hyperparameters"][name] = {"type": "const", "val": value}
    return config


def set_pod_spec(config: Dict[Any, Any], pod_spec: Dict[Any, Any]) -> Dict[Any, Any]:
    config = config.copy()
    config.setdefault("environment", {})
    config["environment"]["pod_spec"] = pod_spec
    return config


def set_profiling_enabled(config: Dict[Any, Any]) -> Dict[Any, Any]:
    config = config.copy()
    config.setdefault("profiling", {})
    config["profiling"]["enabled"] = True
    return config


def set_entrypoint(config: Dict[Any, Any], entrypoint: str) -> Dict[Any, Any]:
    config = config.copy()
    config["entrypoint"] = entrypoint
    return config


def set_environment_variables(
    config: Dict[Any, Any], environment_variables: List[str]
) -> Dict[Any, Any]:
    config = config.copy()
    config["environment"]["environment_variables"] = environment_variables
    return config
