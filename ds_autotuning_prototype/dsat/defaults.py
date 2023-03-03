WORKDIR_PATH = "/run/determined/workdir/"
MODEL_INFO_PROFILING_PATH = WORKDIR_PATH + "model_info.json"
AUTOTUNING_RESULTS_DIR_PATH = WORKDIR_PATH + "autotuning_results"
AUTOTUNING_EXP_DIR_PATH = WORKDIR_PATH + "autotuning_exps"
AUTOTUNING_RESULTS_PATH = "autotuning_metric.json"

END_PROFILE_STEP = 5
OOM_KEY = "OOM_message"
MP_SIZE = 1

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

AUTUTONING_DICT = {
    "enabled": False,
    "results_dir": "autotuning_results",
    "exps_dir": "autotuning_exps",
    "overwrite": False,
    "metric": "throughput",
    "start_profile_step": 3,
    "end_profile_step": 5,
    "fast": True,
    "max_train_batch_size": None,
    "mp_size": 1,
    "num_tuning_micro_batch_sizes": 3,
    "tuner_type": "model_based",
    "tuner_early_stopping": 5,
    "tuner_num_trials": 50,
    "arg_mappings": None,
}
