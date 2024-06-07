#!/usr/bin/env python3

import argparse
import json
import os
import re
import sys

import git
import git.exc
import git.refs.tag

# This is the original versions.json file, frozen so we can generate newer
# versions while still maintaining the functionality of older versions.
VERSIONS = [
    {
        "version": "0.33.0",
        "url": "https://docs.determined.ai/0.33.0/"
    },
    {
        "version": "0.32.1",
        "url": "https://docs.determined.ai/0.32.1/"
    },
    {
        "version": "0.32.0",
        "url": "https://docs.determined.ai/0.32.0/"
    },
    {
        "version": "0.31.0",
        "url": "https://docs.determined.ai/0.31.0/"
    },
    {
        "version": "0.30.0",
        "url": "https://docs.determined.ai/0.30.0/"
    },
    {
        "version": "0.29.1",
        "url": "https://docs.determined.ai/0.29.1/"
    },
    {
        "version": "0.29.0",
        "url": "https://docs.determined.ai/0.29.0/"
    },
    {
        "version": "0.28.1",
        "url": "https://docs.determined.ai/0.28.1/"
    },
    {
        "version": "0.28.0",
        "url": "https://docs.determined.ai/0.28.0/"
    },
    {
        "version": "0.27.1",
        "url": "https://docs.determined.ai/0.27.1/"
    },
    {
        "version": "0.27.0",
        "url": "https://docs.determined.ai/0.27.0/"
    },
    {
        "version": "0.26.7",
        "url": "https://docs.determined.ai/0.26.7/"
    },
    {
        "version": "0.26.6",
        "url": "https://docs.determined.ai/0.26.6/"
    },
    {
        "version": "0.26.4",
        "url": "https://docs.determined.ai/0.26.4/"
    },
    {
        "version": "0.26.3",
        "url": "https://docs.determined.ai/0.26.3/"
    },
    {
        "version": "0.26.2",
        "url": "https://docs.determined.ai/0.26.2/"
    },
    {
        "version": "0.26.1",
        "url": "https://docs.determined.ai/0.26.1/"
    },
    {
        "version": "0.26.0",
        "url": "https://docs.determined.ai/0.26.0/"
    },
    {
        "version": "0.25.1",
        "url": "https://docs.determined.ai/0.25.1/"
    },
    {
        "version": "0.25.0",
        "url": "https://docs.determined.ai/0.25.0/"
    },
    {
        "version": "0.24.0",
        "url": "https://docs.determined.ai/0.24.0/"
    },
    {
        "version": "0.23.0",
        "url": "https://docs.determined.ai/0.23.0/"
    },
    {
        "version": "0.22.0",
        "url": "https://docs.determined.ai/0.22.0/"
    },
    {
        "version": "0.21.0",
        "url": "https://docs.determined.ai/0.21.0/"
    }
]

def parse_args():
    parser = argparse.ArgumentParser(
        prog="gen-versions.py",
        description="Generate Sphinx version switcher JSON file from git tags.",
    )

    parser.add_argument("commit",
        help="commit to stop at while walking the graph to look for tags, beyond which tags will be ignored",
        metavar="commit",
    )

    parser.add_argument("-o", "--out-file",
        help="path to output file, including filename, for generated versions JSON file",
        metavar="path",
        default=None,
    )

    # Escape-hatch argument in case someone needs more granular control over the
    # generated file.
    parser.add_argument("--override-versions",
        help="comma-separated list of versions, in ascending order, to use instead of the existing versions and the tags in git. E.g. 0.0.1,0.0.2,0.0.3",
        metavar="versions",
        type=str
    )

    return parser.parse_args()

def generate_hardcoded(versions, out_file):
    latest = versions.pop()
    versions.reverse()

    out = []
    out.append({
        "version": latest,
        "url": "https://docs.determined.ai/latest/"
    })

    for version in versions:
        out.append({
            "version": version,
            "url": "https://docs.determined.ai/{}/".format(version)
        })

    # Dump to stdout.
    if out_file is not None:
        with open(out_file, "w") as fd:
            json.dump(out, fd, indent=4)
    else:
        print(json.dumps(out, indent=4))

def main():
    args = parse_args()

    if args.override_versions is not None:
        versions = args.override_versions.split(",")
        generate_hardcoded(versions)

        sys.exit(0)

    # Probably run this from inside the repo somewhere.
    try:
        repo = git.Repo(os.getcwd(), search_parent_directories=True)
    except git.exc.InvalidGitRepositoryError as e:
        print("Invalid git repository: {}. Are you running this from a git repository?".format(e), file=sys.stderr)
        sys.exit(-1)
    except git.exc.NoSuchPathError as e:
        print("Path does not exist: {}.", file=sys.stderr)
        sys.exit(-1)

    # Validate commit.
    try:
        repo.rev_parse(args.commit)
    except git.exc.BadName as e:
        print("Bad revision: {}.".format(e))
        sys.exit(-1)

    # git rev-list --branches --ancestry-path <commit>..HEAD
    comms_iter = repo.iter_commits("{}..HEAD".format(args.commit), branches=True, ancestry_path=True)

    # Get commits from iterator.
    commits = []
    try:
        for comm in comms_iter:
            commits.append(comm)
    except git.exc.GitCommandError as e:
        # rev_parse up above should catch these, but you never know.
        print("Unable to list commits: {}. Is your commit ID correct?".format(e))
        sys.exit(-1)

    # Map SHA hash to tag.
    tag_refs = git.refs.tag.TagReference.list_items(repo)

    commit_tags = {}
    for tag in tag_refs:
        commit_tags.update({str(tag.commit): tag.name})

    # Ignore all tags that aren't of the form x.y.z
    tag_regex = re.compile("\d+\.\d+\.\d+$")

    # Emulate git describe --tags <tag> 2>/dev/null and collect tags.
    tags = []
    for commit in commits:
        tag = commit_tags.get(str(commit))
        if tag:
            m = tag_regex.search(tag)
            if m:
                tags.append(m.group())

    # Remove current tag so we don't loop over it by mistake.
    latest_tag = tags.pop()

    # Prepend all new versions that aren't the current tag.
    for tag in tags:
        VERSIONS.insert(0, {
            "version": tag,
            "url": "https://docs.determined.ai/{}/".format(tag),
        })

    # Latest version is a special case.
    VERSIONS.insert(0, {
        "version": latest_tag,
        "url": "https://docs.determined.ai/latest/"
    })

    if args.out_file is not None:
        try:
            with open(args.out_file, "w") as fd:
                json.dump(VERSIONS, fd, indent=4)
        except FileNotFoundError as e:
            print("File not found: {}. Do all parent directories exist?".format(e), file=sys.stderr)
            sys.exit(-1)
    else:
        print(json.dumps(VERSIONS, indent=4))

if __name__ == "__main__":
    main()
