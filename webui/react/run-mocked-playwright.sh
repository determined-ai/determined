#!/bin/bash
set -euxo pipefail

make mb-start
cat "/tmp/mb-playwright.log"
# make mb-start-saved-imposters
export DET_WEBPACK_PROXY_URL="http://localhost:4545"
export DET_WEBSOCKET_PROXY_URL="ws://localhost:4546"
export PW_SERVER_ADDRESS="http://localhost:3001"
export PW_TEST_HTML_REPORT_OPEN='never'
set +e
PW_SERVER_ADDRESS="http://localhost:3001" DET_WEBSOCKET_PROXY_URL="ws://localhost:4546" DET_WEBPACK_PROXY_URL="http://localhost:4545" npm run e2e
test_result=$?
set -e
if [ $test_result -ne 0 ]; then
    echo "Tests failed, re-running without mocks"
    devcluster --oneshot -c .circleci/devcluster/double.devcluster.yaml --target-stage agent1
    make mb-stop
    make mb-start
    make mb-record-imposters
    set +e
    npm run e2e
    test_result=$?
    set -e
    if [ $test_result -ne 0 ]; then
        echo "Tests failed with a real backend service. This is likely a real bug or a test needs updates."
        exit $test_result
    else
        test_result=1 # we still failed and need updates
        echo "Tests failed with mocks, but passed with the real service. Mocks likely need updates. \n\
        Copy 'webui/react/src/e2e/mocks/debug-saved-imposters.jsonn' from the artifacts, \n\
        and replace mocks/saved-imposters.json to update the mocks."
        make mb-save-imposters
    fi
fi

make mb-stop
exit $test_result
