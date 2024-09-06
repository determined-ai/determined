import os

MAX_SLOTS_PER_AGENT = 16

# Until we implement a more automatic solution, expose a temporary workaround of
# allowing ports to be changed using envionment variables for the rare case that
# the default ports are already in use by other processes.

DTRAIN_SSH_PORT = int(str(os.getenv("DTRAIN_SSH_PORT", "12350")))

# Port for communicating between training processes. Used for reducing
# validation metrics.
INTER_TRAIN_PROCESS_COMM_PORT_1 = int(str(os.getenv("INTER_TRAIN_PROCESS_COMM_PORT_1", "12360")))

INTER_TRAIN_PROCESS_COMM_PORT_2 = int(
    str(
        os.getenv(
            "INTER_TRAIN_PROCESS_COMM_PORT_2", INTER_TRAIN_PROCESS_COMM_PORT_1 + MAX_SLOTS_PER_AGENT
        )
    )
)
#  both of the above ports will be offset
#  (value that we get from the port offset registry) in distributed context.

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
