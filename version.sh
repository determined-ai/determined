#!/bin/bash

# Safety first.
# -e: exit at first non-zero error code
# -x: print commands as we execute them
# -o pipefail: exit if one part of a pipe fails
set -exo pipefail

# If VERSION is unset or the empty string, ""
if [ -z ${VERSION} ]; then
	# Grab version from git.
	echo -n "$(git describe --tags --always 2>/dev/null)" | sed -e 's/-/./g' | sed -e 's/v//g'
else
	# Use existing VERSION.
	echo -n "${VERSION}"
fi
