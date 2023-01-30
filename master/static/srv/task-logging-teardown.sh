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

# Wait for 30 seconds total for the logging to finish, otherwise just exit.
waitfor="${DET_LOG_WAIT_TIME:-30}"
deadline="$(($(epoch_seconds) + waitfor))"
timeout="$((deadline - $(epoch_seconds)))"

# read returns 1 on EOF or >128 with timeout, but it's a fifo so that is OK.
# For read's -t timeout feature to work, we need to open the fifo for
# reading and writing for some reason, which is what the `<>` is for.
# See https://stackoverflow.com/a/6448737.
read -N $DET_LOG_WAIT_COUNT -t "$timeout" <>"$DET_LOG_WAIT_FIFO" || true
