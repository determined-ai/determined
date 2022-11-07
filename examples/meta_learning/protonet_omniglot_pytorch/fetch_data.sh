#!/bin/bash
#
# Source: https://github.com/alshedivat/meta-blocks/blob/master/benchmarks/omniglot/fetch_data.sh
# Fetch Omniglot.
#

OMNIGLOT_URL=https://raw.githubusercontent.com/brendenlake/omniglot/master/python

set -e

mkdir tmp
trap 'rm -r tmp' EXIT

if [ ! -d data ]; then
    mkdir data
fi

if [ ! -d data/omniglot ]; then
    mkdir tmp/omniglot
    for name in images_background images_evaluation; do
        echo "Fetching omniglot/$name ..."
        curl -# "$OMNIGLOT_URL/$name.zip" >"tmp/$name.zip"
        echo "Extracting omniglot/$name ..."
        unzip -q "tmp/$name.zip" -d tmp
        rm "tmp/$name.zip"
        mv tmp/$name/* tmp/omniglot
    done
    mv tmp/omniglot data/omniglot
fi
