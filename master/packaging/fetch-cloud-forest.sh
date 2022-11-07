#!/usr/bin/env bash

# Ensure growforest is present.
go install github.com/ryanbressler/CloudForest/growforest

# Load the whole go env.
# Not just process substitution because some versions of MacOS ship with older versions of bash, for
# which `source` only works with regular files.
# See https://lists.gnu.org/archive/html/bug-bash/2006-01/msg00018.html for detail.
source /dev/stdin <<<"$(go env)"

# Find the appropriate binary for packaging and output the path.
if [ "$GOOS" == "$GOHOSTOS" ] && [ "$GOARCH" == "$GOHOSTARCH" ]; then
    # Package is installed for local use.
    echo "$GOPATH/bin/growforest"
else
    echo "$GOPATH/bin/${GOOS}_${GOARCH}/growforest"
fi
