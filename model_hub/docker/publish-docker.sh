#!/bin/sh

set -e
set -x

docker info

if [ "$#" -ne 3 ] ; then
    echo "usage: $0 LOG_NAME TAG ARTIFACTS_DIR" >&2
    exit 1
fi

log_name="$1"
tag="$2"
artifacts="$3"

underscore_name="$(echo -n "$log_name" | tr - _)"

docker push "$tag"

mkdir -p "$artifacts"

log_file="$artifacts/publish-$log_name"
(
    echo "${underscore_name}"
) > "$log_file"
