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

echo $(date -u) "DET_LOG_WAIT_COUNT $DET_LOG_WAIT_COUNT"

# Wait for 30 seconds total for the logging to finish, otherwise just exit.
waitfor="${DET_LOG_WAIT_TIME:-42}"
deadline="$(($(epoch_seconds) + waitfor))"
for ((i = 0; i < DET_LOG_WAIT_COUNT; i++)); do
    timeout="$((deadline - $(epoch_seconds)))"
    echo $(date -u) "Timeout $timeout"    
    test "$timeout" -le 0 && break

    echo $(date -u) "STARTING reading"        
    # read returns 1 on EOF or >128 with timeout, but it's a fifo so that is OK.
    # For read's -t timeout feature to work, we need to open the fifo for
    # reading and writing for some reason, which is what the `<>` is for.
    # See https://stackoverflow.com/a/6448737.
    read -N 1 -t "$timeout" <>"$DET_LOG_WAIT_FIFO" || true
    echo $(date -u) "reading finished"            
done
