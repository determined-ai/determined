#!/usr/bin/env bash

set -x

# Warning: this script is not meant to be ran directly. It is invoked by 'make build'.

# This part of the script ensures that an HPC Launcher installation file is available in the build directory
# to be installed into the generated boot image.  If one does not exist, the latest release launcher
# is downloaded from an HPE internal registry.

# Base URL of the hpe-hpc-launcher release tree to download from if necessary
ARTIFACT_BASE_URL=https://arti.hpc.amslabs.hpecorp.net/artifactory/analytics-misc-stable-local/release/

# Checks the build directory for any debian files. If there is no launcher debians,
# the latest launcher version is downloaded. Otherwise, the debian in build/ is used
CURRENT_VERSION=$(ls build/ | grep hpe-hpc-launcher | grep .deb)
LATEST_VERSION=$(curl -X GET $ARTIFACT_BASE_URL | sed 's/<[^>]*>//g' | grep "^[1-9]" | tail -n 1 | cut -d/ -f1)

if [ -n "$CURRENT_VERSION" ]; then
    # If current version exists and it's not the latest version, prompt the user for action
    if [ "$CURRENT_VERSION" != "$LATEST_VERSION" ]; then
        read -p "Your launcher version is out of date. Do you want to overwrite the outdated version with the new version? (y/n): " answer
        case ${answer:0:1} in
            y | Y)
                echo >&2 "INFO: Downloading hpe-hpc-launcher_${LATEST_VERSION}.deb"
                rm -rf build/$CURRENT_VERSION
                wget -P build/ $ARTIFACT_BASE_URL$LATEST_VERSION/rocky_9_0/${LATEST_VERSION: -1}-0_amd64/hpe-hpc-launcher_$LATEST_VERSION-0_amd64.deb
                CURRENT_VERSION=$(ls build/ | grep hpe-hpc-launcher | grep .deb)
                ;;
            *)
                echo >&2 "INFO: Using existing ${CURRENT_VERSION}"
                ;;
        esac
    else
        echo >&2 "INFO: Using existing ${CURRENT_VERSION}"
    fi
elif [ -z "$CURRENT_VERSION" ]; then
    # Runs a curl command that sorts all of the versions on artifactory and chooses the latest one
    echo >&2 "INFO: Downloading hpe-hpc-launcher_${LATEST_VERSION}.deb"
    wget -P build/ $ARTIFACT_BASE_URL$LATEST_VERSION/rocky_9_0/${LATEST_VERSION: -1}-0_amd64/hpe-hpc-launcher_$LATEST_VERSION-0_amd64.deb
    CURRENT_VERSION=$(ls build/ | grep hpe-hpc-launcher | grep .deb)
fi

# This part of the script sets the workload manager as specified by the user
# (either slurm or pbs) and updates the image specifications accordingly.

WORKLOAD_MANAGER="slurm"
SOURCE_IMAGE_PROJECT_ID="schedmd-slurm-public"
SOURCE_IMAGE_FAMILY="schedmd-v5-slurm-22-05-8-ubuntu-2204-lts"

# Only one argument (predefined) will ever be passed in so this should be okay
if [[ $1 == "pbs" ]]; then
    WORKLOAD_MANAGER="pbs"
    SOURCE_IMAGE_PROJECT_ID="ubuntu-os-cloud"
    SOURCE_IMAGE_FAMILY="ubuntu-2204-lts"
fi

echo >&2 "INFO: Using ${WORKLOAD_MANAGER} as a workload manager"
echo >&2 "INFO: Using ${SOURCE_IMAGE_PROJECT_ID} as source image"
echo >&2 "INFO: Using image from family ${SOURCE_IMAGE_FAMILY}"

# Other predefined variables

SSH_USERNAME="packer2"
CPU_IMAGE_NAME=$(grep "CPUImage" ../../../master/pkg/schemas/expconf/const.go | awk -F'\"' '{print $2}')

cat <<EOF
ssh_username           = "${SSH_USERNAME}"
workload_manager       = "${WORKLOAD_MANAGER}"
image_project_id       = "${SOURCE_IMAGE_PROJECT_ID}"
image_family           = "${SOURCE_IMAGE_FAMILY}"
launcher_deb_name      = "${CURRENT_VERSION}"
cpu_image_name         = "${CPU_IMAGE_NAME}"
EOF
