from determined.pytorch.deepspeed.dsat import _dsat_search_method

ALL_SEARCH_METHOD_CLASSES = {
    "random": _dsat_search_method.RandomDSATSearchMethod,
    "simple": _dsat_search_method.SimpleDSATSearchMethod,
}

MODEL_INFO_PROFILING_PATH = "model_info.json"
AUTOTUNING_RESULTS_PATH = "autotuning_metric.json"
SMALLER_IS_BETTER = True
USE_DSAT_MODE_KEY = "_use_dsat_mode"
GAS_DEFAULT = 1
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


# Using same defaults as DS. Written as a diff between successive stages for brevity.
DEFAULT_ZERO_SEARCH_SPACE = {
    1: {"reduce_bucket_size": [5e7, 5e8, 1e9], "allgather_bucket_size": [5e7, 5e8, 1e9]},
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
    "tuner-type": "random",
    "max-trials": 50,
    "max-concurrent-trials": 16,
    "zero-stages": [1, 2, 3],
    "trials-per-random-config": 3,
    "start-profile-step": 3,
    "end-profile-step": 5,
    "deepspeed-config": "deepspeed_config",
    "metric": "throughput",
}

DEFAULT_SEARCH_RUNNER_CONFIG = {
    "searcher": {"name": "single", "max_length": 0},
    "max_restarts": 5,
    "resources": {"slots_per_trial": 0},
    "entrypoint": "python3 -m determined.pytorch.deepspeed.dsat._run_dsat",
}
