#!/bin/bash -ex

# check for dirty changes
if [[ -n "$(git status --porcelain)" ]]; then
    echo "untracked or dirty files are not allowed, cleanup before running insert-dropdown-url.sh"
    exit 1
fi

# insert-version-url.py inserts a new entry into the versions.json--adding a dropdown link
python3 docs/insert-version-url.py "$(grep -oE '^[0-9.]*' VERSION)"

git add --update
git commit -m "chore: add docs dropdown link for new version"
