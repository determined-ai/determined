#!/bin/bash
set -euxo pipefail

make mb-start
sleep 10 # let MB start up. It's pretty quick.
make mb-start-saved-imposters
export DET_WEBPACK_PROXY_URL="http://localhost:4545"
export DET_WEBSOCKET_PROXY_URL="ws://localhost:4546"
export PW_SERVER_ADDRESS="http://localhost:3001"
export PW_TEST_HTML_REPORT_OPEN='never'
set +e
npm run e2e
test_result=$?
set -e
if [ $test_result -ne 0 ]; then
    echo "Tests failed, re-running without mocks"
    det deploy local cluster-up --no-gpu
    make mb-stop
    make mb-start
    sleep 10
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
        echo "Tests failed with mocks, but passed with a real service. \
        Mocks likely need updates. \
        Copy 'webui/react/src/e2e/mocks/debug-saved-imposters.jsonn' from the artifacts, \
        and replace mocks/saved-imposters.json to update the mocks."
        make mb-save-imposters
    fi
fi

make mb-stop
exit $test_result
