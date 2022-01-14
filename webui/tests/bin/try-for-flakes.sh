#!/bin/bash -e

# eg ./bin/try-for-flakes.sh true "npx gauge run --env ci ./specs/02-authentication.spec"

# if tests are re-runnable
re_runnable=${1:-false}
test_cmd=${2:-"./bin/e2e-tests.py run-e2e-tests"}

c=0

if $re_runnable; then
  while true; do
    c=$((c+1))
    echo "run #$c"
    ${test_cmd}
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
