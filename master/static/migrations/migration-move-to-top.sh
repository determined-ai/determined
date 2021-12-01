#!/bin/bash

migration=$1

if [ -z migration ]; then
    echo "missing migration"
    exit 1
fi

seq_num=$(echo $migration | cut -d '_' -f1)
name=$(echo $migration | cut -d '_' -f2- | cut -d '.' -f1)
new_seq=$(date +%Y%m%d%H%M%S)

mv "${seq_num}_${name}.up.sql" "${new_seq}_${name}.up.sql" 
mv "${seq_num}_${name}.down.sql" "${new_seq}_${name}.down.sql" 
