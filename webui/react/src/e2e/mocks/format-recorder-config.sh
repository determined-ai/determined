#!/bin/bash
set -euxo pipefail
script_dir="$(dirname "${BASH_SOURCE[0]}")"
export MB_BACKEND_ADDRESS="${MB_BACKEND_ADDRESS:-localhost:8080}"

npm install uglify-js
export PRED_GEN_SCRIPT=$(npx uglify-js "${script_dir}/predicate-generator.js" --compress --mangle --output-opts quote_style=3)

envsubst < "${script_dir}/proxy-always-template.json" > "${script_dir}/proxy-always.json"