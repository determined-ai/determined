#!/bin/bash
#
#  This script grabs the image name from the recent packer build and automatically
#  updates ../terraform/variables with the image name. Whenever `make slurmcluster`
#  is invoked, the new image will then be used.
#
# Grabs the value from the artifact_id key in manifest.json
IMAGE_NAME=$(grep -o '"artifact_id": "[^"]*' build/manifest.json | grep -o '[^"]*$')
# Checks for an image name. If none, then the build failed
if [ -z "$IMAGE_NAME" ]; then
    echo >&2 "ERROR: Unable to determine artifact_id from tools/slurm/packer/build/manifest.json"
    exit 1
fi
# Deletes the manifest.json file in build directory
rm -f build/manifest.json
# The sed command replaces the image name in variables.tf with the new image that was built
# Removes the need to manually replace it.
sed -i "" "s/det-environments-slurm-ci-\(.*\)/"$IMAGE_NAME"\"/g" ../terraform/variables.tf
