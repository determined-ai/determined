WORKDIR_PATH = "/run/determined/workdir/"
DS_PROFILER_OUTPUT_PATH = WORKDIR_PATH + "flops_profiler_output.txt"
MODEL_INFO_PROFILING_PATH = WORKDIR_PATH + "model_info_profiling_results.json"
AUTOTUNING_RESULTS_DIR_PATH = WORKDIR_PATH + "autotuning_results"
AUTOTUNING_EXP_DIR_PATH = WORKDIR_PATH + "autotuning_exps"
AUTOTUNING_RESULTS_PATH = "autotuning_metric.json"

END_PROFILE_STEP = 5

MODEL_INFO_PROFILING_DS_CONFIG = {
    "train_micro_batch_size_per_gpu": 1,
    "zero_optimization": {
        "stage": 0
    },  # DS set the stage to 3; not sure why? See DEFAULT_MIN_MEM_CONFIG. Verify not crucial.
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


OOM_KEY = "OOM"
