#!/bin/bash -ex
# Retags all docker images from latest Environments build
# bash tools/scripts/update-docker-tags.sh OLD_VERSION NEW_VERSION [--release]

if [ "$#" -lt 2 ] || [ "$#" -gt 2 ] || [ "$#" -eq 3 ] && [ "$3" != "--release" ]; then
	echo "usage: $0 OLD_VERSION NEW_VERSION [--release]" >&2
	exit 1
fi

# check for dirty changes
if [[ -n "$(git status --porcelain)" ]]; then
	echo "untracked or dirty files are not allowed, cleanup before running update-docker-tags.sh"
	exit 1
fi

# parse tag args
export OLD_TAG="$1"
export NEW_TAG="$2"

# get list of images to replace via OLD_TAG in bumpenvs.yaml
export IMAGES=$(grep -oP "(?<=new: ).*(?=,)" tools/scripts/bumpenvs.yaml | grep $OLD_TAG)

# update tags on dockerhub
for NAME in $IMAGES; do
	if [ "$3" == "--release" ]; then
		export NEW_NAME="$(echo $NAME | grep -oP '.*(?=-dev:)'):$NEW_TAG"
	else
		export NEW_NAME="$(echo $NAME | grep -o '.*:')$NEW_TAG"
	fi
	echo "Adding $NEW_NAME (clone of $NAME) to docker repo"
	docker buildx imagetools create $NAME --tag $NEW_NAME
done

# bumpenvs
echo "Updating bumpenvs.yaml"
if [ "$3" == "--release" ]; then
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
