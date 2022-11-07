#!/bin/bash

# replace relative imports with absolute imports.
# WARN this is not guaranteed to not break imports or to not substitute imports with similar looking ones.

# list all first level dirs in the `src` directory and print just the names
function list_dirs() {
  cd src && find . -type d -depth 1 | sed 's/^\.\///'
}

# find relative import patterns with top level dir names
function rel_pattern() {
  # echo "from '.*${1}"
  echo " '[./]*${1}\/"
}

function cmd() {
  dir_name=$1
  echo "grep --exclude-dir=\".git;node_modules\" -E \"$(rel_pattern $dir_name)\" -R . -l | xargs sed -i '' \"s/$(rel_pattern $dir_name)/ '${dir_name}\//g\""
}

for dir in $(list_dirs); do
  do=$(cmd $dir)
  echo $do
  sh -c "$do" & # what could go wrong?!
done

wait $(jobs -p)

# this could be faster but it's not ready
# export -f rel_pattern
# export -f cmd
# list_dirs | xargs -n1 -I{} bash -c 'cmd {}' | xargs -n1 -P1 -I{} sh -c {}
