DSAT_MAX_LENGTH_STEPS = 5
WORKDIR_PATH = "/run/determined/workdir/"
PROFILER_OUTPUT_FILE_PATH = WORKDIR_PATH + "flops_profiler_output.txt"
AUTOTUNING_MODEL_PROFILE_OUTPUT_FILE_PATH = WORKDIR_PATH + "model_info_profiling_results.json"

FLOPS_PROFILER_CONFIG = {
    "enabled": True,
    "profile_step": DSAT_MAX_LENGTH_STEPS - 1,
    "module_depth": -1,
    "top_modules": 10,
    "detailed": True,
    "output_file": PROFILER_OUTPUT_FILE_PATH,
}

DSAT_SEARCHER_RESOURCES_CONFIG = {
    "train_micro_batch_size_per_gpu": 1,
    "zero_optimization": {
        "stage": 0
    },  # DS set the stage to 3; not sure why? See DEFAULT_MIN_MEM_CONFIG
    "flops_profiler": FLOPS_PROFILER_CONFIG,
}
MODEL_INFO_PROFILING_DS_CONFIG = {
    "train_micro_batch_size_per_gpu": 1,
    "zero_optimization": {
        "stage": 0
    },  # DS set the stage to 3; not sure why? See DEFAULT_MIN_MEM_CONFIG
    "autotuning": {
        "enabled": True,
        # The two fields below essentially use DS internals!
        "model_info_path": AUTOTUNING_MODEL_PROFILE_OUTPUT_FILE_PATH,
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
