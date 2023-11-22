#!/bin/bash -ex

# check for dirty changes
if [[ -n "$(git status --porcelain)" ]]; then
    echo "untracked or dirty files are not allowed, cleanup before running insert-dropdown-url.sh"
    exit 1
fi

# get latest environments commit
export ENVS_HASH="$(git ls-remote https://github.com/determined-ai/environments.git -h HEAD -q | cut -f1)"
# update bumpenvs yaml with the given hash
python3 tools/scripts/update-bumpenvs-yaml.py tools/scripts/bumpenvs.yaml ENVS_HASH
# run the bumpenvs procedure
python3 tools/scripts/bumpenvs.py tools/scripts/bumpenvs.yaml

# check to see if bumpenvs.py published resulted in any file changes or not
if [[ -z "$(git status --porcelain)" ]]; then
    echo "no change to environment images found"
    exit 0
fi
git add --update
git commit -m "chore: bumpenvs to environments commit $ENVS_HASH"
