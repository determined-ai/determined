#!/usr/bin/env bash

MAX_RETRIES=${MAX_RETRIES:-3}

i=0
until "$@"; do
    if [[ $i -ge $MAX_RETRIES ]]; then
        echo "Command $1 failed after $i retries"
        exit 1
    fi
    delay=$((2 ** i))
    echo "Command $1 failed, retrying in $delay second(s)"
    sleep $delay
    ((i = i + 1))
done
