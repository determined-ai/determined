#!/bin/bash -e

# if tests are re-runnable
re_runnable=${1:-false}

c=0

if $re_runnable; then
  ./bin/e2e-tests.py pre-e2e-tests
  while true; do
    c=$((c+1))
    echo "run #$c"
    ./bin/e2e-tests.py run-e2e-tests
  done
else # run the whole suite
  while true; do
    c=$((c+1))
    echo "run #$c"
    make test
  done
fi

echo "result: test failure at run #$c of $(git rev-parse --short HEAD)"
# TODO use trap to show successful result upon SIGTERM SIGINT
