#!/bin/bash

set -e

name="$1"

if [ -z "$name" ]; then
    echo "usage: $0 NAME"
    echo "where NAME for '20200401000000_initial.up.sql' would be 'initial'"
    exit 1
fi

new_base="$(date +%Y%m%d%H%M%S)_$name"

mv *"_$name.tx.up.sql" "$new_base.tx.up.sql"
mv *"_$name.tx.down.sql" "$new_base.tx.down.sql"
