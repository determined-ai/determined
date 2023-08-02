#!/usr/bin/env bash
set -xeuo pipefail

VERSION=$(cat VERSION)
GO_LDFLAGS="-X github.com/determined-ai/determined/master/version.Version=${VERSION}"

nodemon --watch './**/*' -e go --signal SIGTERM --exec \
    go run -ldflags "${GO_LDFLAGS}" \
    ./master/cmd/determined-master -- --log-level debug 2>&1
