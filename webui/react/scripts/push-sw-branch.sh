#!/bin/bash -ex

# used to facilitate testing changes originating in core
# in saas.

clone_dir=/tmp/shared
shared_web_url=https://github.com/determined-ai/shared-web
shared_dir="$(pwd)/src/shared"

# get current branch name
core_branch_name=$(git rev-parse --abbrev-ref HEAD)
core_hash=$(git rev-parse HEAD)

echo "you might want to run make push-shared to update the target repor first"

[ -d $clone_dir ] || git clone $shared_web_url $clone_dir

cd $clone_dir
git checkout main
git pull
git checkout -b ${core_branch_name}

cp -r $shared_dir/* $clone_dir
git add .
git commit -m "bring in changes from core/${core_branch_name}/${core_hash}"
