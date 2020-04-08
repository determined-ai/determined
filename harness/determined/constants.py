from pathlib import Path

# The container path which we use as a working directory.
WORKDIR_CONTAINER_PATH = "/run/determined/workdir"

# The port to use (from the container's point of view; it will be mapped to
# some arbitrary port on the host) for communication across containers. Must
# match LocalRendezvousPort in master/internal/trial.go.
LOCAL_RENDEZVOUS_PORT = 1734

# The default docker repository for task environments.
TASK_ENV_REPO = "determinedai/task-environment"

# The path used to serialize the training process environment variables
# inside the trial container.
TRAIN_PROCESS_ENVIRONMENT_VARIABLE_PATH = Path("/tmp/det_train_process_env.json")

# The default configs to use in the Determined Native API.
DEFAULT_SEARCHER_CFG = {"name": "single", "max_steps": 1}
DEFAULT_RESOURCES_CFG = {"slots_per_trial": 1, "native_parallel": False}
DEFAULT_BATCHES_PER_STEP = 100
DEFAULT_OPTIMIZATIONS = {
    "aggregation_frequency": 1,
    "average_aggregated_gradients": True,
    "gradient_compression": False,
    "mixed_precision": "O0",
}
DEFAULT_EXP_CFG = {
    "searcher": DEFAULT_SEARCHER_CFG,
    "batches_per_step": DEFAULT_BATCHES_PER_STEP,
    "resources": DEFAULT_RESOURCES_CFG,
    "optimizations": DEFAULT_OPTIMIZATIONS,
}

# TODO (DET-1189): Use port registry to allocate ssh port.
# SSH port used by Horovod.
HOROVOD_SSH_PORT = 12350

# GLOO port used by Horovod for the Gloo controller.
HOROVOD_GLOO_RENDEZVOUS_PORT = 12355

# Port for communicating between training processes. Used for reducing
# validation metrics.
INTER_TRAIN_PROCESS_COMM_PORT = 12360

# Default trial runner interface. For distributed training this
# specifies that the network interface must be auto-detected.
AUTO_DETECT_TRIAL_RUNNER_NETWORK_INTERFACE = "DET_AUTO_DETECT_NETWORK_INTERFACE"

# The key of user's experiment in the globals that retrieved by running users' code.
EXPERIMENT_GLOBALS_KEY = "__det__experiment"

# How many seconds horovod waits for startup to complete before failing.
HOROVOD_STARTUP_TIMEOUT_SECONDS = 1200

# Path for file that stores output of horovod auto-tuning. Only created when
# horovod auto-tuning is enabled.
HOROVOD_AUTOTUNE_LOG_FILEPATH = "/tmp/autotune_log.csv"
