#!/bin/bash

attempt_counter=0
max_attempts=10
until [ "$(curl -s localhost:8080/agents | jq '. | length')" -eq 1 ]; do
  if [ ${attempt_counter} -eq ${max_attempts} ]; then
    echo "Max attempts reached"
    exit 1
  fi
  printf '.'
  attempt_counter=$((attempt_counter + 1))
  sleep 2
done
