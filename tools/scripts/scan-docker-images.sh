#!/usr/bin/env bash

function print_usage_and_exit_with_error() {
  echo Error: ${1}
  echo
  echo Usage:
  echo 
  echo "    ${0} path/to/bumpenvs.yaml"
  exit 1
}

if [ ${#} -ne 1 ]; then
  print_usage_and_exit_with_error "${#} arguments received, expected 1"
fi

if [ ! -f ${1} ]; then
  print_usage_and_exit_with_error "File ${1} does not exist"
fi

BUMPENVS=${1}

for command in anchore-cli awk curl docker-compose grep jq yq; do
  if ! which ${command} > /dev/null; then
    echo ${0} requires ${command}
    exit 2
  fi
done

# Set up Anchore Engine
COMPOSE_FILE=/tmp/determined-anchore-engine.yaml
curl https://engine.anchore.io/docs/quickstart/docker-compose.yaml > ${COMPOSE_FILE}
docker-compose -f ${COMPOSE_FILE} up -d

# Configure Anchore CLI
export ANCHORE_CLI_USER=admin
export ANCHORE_CLI_PASS=foobar

timeout 30 bash -c "while ! anchore-cli image list; do sleep 1; done"

# Start download and scanning
IMAGES=$(yq eval -o=p ${BUMPENVS} | grep '_hashed.new = ' | awk '{ print $3 }')
for image in ${IMAGES}; do
  anchore-cli image add ${image}
done

IMAGES=$(yq eval -o=p ${BUMPENVS} | grep '_hashed.new = ' | awk '{ print $3 }')
for image in ${IMAGES}; do
  anchore-cli image wait ${image}
done

# Wait on results and check them
failures=0
separator='################################################################################'
for image in ${IMAGES}; do
  # Run the following for a full report: anchore-cli image vuln ${image} all
  output=$(anchore-cli --json image vuln ${image} all | jq -r '.vulnerabilities[] | select(.severity == "High" or .severity == "Critical") | .vuln')
  if [ -n "${output}" ]; then
    failures=$((failures+1))
    echo "${separator}"
    echo "${image}"
    echo "${separator}"
    echo "${output}"
  fi
done

echo ${separator}
echo ${failures} files have high-severity vulnerabilities

docker-compose -f ${COMPOSE_FILE} down

exit ${failures}
