#!/usr/bin/env python3

import argparse
import json
import os
import re
import sys

import git
import git.exc
import git.refs.tag

# These are the tags that are present in git, but not in the original
# versions.json file. We specify them here to ensure we don't create links to
# documentation that doesn't exist, and to recreate part of the original
# versions.json file.
EXCLUDE_VERSIONS = [
    "0.26.5",
    "0.23.4",
    "0.23.3",
    "0.23.2",
    "0.23.1",
    "0.22.2",
    "0.22.1",
    "0.21.2",
    "0.21.1",
]


def parse_args():
    parser = argparse.ArgumentParser(
        prog="gen-versions.py",
        description="Generate Sphinx version switcher JSON file from git tags.",
    )

    parser.add_argument(
        "commit",
        help="commit to stop at while walking the graph, from each tag ref as a starting point, to look for tags, beyond which tags will be ignored. This corresponds to ^commit in git-rev-list.",
        metavar="commit",
    )

    parser.add_argument(
        "-o",
        "--out-file",
        help="path to output file, including filename, for generated versions JSON file",
        metavar="path",
        default=None,
    )

    # Escape-hatch argument to exclude versions that are returned from
    # git-rev-list. This basically lets us recreate the original file we already
    # have, along with appending subsequent new version tags. We need this
    # because the git-rev-list tags returned are a superset of the versions
    # originally in versions.json. I.e. some of the existing tagged patch
    # release versions don't have corresponding separate doc links.
    parser.add_argument(
        "--exclude-versions",
        help="comma-separated list of additional versions to exclude from the versions returned by walking the git DAG",
        metavar="versions",
        type=str,
    )

    return parser.parse_args()


def main():
    args = parse_args()

    exclude_versions = []
    if args.exclude_versions is not None:
        # Include the default EXCLUDE_VERSIONS to reflect the historical
        # versions.json file.
        exclude_versions = args.exclude_versions.split(",")
        exclude_versions.extend(EXCLUDE_VERSIONS)

    # Probably run this from inside the repo somewhere.
    try:
        repo = git.Repo(os.getcwd(), search_parent_directories=True)
    except git.exc.InvalidGitRepositoryError as e:
        print(
            f"Invalid git repository: {e}. Are you running this from a git repository?",
            file=sys.stderr,
        )
        raise
    except git.exc.NoSuchPathError as e:
        print(f"Path does not exist: {e}.", file=sys.stderr)
        raise

    # Validate commit.
    try:
        repo.rev_parse(args.commit)
    except git.exc.BadName as e:
        print(f"Bad revision: {e}.", file=sys.stderr)
        raise

    # git rev-list --tags --ancestry-path ^commit
    comms_iter = repo.iter_commits(f"^{args.commit}", ancestry_path=True, tags=True)

    # Get commits from iterator.
    commits = []
    try:
        for comm in comms_iter:
            commits.append(comm)
    except git.exc.GitCommandError as e:
        # rev_parse up above should catch these, but you never know.
        print(f"Unable to list commits: {e}. Is your commit ID correct?", file=sys.stderr)
        raise

    # Map SHA hash to tag.
    tag_refs = git.refs.tag.TagReference.list_items(repo)

    commit_tags = {}

    # Ignore all tags that aren't of the form x.y.z
    tag_regex = re.compile("\d+\.\d+\.\d+$")

    for tag in tag_refs:
        m = tag_regex.search(str(tag))
        if m:
            commit_tags.update({str(tag.commit): m.group()})

    # Emulate git describe --tags <tag> 2>/dev/null and collect tags.
    tags = []
    for commit in commits:
        tag = commit_tags.get(str(commit))
        if tag and tag not in exclude_versions:
            tags.append(tag)

    versions = []
    if tags:
        # Descending order.
        tags.reverse()

        # Remove latest tag as it is a special case.
        latest_tag = tags.pop()

        # Append all new versions that aren't the latest tag.
        for tag in tags:
            versions.append(
                {
                    "version": tag,
                    "url": f"https://docs.determined.ai/{tag}/",
                }
            )

        versions.append(
            {
                "version": latest_tag,
                "url": "https://docs.determined.ai/latest/",
            }
        )

    versions.reverse()

    if args.out_file is not None:
        try:
            with open(args.out_file, "w") as fd:
                json.dump(versions, fd, indent=4)
        except FileNotFoundError as e:
            print("File not found: {e}. Do all parent directories exist?", file=sys.stderr)
            raise
    else:
        print(json.dumps(versions, indent=4))


if __name__ == "__main__":
    main()
