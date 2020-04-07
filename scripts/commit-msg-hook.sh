#!/bin/bash

set -o pipefail
if ! "$(go env GOPATH)"/bin/conform enforce --commit-msg-file "$1" 2>&1 | sed 's|\(.*FAILED.*\)|\x1b[31m\1\x1b[m|'; then
    echo -e "\n\x1b[33mCommit message failed check; run

    git commit -eF \"$1\"

from the repo root to retry editing the message.\x1b[m"
    exit 1
fi
