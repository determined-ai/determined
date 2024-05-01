#!/bin/bash
set -euxo pipefail

cd webui/react
make mb-start
export DET_WEBPACK_PROXY_URL="http://localhost:4545"
export DET_WEBSOCKET_PROXY_URL="ws://localhost:4546"
export PW_SERVER_ADDRESS="http://localhost:3001"
export PW_TEST_HTML_REPORT_OPEN='never'
test_result=0
if [[ "$PW_MOCK_RECORD_ONLY" != "true" ]]; then
    make mb-start-saved-imposters
    set +e
    PW_USER_NAME="admin" PW_PASSWORD="" \
    PW_SERVER_ADDRESS="http://localhost:3001"\
    DET_WEBSOCKET_PROXY_URL="ws://localhost:4546" DET_WEBPACK_PROXY_URL="http://localhost:4545" \
    npm run e2e -- userManagement.spec.ts -g "With a Test User" --project=mock-env
    test_result=$?
    set -e
    if [[ "$PW_MOCK_ONLY" == "true" ]]; then 
        exit $test_result
    fi
fi
if [[ "$PW_MOCK_RECORD_ONLY" == "true" || $test_result -ne 0 ]]; then
    echo "Running tests with mock recorder"
    make mb-stop
    make mb-start
    make mb-record-imposters
    set +e
    PW_USER_NAME="admin" PW_PASSWORD="" \
    PW_SERVER_ADDRESS="http://localhost:3001"\
    DET_WEBSOCKET_PROXY_URL="ws://localhost:4546" DET_WEBPACK_PROXY_URL="http://localhost:4545" \
    npm run e2e -- userManagement.spec.ts -g "With a Test User" --project=mock-env
    test_result=$?
    set -e
    if [[ $test_result -ne 0 ]]; then
        echo "Tests failed with a real backend service. This is likely a real bug or a test needs updates."
        exit $test_result
    else
        echo "Tests failed with mocks, but passed with the real service. Mocks likely need updates. \n\
        Copy 'webui/react/src/e2e/mocks/debug-saved-imposters.jsonn' from the artifacts, \n\
        and replace mocks/saved-imposters.json to update the mocks."
        make mb-save-imposters
    fi
fi

make mb-stop
cd ../..
exit $test_result
