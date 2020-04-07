#!/bin/sh

# This script contains setup for miscellaneous repo-local settings that may be
# helpful for developers.

cd "$(git rev-parse --show-toplevel)"

git config --local commit.template .github/commit_template.txt
ln -s ../../scripts/commit-msg-hook.sh .git/hooks/commit-msg
