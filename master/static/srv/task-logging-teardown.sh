#!/usr/bin/env bash

# Replace overridden stdout and stderr with original and close them, since the
# command is finished.
exec >&1- >&2- 1>&$ORIGINAL_STDOUT 2>&$ORIGINAL_STDERR

# We use the bash builtin printf for getting the epoch time in seconds.
# This requires bash 4.2 (from 2011) and it depends on strftime(3) supporting
# the %s directive, which is not in posix.
epoch_seconds() {
    printf '%(%s)T\n' -1
}

# Wait for up to DET_LOG_WAIT_TIME seconds for the logging to finish
# At this point the launching entry point script has exited and we are
# waiting for the child logging processes to complete.  The child
# processes will exit when reaching EOF of the log stream they are
# processing, so in the normal case they terminate quickly and
# each stream processor writes a single character to the DET_LOG_WAIT_FIFO
# to indicate they are no longer waiting.
#
# This wait time it to handle the case when log stream procesors have
# not exited yet -- either becuase someone is still writing to the stream,
# or the DET_MASTER is not reachable and therefor we are slow in flushing
# the logs to the master.
#
# After this wait, the container entrypoint immediately exits and all
# processing within the container is SIGKILLed without any opportunity
# for any furhter processing, so avoiding a premature exit is important.
waitfor="${DET_LOG_WAIT_TIME:-30}"

deadline="$(($(epoch_seconds) + waitfor))"
timeout="$((deadline - $(epoch_seconds)))"

# read returns 1 on EOF or >128 with timeout, but it's a fifo so that is OK.
# For read's -t timeout feature to work, we need to open the fifo for
# reading and writing for some reason, which is what the `<>` is for.
# See https://stackoverflow.com/a/6448737.
read -N $DET_LOG_WAIT_COUNT -t "$timeout" <>"$DET_LOG_WAIT_FIFO" || true
