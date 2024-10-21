#!/bin/bash

# This script dynamically determines an appropriate version string for the
# currently checked-out commit. As tag versions will typically be provided by CI
# for releases, this script is primarily to support local builds that work as
# one would expect.
#
# Consider the following git DAG representing a hypothetical Determined git
# tree:
#
#                     HEAD
#                      |
#                  I---J (feat-behind-latest-tag)
#                /
#...A---B---C---D---E---F---M---N---O (main)
#    \                   \
#     G---H (1.1.x)       K---L (1.2.x)
#         |                   |
#       1.1.0               1.2.0
#
# The newest version is 1.2.0, as tagged on the release branch 1.2.x, at L. The
# previous version is 1.1.0, on release branch 1.1.x, at H. Our feature branch
# is feat-behind-latest-tag, with HEAD at J. The goal is to output the nearest
# tag on the release branch behind wherever we are. At first, I tried to finagle
# git-rev-list to make this happen, but I was unable to get a working
# solution. So instead, I used a combination of git-describe and git-tag.
#
# git-describe will give us a tag if we're currently on a release branch, or any
# branch with tags, which is unlikely, but possible. If we're on a feature
# branch, which is much more likely, we need to use git-tag and git-merge-base
# to work backward: we find the merge-base of HEAD and main, then search for all
# tags that don't contain that commit (i.e. all tags created before that
# commit). From there, we just sort and filter the tag list, and grab the top
# element, which is the most recent, previous tag.
#
# So, in our diagram, if run from B, C, D, E, F, I, J, or K, the script will return
# 1.1.0. If run from L, M, N, or O, it will return 1.2.0. And so on.

OPTSTRING=":td"

# Options parsing
# -t outputs the full git tag, necessary for some build steps.
# -d outputs a Docker-friendly image tag, necessary for pushing to Docker Hub.
# The default case simply strips the leading 'v'.
while getopts ${OPTSTRING} opt; do
    case ${opt} in
        t)
            TAG_OUTPUT=1
            ;;
        d)
            DOCKER_OUTPUT=1
            ;;
        ?)
            echo "Invalid option: -${OPTARG}."
            exit 1
            ;;
    esac
done

# Set VERSION to CIRCLE_TAG in case we're running in CircleCI. This makes it
# easier to avoid fiddling with environment variables there.
if [[ -n ${CIRCLE_TAG} ]]; then
    VERSION=${CIRCLE_TAG}
fi

# If VERSION is unset or the empty string, "". This will be the default case for
# local builds.
if [[ -z ${VERSION} ]]; then
    # Check if this branch has any tags (typically, only release branches will
    # have tags).
    MAYBE_TAG=$(git describe --tags --abbrev=0 2>/dev/null)
    SHA=$(git rev-parse --short HEAD)

    # No tag on current branch.
    if [[ -z ${MAYBE_TAG} ]]; then
        # Use git to find the merge base between the current branch and main,
        # and then find the closest tag behind that, using --no-contains. Then,
        # use grep to remove some special cases, namely: old Determined version
        # tags beginning with 'v', and all tags that end in '-ee'. Then, use
        # head to grab the first one, since the list is sorted in descending
        # order, handling -rc tags correctly courtesy of
        # versionsort.suffix.
        MAYBE_TAG=$(
            git \
                -c versionsort.suffix='-rc' \
                tag \
                --sort='-v:refname:short' \
                --format='%(refname:short)' \
                --no-contains=$(git merge-base HEAD main) \
                | grep -E -v 'v0.12|-ee' \
                | head -n 1
        )
    fi

    # Filter out additional +metadata from tag, should it exist. This prevents
    # tags like 0.751.0+dryrun+27a014b44.
    MAYBE_TAG=$(grep -Eo 'v?\d+\.\d+\.\d+' <(printf "%s" "$MAYBE_TAG"))

    # Munge the tag into the form we want. Note: we always append a SHA hash,
    # even if we're on the commit with the tag. This is partially because I feel
    # like it will be more consistent and result in fewer surprises, but also it
    # might help indicate that this is a local version.
    if [[ -n ${TAG_OUTPUT} ]]; then
        echo -n "${MAYBE_TAG}+${SHA}"
    elif [[ -n ${DOCKER_OUTPUT} ]]; then
        # Docker image tags must have the following format:
        # [A-Za-z0-9_][A-Za-z0-9_\.\-]{0,127}
        echo -n "${MAYBE_TAG#v}-${SHA}" | sed -e 's/\+/-/g'
    else
        echo -n "${MAYBE_TAG#v}+${SHA}"
    fi
else
    # Use existing VERSION, which is much easier. This should be the default
    # case for CI, as VERSION will already be set. We also remove the 'v' from
    # the tag for the version string, as that is what the current CI
    # functionality expects. Some build steps expect
    # the version string to be the full tag, so check for a -t flag and return
    # the full tag version string if it's set. Otherwise, return the version
    # string without the 'v'.
    if [[ -n ${TAG_OUTPUT} ]]; then
        echo -n "${VERSION}"
    elif [[ -n ${DOCKER_OUTPUT} ]]; then
        # Docker image tags must have the following format:
        # [A-Za-z0-9_][A-Za-z0-9_\.\-]{0,127}
        echo -n "${VERSION#v}" | sed -e 's/\+/-/g'
    else
        echo -n "${VERSION#v}"
    fi
fi
