#!/bin/bash

# if tests are re-runnable
re_runnable=${1:-false}

c=0

if $re_runnable; then
  make pre-e2e-tests
  while true; do
    c=$((c+1))
    echo "run #$c"
    make docker-run-e2e-tests || break
  done
else # run the whole suite
  while true; do
    c=$((c+1))
    echo "run #$c"
    make docker-e2e-tests || break
  done
fi

echo "test failure at run #$c"
