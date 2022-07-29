import os

# The maximum number of slots we expect any agent to have. Since we offset some ports
# by the min(device_id) belonging to the trial, if we have two ports offset in this way,
# we separate them by the max(min(device_id)) to avoid collisions. The two rendezvous ports
# are examples of this.
MAX_SLOTS_PER_AGENT = 16

# The default configs to use in when running test experiments.
#
# TODO: Unify the defaults used here with the defaults used in master.
DEFAULT_SEARCHER_CFG = {"name": "single", "max_length": {"batches": 100}}
DEFAULT_RESOURCES_CFG = {"slots_per_trial": 1, "native_parallel": False}
DEFAULT_SCHEDULING_UNIT = 100
DEFAULT_OPTIMIZATIONS = {
    "aggregation_frequency": 1,
    "average_aggregated_gradients": True,
    "average_training_metrics": True,
    "gradient_compression": False,
    "mixed_precision": "O0",
}
DEFAULT_EXP_CFG = {
    "searcher": DEFAULT_SEARCHER_CFG,
    "scheduling_unit": DEFAULT_SCHEDULING_UNIT,
    "resources": DEFAULT_RESOURCES_CFG,
    "optimizations": DEFAULT_OPTIMIZATIONS,
}

# Until we implement a more automatic solution, expose a temporary workaround of
# allowing ports to be changed using envionment variables for the rare case that
# the default ports are already in use by other processes.

# TODO (DET-1189): Use port registry to allocate ssh port.
# SSH port used for agents during dtrain (currently used with horovod and deepspeed backend).
DTRAIN_SSH_PORT = int(str(os.getenv("DTRAIN_SSH_PORT", "12350")))

# GLOO port used by Horovod for the Gloo controller.
HOROVOD_GLOO_RENDEZVOUS_PORT = int(str(os.getenv("HOROVOD_GLOO_RENDEZVOUS_PORT", "12355")))

# Port for communicating between training processes. Used for reducing
# validation metrics.
INTER_TRAIN_PROCESS_COMM_PORT_1 = int(str(os.getenv("INTER_TRAIN_PROCESS_COMM_PORT_1", "12360")))

INTER_TRAIN_PROCESS_COMM_PORT_2 = INTER_TRAIN_PROCESS_COMM_PORT_1 + MAX_SLOTS_PER_AGENT

# How many seconds horovod waits for startup to complete before failing.
HOROVOD_STARTUP_TIMEOUT_SECONDS = 1200

# Path for file that stores output of horovod auto-tuning. Only created when
# horovod auto-tuning is enabled.
HOROVOD_AUTOTUNE_LOG_FILEPATH = "/tmp/autotune_log.csv"

# How many seconds GLOO waits for all tasks to connect before failing.
# Increasing this from a default of 30 is necessary when there is a
# large number of machines.
HOROVOD_GLOO_TIMEOUT_SECONDS = 240

# The well-known locations of the executing container's STDOUT and STDERR.
CONTAINER_STDOUT = "/run/determined/train/logs/stdout.log"
CONTAINER_STDERR = "/run/determined/train/logs/stderr.log"

MANAGED_TRAINING_MODEL_COPY = "/run/determined/train/model"
