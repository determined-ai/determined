#!/bin/sh

set -e
set -x

docker info

if [ "$#" -ne 5 ] ; then
    echo "usage: $0 LOG_NAME BASE_TAG HASH VERSION ARTIFACTS_DIR" >&2
    exit 1
fi

log_name="$1"
base_tag="$2"
hash="$3"
version="$4"
artifacts="$5"

underscore_name="$(echo -n "$log_name" | tr - _)"

docker push "$base_tag-$hash"
docker push "$base_tag-$version"

mkdir -p "$artifacts"

log_file="$artifacts/publish-$log_name"
(
    echo "${underscore_name}_hashed: $base_tag-$hash"
    echo "${underscore_name}_versioned: $base_tag-$version"
) > "$log_file"
