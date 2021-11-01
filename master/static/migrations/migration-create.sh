#!/bin/bash

name=$1

if [ -z name ]; then
    echo "missing migration name"
    exit 1
fi

touch $(date +%Y%m%d%H%M%S)_${name}.{up,down}.sql
