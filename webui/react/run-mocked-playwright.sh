#!/bin/bash
set -euxo pipefail

make mb-start
sleep 3 # let MB start up. It's pretty quick.
make mb-start-saved-imposters
export DET_WEBPACK_PROXY_URL="http://localhost:4545"
# export DET_WEBSOCKET_PROXY_URL="ws://localhost:4546"
npm run preview >/dev/null 2>&1 & PREVIEW_PID=$!
echo "preview running on ${PREVIEW_PID} pid."
PW_SERVER_ADDRESS="http://localhost:3001"  npm run e2e
if [ $$? -ne 0 ]; 
then 
    make mb-stop
    make mb-start
    sleep 3
    make mb-record-imposters
    PW_SERVER_ADDRESS="http://localhost:3001"  npm run e2e 
    make mb-save-imposters
fi
kill "$PREVIEW_PID"
make mb-stop
