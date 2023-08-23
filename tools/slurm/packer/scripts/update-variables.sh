#!/bin/bash
#
#  This script grabs the image name from the recent packer build and automatically
#  updates ../terraform/variables with the image name. Whenever `make slurmcluster`
#  is invoked, the new image will then be used.
#
# Grabs the value from the artifact_id key in manifest.json
IMAGE_NAME=$(grep -o '"artifact_id": "[^"]*' build/manifest.json | grep -o '[^"]*$' | tr -d '\n')
# Checks for an image name. If none, then the build failed
if [ -z "$IMAGE_NAME" ]; then
    echo >&2 "ERROR: Unable to determine artifact_id from tools/slurm/packer/build/manifest.json"
    exit 1
fi
# Deletes the manifest.json file in build directory
rm -f build/manifest.json
# The sed command replaces the image name in images.conf according to the workload manager
# specified (passed into this script as an environment variable) with the new image that was built
# (either slurm or pbs). This removes the need to manually replace it.
if [ "$WORKLOAD_MANAGER" == "slurm" ]; then
    sed -i "" "/^slurm:/ s/:.*/: $IMAGE_NAME/" ../terraform/images.conf
elif [ "$WORKLOAD_MANAGER" == "pbs" ]; then
    sed -i "" "/^pbs:/ s/:.*/: $IMAGE_NAME/" ../terraform/images.conf
else
    echo "ERROR: No $WORKLOAD_MANAGER property in ../terraform/images.conf" >&2
    exit 1
fi

# List images
max_images=10
image_list=$(gcloud compute images list --filter=family:det-environments-slurm-ci --format="value(NAME, creationTimestamp, description, labels)" --sort-by=~creationTimestamp)
active_images=$(echo $image_list | wc -l)
echo >&2 "Existing Images"
echo >&2 "==============="
echo >&2 "$image_list"
echo >&2
echo >&2 "Consider pruning the $((active_images - max_images)) oldest images not in use by a release branch."
