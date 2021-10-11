#!/usr/bin/env bash

# Ensure growforest is present.
go install github.com/ryanbressler/CloudForest/growforest

# Load the whole go env.
source <(go env)

# Find the appropriate binary for packaging and output the path.
if [ "$GOOS" == "$GOHOSTOS" ] && [ "$GOARCH" == "$GOHOSTARCH" ]; then
    # Package is installed for local use.
    echo "$GOPATH/bin/growforest"
else
    echo "$GOPATH/bin/${GOOS}_${GOARCH}/growforest"
fi
