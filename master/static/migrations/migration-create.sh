#!/bin/bash

set -e

name="$1"

if [ -z "$name" ]; then
  echo "usage: $0 NAME"
  exit 1
fi

touch "$(date +%Y%m%d%H%M%S)_$name.tx."{up,down}".sql"
