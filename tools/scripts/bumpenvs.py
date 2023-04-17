#!/usr/bin/env python3

"""
Update all the environment image tags in the repository based on bumpenvs.yaml.

Usage: bumpenvs.py path/to/bumpenvs.yaml
"""

import os
import subprocess
import sys
from typing import List

from ruamel import yaml


def samefile(a: str, b: str) -> bool:
    return os.path.samefile(os.path.expanduser(a), os.path.expanduser(b))


def list_git_paths_with_tag(tag: str, ignore_files: List[str]) -> List[str]:
    cmd = ["git", "rev-parse", "--show-toplevel"]
    r = subprocess.run(cmd, stdout=subprocess.PIPE, universal_newlines=True, check=True)

    git_root = r.stdout.rstrip("\n")

    cmd = ["git", "grep", "-z", "-l", tag, git_root]
    r = subprocess.run(cmd, stdout=subprocess.PIPE, universal_newlines=True, check=True)
    paths = r.stdout.rstrip("\0").split("\0")

    return [p for p in paths if not any(samefile(p, ignore) for ignore in ignore_files)]


def replace_tags(path: str, old_tag: str, new_tag: str) -> None:
    with open(path) as f:
        lines_in = f.readlines()

    lines_out = [line.replace(old_tag, new_tag) for line in lines_in]

    # Print git-style diffs.
    print(path, file=sys.stderr)
    for before, after in zip(lines_in, lines_out):
        if before != after:
            print(f"\x1b[31m-{before.rstrip()}\x1b[m", file=sys.stderr)
            print(f"\x1b[32m+{after.rstrip()}\x1b[m", file=sys.stderr)
            print(file=sys.stderr)

    with open(path, "w") as f:
        f.writelines(lines_out)


def main(config_file: str) -> None:
    with open(config_file) as f:
        config = yaml.safe_load(f)

    for image_type, tag_pair in config.items():
        if "old" not in tag_pair:
            print(
                f'\x1b[33mskipping {image_type} which has no "old" tag\x1b[m',
                file=sys.stderr,
            )
            continue

        old_tag = tag_pair["old"]
        new_tag = tag_pair["new"]

        for path in list_git_paths_with_tag(old_tag, ignore_files=[config_file]):
            replace_tags(path, old_tag, new_tag)

    print("Successfully modified all files.", file=sys.stderr)


if __name__ == "__main__":
    if len(sys.argv) < 2:
        print(__doc__, file=sys.stderr)
        sys.exit(1)
    main(sys.argv[1])
