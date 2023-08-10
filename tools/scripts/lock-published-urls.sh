#!/bin/bash -ex

# check for dirty changes
if [[ -n "$(git status --porcelain)" ]]; then
    echo "untracked or dirty files are not allowed, cleanup before running lock-published-urls.sh"
    exit 1
fi

# redirects.py inspects files to detect missing redirects, so generated files
# must be generated before the publish step
make -C docs attributions.rst

# lock in current published urls
python3 docs/redirects.py publish

# check to see if redirects.py published resulted in any file changes or not
if [[ -z "$(git status --porcelain)" ]]; then
    echo "no change to published urls"
    exit 0
fi
git add --update
git commit -m "chore: lock published urls to preserve redirects"
