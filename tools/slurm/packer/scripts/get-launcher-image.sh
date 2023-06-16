#!/bin/bash
#
#  This script ensures that an HPC Launcher installation file is available in the build directory
#  to be installed into the generated boot image.  If one does not exist, the latest release launcher
#  is downloaded from an HPE internal registry.
#
# Outputs:
# This script must print the file name of the hpe-hpc-launcher*.deb installation file within the build directory.
#
# Base URL of the hpe-hpc-launcher release tree to download from if necessary
ARTIFACT_BASE_URL=https://arti.hpc.amslabs.hpecorp.net/artifactory/analytics-misc-stable-local/release/
# Checks the build directory for any debian files. If there is no launcher debians,
# the latest launcher version is downloaded. Otherwise, the debian in build/ is used
CURRENT_VERSION=$(ls build/ | grep hpe-hpc-launcher | grep .deb)
if [ -z "$CURRENT_VERSION" ]; then
    # Runs a curl command that sorts all of the versions on artifactory and chooses the latest one
    LATEST_VERSION=$(curl -X GET $ARTIFACT_BASE_URL | sed 's/<[^>]*>//g' | grep "^[1-9]" | tail -n 1 | cut -d/ -f1)
    echo >&2 "INFO: Downloading hpe-hpc-launcher_$(LATEST_VERSION).deb"
    wget -P build/ $ARTIFACT_BASE_URL$LATEST_VERSION/rocky_9_0/${LATEST_VERSION: -1}-0_amd64/hpe-hpc-launcher_$LATEST_VERSION-0_amd64.deb
    CURRENT_VERSION=$(ls build/ | grep hpe-hpc-launcher | grep .deb)
else
    echo >&2 "INFO: Using existing ${CURRENT_VERSION}"
fi
# Packer can extract variables from the command line so the launcher name is exported
# via echo. This fixes the need to have a static name that needs to be updated.
echo $CURRENT_VERSION
