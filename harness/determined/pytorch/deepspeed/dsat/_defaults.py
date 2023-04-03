MODEL_INFO_PROFILING_PATH = "model_info.json"
AUTOTUNING_RESULTS_PATH = "autotuning_metric.json"

ZERO_STAGE = 0

SMALLER_IS_BETTER = True
USE_DSAT_MODE_KEY = "_use_dsat_mode"
GAS_DEFAULT = 1
OVERWRITE_KEY = "overwrite_deepspeed_args"

MODEL_INFO_PROFILING_DS_CONFIG = {
    "train_micro_batch_size_per_gpu": 1,
    "zero_optimization": {
        "stage": 1
    },  # Stage 3 gives the best chance for the model to successfully run without OOM and it's
    # what native DSAT uses in its model profiling run, but some features like MOE aren't available
    # with stage 3, so we just use 1. TODO: We should discuss.
    "autotuning": {
        "enabled": True,
        # The two fields below essentially use DS internals! Maybe fragile.
        "model_info_path": MODEL_INFO_PROFILING_PATH,
        "model_info": {"profile": True},
    },
}


# Using same defaults as DS. Written as a diff between successive stages for brevity.
NEW_ZERO_OPTIM_KEYS_AND_DEFAULTS_PER_STAGE = {
    0: dict(),
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

AUTOTUNING_DICT = {
    "enabled": True,
    "results_dir": "autotuning_results",
    "exps_dir": "autotuning_exps",
    "overwrite": False,
    "metric": "throughput",  # TODO: dynamically populate based on searcher fields.
    "start_profile_step": 3,
    "end_profile_step": 5,
    "fast": True,
    "max_train_batch_size": None,
    "mp_size": 1,
    "num_tuning_micro_batch_sizes": 5,
    "tuner_type": "model_based",
    "tuner_early_stopping": 25,
    "tuner_num_trials": 25,
    "arg_mappings": None,
}

DEFAULT_SEARCH_RUNNER_OVERRIDES = {
    "searcher": {"name": "single", "max_length": 0},
    # TODO: don't hardcode the searcher's max_restarts.
    "max_restarts": 3,
    # TODO: taking slots_per_trial: 0 to imply cpu-only here, but that's apparently an unsafe assumption
    # e.g. on Grenoble.
    "resources": {"slots_per_trial": 0},
    # TODO: remove the environment section; just needed for GG's GCP cluster.
    "environment": {
        "image": {
            "cpu": "determinedai/environments:cuda-11.3-pytorch-1.10-tf-2.8-deepspeed-0.7.0-gpu-0.20.1",
            "gpu": "determinedai/environments:cuda-11.3-pytorch-1.10-tf-2.8-deepspeed-0.7.0-gpu-0.20.1",
        }
    },
}
