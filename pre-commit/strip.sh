#!/bin/bash

# strip all TARGET string what matches PATTERN
# bash strip.sh abcb b -> ac

if [ $# -ne 2 ]; then
  echo "2 arguments are required"
  echo "usage: $0 'source string' 'pattern string'"
  exit 1
fi

SOURCE="$1"
PATTERN="$2"
echo "${SOURCE//"$PATTERN"}" # replace all. `//` means global
