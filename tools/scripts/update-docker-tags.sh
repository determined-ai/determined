# Retags all docker images from latest Environments build
# bash tools/scripts/update-docker-tags.sh OLD_VERSION NEW_VERSION

# get current environments hash and parse version args
export OLD_VERSION="$1"
export NEW_VERSION="$2"
ENVS_HASH="$(git ls-remote https://github.com/determined-ai/environments.git -h HEAD -q | cut -f1)"

# get list of tags to replace via latest commit hash
export IMAGES=`wget -q -O - "https://hub.docker.com/v2/repositories/determinedai/environments/tags" | \
grep -o '"name": *"[^"]*' | grep -o '[^"]*$' | grep $OLD_VERSION`

# update tags on dockerhub
for IMAGE_TAG in $IMAGES
do
  NEW_TAG=`echo IMAGE_TAG | grep -o '.*-'`NEW_VERSION
  echo "Updating $IMAGE_TAG to $NEW_TAG"
  docker buildx create IMAGE_TAG --tag NEW_TAG
done
