#!/bin/bash

set -e

name="$1"

# get the current max migration number
function get_max() {
    local max=0
    for f in *.tx.*.sql; do
        local num=$(echo $f | cut -d'_' -f1)
        if [ $num -gt $max ]; then
            max=$num
        fi
    done
    echo $max
}

if [ -z "$name" ]; then
    echo "usage: $0 NAME"
    echo "where NAME for '20200401000000_initial.up.sql' would be 'initial'"
    exit 1
fi

new_time=$(date +%Y%m%d%H%M%S)
new_full_name="${new_time}_$name"

cur_max=$(get_max)
if [ "$new_time" -le "$cur_max" ]; then
    # potentially a timezone difference
    echo "new migration $new_time is not greater than existing max $cur_max"
    exit 1
fi

mv *"_$name.tx.up.sql" "$new_full_name.tx.up.sql"
mv *"_$name.tx.down.sql" "$new_full_name.tx.down.sql"
