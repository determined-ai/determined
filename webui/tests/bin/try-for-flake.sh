#!/bin/bash

# if tests are re-runnable
re_runnable=${1:-false}
cmd_prefix=docker-

c=0

if $re_runnable; then
  make pre-e2e-tests
  while true; do
    c=$((c+1))
    echo "run #$c"
    make ${cmd_prefix}run-e2e-tests || break
  done
else # run the whole suite
  while true; do
    c=$((c+1))
    echo "run #$c"
    make ${cmd_prefix}e2e-tests || break
  done
fi

echo "result: test failure at run #$c of $(git rev-parse --short HEAD)"
# TODO use trap to show successful result upon SIGTERM SIGINT
