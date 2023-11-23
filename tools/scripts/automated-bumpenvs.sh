#!/bin/bash -ex

# in case these are already set
ENVS_HASH=''
DEV_FLAG=''
NO_CLOUD_IMAGES_FLAG=''

while getopts "h:dn" opt; do
  case $opt in
    h)
      ENVS_HASH="$OPTARG"
      echo "using the following envs hash: $ENVS_HASH"
    ;;
    d)
      echo "--dev flag will be used"
      DEV_FLAG="--dev"
    ;;
    n)
      echo "--no-cloud-images flag will be used"
      NO_CLOUD_IMAGES_FLAG="--no-cloud-images"
    ;;
    \?) echo "Invalid option -$OPTARG" >&2
    exit 1
    ;;
  esac
done

# check for dirty changes
if [[ -n "$(git status --porcelain)" ]]; then
    echo "untracked or dirty files are not allowed, cleanup before running insert-dropdown-url.sh"
    exit 1
fi

if [[ -z $ENVS_HASH ]]; then
    # get latest environments commit
    ENVS_HASH="$(git ls-remote https://github.com/determined-ai/environments.git -h HEAD -q | cut -f1)"
    echo "nah"
fi

# update bumpenvs yaml with the given hash
python3 tools/scripts/update-bumpenvs-yaml.py tools/scripts/bumpenvs.yaml ENVS_HASH DEV_FLAG NO_CLOUD_IMAGES_FLAG

# run the bumpenvs procedure
python3 tools/scripts/bumpenvs.py tools/scripts/bumpenvs.yaml

# check to see if bumpenvs.py published resulted in any file changes or not
if [[ -z "$(git status --porcelain)" ]]; then
    echo "no change to environment images found"
    exit 0
fi
git add --update
git commit -m "chore: bumpenvs to environments commit $ENVS_HASH"
