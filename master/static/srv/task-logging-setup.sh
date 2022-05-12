#!/usr/bin/env bash

STDOUT_FILE=/run/determined/train/logs/stdout.log
STDERR_FILE=/run/determined/train/logs/stderr.log

mkdir -p "$(dirname "$STDOUT_FILE")" "$(dirname "$STDERR_FILE")"

# Create symbolic links from well-known files to this process's STDOUT and
# STDERR. Anything written to those files will be inserted into the output
# streams of this process, allowing distributed training logs to route through
# individual containers rather than all going through SSH back to agent 0.
ln -sf /proc/$$/fd/1 "$STDOUT_FILE"
ln -sf /proc/$$/fd/2 "$STDERR_FILE"

if [ -n "$DET_K8S_LOG_TO_FILE" ]; then
	# To do logging with a sidecar in Kubernetes, we need to log to files that
	# can then be read from the sidecar. To avoid a disk explosion, we need to
	# layer on some rotation. multilog is a tool that automatically writes its
	# stdin to rotated log files; the following line pipes stdout and stderr of
	# this process to separate multilog invocations. "n2" means to only store
	# one old log file -- the logs are being streamed out by Fluent Bit, so we
	# don't need to keep any more old ones around. Create the dirs ahead of time
	# so they are 0755 (when they don't exist, multilog makes them 0700 and
	# Fluent Bit can't access them with the non-root user).
	STDOUT_ROTATE_DIR="$STDOUT_FILE-rotate"
	STDERR_ROTATE_DIR="$STDERR_FILE-rotate"
	mkdir -p -m 755 $STDOUT_ROTATE_DIR
	mkdir -p -m 755 $STDERR_ROTATE_DIR

	# Create a fifo to monitor process substitution exits, and a count to know how many to wait on.
	DET_LOG_WAIT_FIFO=/run/determined/train/logs/wait.fifo
	DET_LOG_WAIT_COUNT=0
	mkfifo $DET_LOG_WAIT_FIFO

	# We save the original stdout and stderr. These process substitions block until their stdin
	# is closed and, when we clean up, by saving these we can close them safely and replace stdout
	# and stderr for the shell with the original.
	exec {ORIGINAL_STDOUT}>&1 1> >(
		multilog n2 "$STDOUT_ROTATE_DIR"
		: >$DET_LOG_WAIT_FIFO
	) \
	{ORIGINAL_STDERR}>&2 2> >(
		multilog n2 "$STDERR_ROTATE_DIR"
		: >$DET_LOG_WAIT_FIFO
	)

	((DET_LOG_WAIT_COUNT += 2))
fi
