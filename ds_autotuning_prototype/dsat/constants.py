DSAT_MAX_LENGTH_STEPS = 5
WORKDIR_PATH = "/run/determined/workdir/"
DS_PROFILER_OUTPUT_PATH = WORKDIR_PATH + "flops_profiler_output.txt"
MODEL_INFO_PROFILING_PATH = WORKDIR_PATH + "model_info_profiling_results.json"

FLOPS_PROFILER_CONFIG = {
    "enabled": True,
    "profile_step": DSAT_MAX_LENGTH_STEPS - 1,
    "module_depth": -1,
    "top_modules": 10,  # TODO: Verify that this is a reasonable value. Also let user config this whole section.
    "detailed": True,
    "output_file": DS_PROFILER_OUTPUT_PATH,
}


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

# TODO: Should remove all references to SINGLE_SEARCHER_CONFIG and make max_length
# available for user to define.
SINGLE_SEARCHER_CONFIG = {
    "name": "single",
    "max_length": DSAT_MAX_LENGTH_STEPS,
    "metric": "placeholder",
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
