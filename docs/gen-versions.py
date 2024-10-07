#!/usr/bin/env python3

import argparse
import json
import os
import subprocess
import sys

def parse_args():
    parser = argparse.ArgumentParser(
        prog="gen-versions.py",
        description="Generate Sphinx version switcher JSON file from git tags.",
    )

    parser.add_argument(
        "-o",
        "--out-file",
        help="path to output file, including filename, for generated versions JSON file",
        metavar="path",
        default=None,
    )

    return parser.parse_args()


def main():
    args = parse_args()

    completed = subprocess.run(["./gen-versions.sh"], capture_output=True)
    versions = completed.stdout.splitlines()

    output = []

    # Special case for latest.
    latest = versions.pop(0)
    latest = latest.decode()
    output.append(
        {
            "version": latest,
            "url": "https://docs.determined.ai/latest/",
        }
    )

    for version in versions:
        version = version.decode()
        output.append(
            {
                "version": version,
                "url": f"https://docs.determined.ai/{version}/",
            }
        )

    if args.out_file is not None:
        try:
            with open(args.out_file, "w") as fd:
                json.dump(output, fd, indent=4)
        except FileNotFoundError as e:
            print("File not found: {e}. Do all parent directories exist?", file=sys.stderr)
            raise
    else:
        print(json.dumps(output, indent=4))


if __name__ == "__main__":
    main()
