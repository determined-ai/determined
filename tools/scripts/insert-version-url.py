#!/usr/bin/env python3

"""
Insert an entry into the version dropdown json file

Usage: add_version_dropdown.py docs/_static/version-switcher/versions.json your.version.here
"""

import json
import sys

def create_url_entry(version):
    url=f"https://docs.determined.ai/{version}/"

    entry= {
        "version": version,
        "url": url
    }

    return entry

def append_entry(json_file, entry, entry_position=1):
    with open(json_file, 'rb') as f:
        all_urls = json.load(f)

    all_urls.insert(entry_position, entry)

    with open(json_file, 'w') as f:
        json.dump(all_urls, f, indent=4*" ")
        f.write('\n')

def main(json_file, version):
    new_entry = create_url_entry(version)
    append_entry(json_file, new_entry)

    print(f"Added dropdown link for version {version}")


if __name__=="__main__":
    if len(sys.argv) < 3:
        print(__doc__, file=sys.stderr)
        sys.exit(1)
    main(sys.argv[1], sys.argv[2])

