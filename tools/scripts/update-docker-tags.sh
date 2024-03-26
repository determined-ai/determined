#!/bin/bash -ex
# Retags all docker images from latest Environments build
# bash tools/scripts/update-docker-tags.sh OLD_VERSION NEW_VERSION

# check for dirty changes
if [[ -n "$(git status --porcelain)" ]]; then
    echo "untracked or dirty files are not allowed, cleanup before running update-docker-tags.sh"
    exit 1
fi

# parse version args
export OLD_VERSION="$1"
export NEW_VERSION="$2"

# get list of tags to replace via OLD_VERSION in bumpenvs.yaml
export IMAGES=$(grep -o -P "(?<=new: ).*(?=,)" tools/scripts/bumpenvs.yaml | grep $OLD_VERSION)

# update tags on dockerhub
for OLD_TAG in $IMAGES; do
    NEW_TAG="$(echo $OLD_TAG | grep -o '.*-')$NEW_VERSION"
    echo "Adding $NEW_TAG (clone of $OLD_TAG) to docker repo"
    docker buildx create OLD_TAG --tag NEW_TAG
    echo "Replacing $OLD_TAG with $NEW_TAG in bumpenvs.yaml"
    sed -i -e "s@$OLD_TAG@$NEW_TAG@" tools/scripts/bumpenvs.yaml
done

# check to see if redirects.py published resulted in any file changes or not
if [[ -z "$(git status --porcelain)" ]]; then
    echo "no change to published urls, is the proper VERSION being passed?"
    exit 1
fi
git add --update
git commit -m "chore: bump current environment image versions to $NEW_VERSION"
