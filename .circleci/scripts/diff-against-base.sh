#!/bin/bash 

set -ex

# if upstream doen't exist add it
if ! git remote | grep upstream > /dev/null; then
  git remote add upstream https://github.com/determined-ai/determined
fi
git fetch upstream
MERGE_BASE="$(git merge-base upstream/master HEAD)"
git diff --no-commit-id --name-only "$MERGE_BASE" HEAD
