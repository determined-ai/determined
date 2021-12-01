#!/usr/bin/env bash

MAJOR_SEPARATOR='================================================================================'
MINOR_SEPARATOR='--------------------------------------------------------------------------------'

# These vulnerabilities pertain to the Docker engine (or proprietary distributions), but only the Python client is included in the images
DOCKER="CVE-2017-7297 CVE-2019-13139 CVE-2019-13509 CVE-2019-16884 CVE-2019-5736"
# This vulnerability is fixed in JupyterLab 3.2.0, but was still flagged after we moved to that version
FIXED_IN_JUPYTER_3_2_0="CVE-2021-32797"
# These vulnerabilities are fixed in newer versions of TensorFlow, but not in 1.15, which is out of maintenance and no longer our default
TENSORFLOW_1_15="GHSA-2r8p-fg3c-wcj4 GHSA-4xfp-4pfp-89wg GHSA-5xwc-mrhx-5g3m GHSA-6gv8-p3vj-pxvr GHSA-6p5r-g9mq-ggh2 GHSA-7fvx-3jfc-2cpc GHSA-8pmx-p244-g88h GHSA-9c8h-vvrj-w2p8 GHSA-c5x2-p679-95wc GHSA-c9qf-r67m-p7cg GHSA-cgfm-62j4-v4rf GHSA-cwv3-863g-39vx GHSA-f5cx-5wr3-5qrc GHSA-f8h4-7rgh-q2gm GHSA-fcwc-p4fc-c5cc GHSA-g25h-jr74-qp5j GHSA-g8wg-cjwc-xhhp GHSA-gh6x-4whr-2qv4 GHSA-h4pc-gx2w-f2xv GHSA-hpv4-7p9c-mvfr GHSA-hwr7-8gxx-fj5p GHSA-jf7h-7m85-w2v2 GHSA-m7fm-4jfh-jrg6 GHSA-q3g3-h9r4-prrc GHSA-qr82-2c78-4m8h GHSA-r4c4-5fpq-56wg GHSA-r6jx-9g48-2r5r GHSA-v768-w7m9-2vmm GHSA-v82p-hv3v-p6qp GHSA-w4xf-2pqw-5mq7 GHSA-w74j-v8xh-3w5h GHSA-wp77-4gmm-7cq8 GHSA-374m-jm66-3vj8 GHSA-3rcw-9p9x-582v GHSA-49rx-x2rw-pc6f GHSA-4f99-p9c2-3j8x GHSA-57wx-m983-2f88 GHSA-7pxj-m4jf-r6h2 GHSA-cqv6-3phm-hcwx GHSA-f54p-f6jp-4rhr GHSA-fr77-rrx3-cp7g GHSA-j86v-p27c-73fm GHSA-m342-ff57-4jcc GHSA-pgcq-h79j-2f69 GHSA-rg3m-hqc5-344v GHSA-vwhq-49r4-gj9v"

IGNORED_VULNERABILITIES=" ${DOCKER} ${FIXED_IN_JUPYTER_3_2_0} ${TENSORFLOW_1_15} "

# Exit codes
SUCCESS=0
INVALID_ARGS=1
MISSING_DEPENDENCIES=2
VULNERABILITIES_FOUND=3


function print_usage_and_exit_with_error() {
  echo ${MAJOR_SEPARATOR}
  echo "Error: ${1}"
  echo
  echo Usage:
  echo
  echo "    ${0} {path/to/bumpenvs.yaml | --images [image_name ...]}"
  echo ${MAJOR_SEPARATOR}
  exit ${INVALID_ARGS}
}

if [ ${#} -lt 1 ]; then
  print_usage_and_exit_with_error "${#} arguments received, expected 1 or more"
fi

while [[ ${#} -gt 0 ]]; do
  key=${1}

  case $key in
    --images)
      shift
      IMAGES=$*
      break
      ;;
    *)
      if [ ${#} -ne 1 ]; then
        print_usage_and_exit_with_error "${#} arguments received, expected 1"
      fi

      BUMPENVS=${1}
      break
    ;;
  esac
done

for command in anchore-cli awk curl docker-compose grep jq yq; do
  if ! which ${command} > /dev/null; then
    echo "${0} requires ${command}"
    exit ${MISSING_DEPENDENCIES}
  fi
done

if [ -n "$BUMPENVS" ]; then
  if [ ! -f "$BUMPENVS" ]; then
    print_usage_and_exit_with_error "File ${1} does not exist"
  fi

  IMAGES=$(yq eval -o=p "${BUMPENVS}" | grep '_hashed.new = ' | awk '{ print $3 }')
fi

# Set up Anchore Engine
COMPOSE_FILE=/tmp/determined-anchore-engine.yaml
curl https://engine.anchore.io/docs/quickstart/docker-compose.yaml > ${COMPOSE_FILE}
docker-compose -f ${COMPOSE_FILE} up -d

# Configure Anchore CLI
export ANCHORE_CLI_USER=admin
export ANCHORE_CLI_PASS=foobar

timeout 30 bash -c "while ! anchore-cli image list; do sleep 1; done"

# Start download and scanning
for image in ${IMAGES}; do
  anchore-cli image add "${image}"
done

# Wait on results
for image in ${IMAGES}; do
  anchore-cli image wait "${image}"
done

# Check results
total_failures=0
for image in ${IMAGES}; do
  echo "${MAJOR_SEPARATOR}"
  echo "${image}"
  echo "${MINOR_SEPARATOR}"
  # Run the following for a full report: anchore-cli image vuln ${image} all
  output=$(anchore-cli --json image vuln "${image}" all | jq -r '.vulnerabilities[] | select(.severity == "High" or .severity == "Critical") | .vuln' | sort)
  file_failures=0
  while IFS= read -r vulnerability; do
    if [[ ${IGNORED_VULNERABILITIES} == *"${vulnerability}"* ]]; then
      continue
    fi
    file_failures=$((file_failures+1))
    total_failures=$((total_failures+1))
    echo "${vulnerability}"
  done <<< "${output}"
  echo ${file_failures} critical or high severity vulnerabilities found in image
done

echo ${MAJOR_SEPARATOR}
echo ${total_failures} critical or high severity vulnerabilities found in total

docker-compose -f ${COMPOSE_FILE} down

if [ ${total_failures} -gt 0 ]; then
  exit ${VULNERABILITIES_FOUND}
fi

exit ${SUCCESS}
