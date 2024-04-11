#!/bin/bash -ex
# Retags all docker images from latest Environments build
# tools/scripts/update-docker-tags.sh NEW_VERSION [--release]

if [ "$#" -lt 1 ] || [ "$2" != "--release" ] || [ "$#" -gt 2 ]; then
    echo "usage: $0 NEW_VERSION [--release]" >&2
    exit 1
fi

# check for dirty changes
if [[ -n "$(git status --porcelain)" ]]; then
    echo "untracked or dirty files are not allowed, cleanup before running update-docker-tags.sh"
    exit 1
fi

# parse tag args
export OLD_TAG=$(cat tools/scripts/environments-target.txt)
export NEW_TAG="$1"

# get list of images to replace via OLD_TAG in bumpenvs.yaml
export IMAGES=$(grep -oP "(?<=new: ).*(?=,)" tools/scripts/bumpenvs.yaml | grep -F $OLD_TAG)

# update tags on dockerhub
for NAME in $IMAGES; do
    if [ "$2" == "--release" ]; then
        export NEW_NAME="$(echo $NAME | grep -oP '.*(?=-dev:)'):$NEW_TAG"
    else
        export NEW_NAME="$(echo $NAME | grep -o '.*:')$NEW_TAG"
    fi
    echo "Adding $NEW_NAME (clone of $NAME) to docker repo"
    docker buildx imagetools create $NAME --tag $NEW_NAME
done

# update environments-target.txt
echo $NEW_TAG >tools/scripts/environments-target.txt

# bumpenvs
echo "Updating bumpenvs.yaml"
if [ "$2" == "--release" ]; then
    python tools/scripts/retag-bumpenvs-yaml.py tools/scripts/bumpenvs.yaml $OLD_TAG $NEW_TAG --release
else
    python tools/scripts/retag-bumpenvs-yaml.py tools/scripts/bumpenvs.yaml $OLD_TAG $NEW_TAG
fi
echo "Performing bumpenvs"
python -m tools/scripts/bumpenvs.py tools/scripts/bumpenvs.yaml

# check to see if update-docker-tags.py resulted in any file changes or not
if [[ -z "$(git status --porcelain)" ]]; then
    echo "no change to files, is the proper tag being passed?"
    exit 1
fi
git add --update
git commit -m "chore: bump current environment image versions to $NEW_TAG"
