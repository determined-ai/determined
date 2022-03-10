#!/bin/bash
"$DET_PYTHON_EXECUTABLE" -m determined.exec.prep_container --rendezvous

MIN_PERCENTAGE=10  # Minimum percentage GPU usage which triggers a failure.
SAMPLE_FREQ=5      # How frequently to sample the GPU usage.
NUM_SAMPLES=8      # Number of samples to look at when evaluating GPU usage.
DELAY_SAMPLES=2    # Number of samples to wait before evaluating GPU usage.
IDLE_DEBUG=true    # Print debug statements

IDLE_ARGS="$MIN_PERCENTAGE $SAMPLE_FREQ $NUM_SAMPLES $DELAY_SAMPLES $IDLE_DEBUG"

EXEC_ARGS="$DET_PYTHON_EXECUTABLE -m determined.exec.launch_autohorovod $@"
exec "$DET_PYTHON_EXECUTABLE" -m idle_gpu_watcher $IDLE_ARGS $EXEC_ARGS
