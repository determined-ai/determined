#!/usr/bin/env python3

"""
Insert an entry into the version dropdown json file.

Usage: insert-version-url.py your.version.here
"""

import json
import os
import sys


def insert_entry(json_file, entry):
    # The current dev version comes first and all released versions are in reverse chronological
    # order, so the latest release always goes in index 1.
    entry_position = 1
    with open(json_file, "rb") as f:
        all_urls = json.load(f)

    all_urls.insert(entry_position, entry)

    with open(json_file, "w") as f:
        json.dump(all_urls, f, indent=4 * " ")
        f.write("\n")


def insert_version_url(json_file, version):
    insert_entry(json_file, {"version": version, "url": f"https://docs.determined.ai/{version}/"})

    print(f"Added dropdown link for version {version}")


if __name__ == "__main__":
    if len(sys.argv) < 2:
        print(__doc__, file=sys.stderr)
        sys.exit(1)

    current_directory = os.path.dirname(__file__)
    versions_json_path = os.path.join(current_directory, "_static/version-switcher/versions.json")
    insert_version_url(versions_json_path, sys.argv[1])
