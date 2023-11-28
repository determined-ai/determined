# Retags all docker images from latest Environments build
# bash tools/scripts/update-docker-tags.sh OLD_VERSION NEW_VERSION

# get current environments hash and parse version args
export OLD_VERSION="$1"
export NEW_VERSION="$2"

# get list of tags to replace via OLD_VERSION in bumpenvs.yaml
export IMAGES=`grep -o -P "(?<=new: ).*(?=,)" tools/scripts/bumpenvs.yaml | grep $OLD_VERSION`

# update tags on dockerhub
for OLD_TAG in $IMAGES
do
  NEW_TAG=`echo OLD_TAG | grep -o '.*-'`NEW_VERSION
  echo "Adding $OLD_TAG (clone of $NEW_TAG) to docker repo"
  #docker buildx create OLD_TAG --tag NEW_TAG
  echo "Replacing $OLD_TAG to $NEW_TAG in bumpenvs.yaml"
  perl 's/$OLD_TAG/$NEW_TAG/g' tools/scripts/bumpenvs.yaml
done
