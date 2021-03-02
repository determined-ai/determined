#!/usr/bin/env bash

# Install native version to ensure it's on PATH
GO111MODULE=off go install github.com/ryanbressler/CloudForest/growforest

# Ensure the Linux binary is installed regardless of OS
GO111MODULE=off GOOS=linux go install github.com/ryanbressler/CloudForest/growforest

# Find the Linux binary for packaging and output the path
if [ "$(go env GOOS)" == "linux" ]; then
  echo $(which growforest)
else
  echo $(dirname $(which growforest))/linux_amd64/growforest
fi
