#!/usr/bin/env bash

CLEAR="\e[39m"
MAGENTA="\e[95m"
BLUE="\e[94m"

trap 'kill ${MASTER_PID}; exit' INT
../master/build/determined-master --config-file master.yaml \
    2>&1 | sed -e "s/^/$(printf "${MAGENTA}determined-master  | ${CLEAR}"  )/" &
MASTER_PID=$!

attempt_counter=0
max_attempts=10
until curl --output /dev/null --silent --head http://localhost:8080; do
    if [[ ${attempt_counter} -eq ${max_attempts} ]];then
        echo "Max attempts reached"
        exit 1
    fi
    attempt_counter=$((attempt_counter+1))
    sleep 2
done

../agent/build/determined-agent run --config-file agent.yaml \
    2>&1 | sed -e "s/^/$(printf "${BLUE}determined-agent   | ${CLEAR}")/" &
AGENT_PID=$!

wait ${MASTER_PID} ${AGENT_PID}
