from typing import Dict, List, Union

ALL_SEARCH_METHOD_NAMES = ["binary", "_test", "asha", "random"]

MODEL_INFO_PROFILING_PATH = "model_info.json"
AUTOTUNING_RESULTS_PATH = "autotuning_metric.json"
SMALLER_IS_BETTER = True
USE_DSAT_MODE_KEY = "_use_dsat_mode"
GAS_DEFAULT = 1
CONFIG_KEY = "deepspeed_config"
OVERWRITE_KEY = "overwrite_deepspeed_args"
ARGS_PKL_PATH = "args.pkl"

SMALLER_IS_BETTER_METRICS = ["forward", "backward", "latency"]
LARGER_IS_BETTER_METRICS = ["throughput", "FLOPS_per_gpu"]

# Native DS AT uses the below settings for the model info profile run, but also with the the stage
# set to 3, presumably since that gives a general model the best chance to run without OOMing.
# However, since some model cannot run with stage 3, we do not enforce that choice here and the
# zero configuration in the submitted deepspeed config will be used.
MODEL_INFO_PROFILE_DS_CONFIG = {
    "train_micro_batch_size_per_gpu": 1,
    "autotuning": {
        "enabled": True,
        # The two fields below essentially use DS internals! Maybe fragile.
        "model_info_path": MODEL_INFO_PROFILING_PATH,
        "model_info": {"profile": True},
    },
}


# Using similar. Written as a diff between successive stages for brevity.
reduce_bucket_size_defaults = [n * 10**m for n in (1, 5) for m in range(6, 10)]
allgather_bucket_size_defaults = [n * 10**m for n in (1, 5) for m in range(6, 10)]

DEFAULT_ZERO_SEARCH_SPACE: Dict[int, Dict[str, List[Union[bool, float]]]] = {
    0: {},
    1: {
        "reduce_bucket_size": reduce_bucket_size_defaults,
        "allgather_bucket_size": allgather_bucket_size_defaults,
    },
    2: {
        "overlap_comm": [True, False],
        "reduce_scatter": [True, False],
        "contiguous_gradients": [True, False],
    },
    3: {
        "allgather_partitions": [True, False],
    },
}

AUTOTUNING_DICT = {"autotuning": {"enabled": True}}

AUTOTUNING_ARG_DEFAULTS = {
    "max-trials": 64,
    "max-concurrent-trials": 16,
    "zero-stages": [1, 2, 3],
    "trials-per-random-config": 5,
    "start-profile-step": 3,
    "end-profile-step": 5,
    "metric": "FLOPS_per_gpu",
    "random-seed": 42,
    "run-full-experiment": False,
    "search-range-factor": 1.0,
    "divisor": 2,
    "min-binary-search-trials": 3,
    "max-rungs": 5,
    "asha-early-stopping": 0,
}

DEFAULT_SEARCH_RUNNER_CONFIG = {
    "searcher": {"name": "single", "max_length": 0},
    "max_restarts": 5,
    "resources": {"slots_per_trial": 0},
    "entrypoint": "python3 -m determined.pytorch.dsat._run_dsat",
}
