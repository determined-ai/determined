import argparse
import html
import io
import json
import os
import pathlib
import re
import sys
import traceback
from xml.etree import ElementTree

from algoliasearch import search_client

HERE = pathlib.Path(__file__).parent

EXCLUDES = ["release-notes/", "attributions.xml"]

BUILD = str(HERE / ".." / "site" / "xml")


if __name__ == "__main__":
    # get path to blob to upload
    # get account creds
    # check if objects already exist at destination
    # check if upload being done is a preview site
    # opt: flag to delete blobs missing from destination
    parser = argparse.ArgumentParser()
    parser.add_argument("--json", action="store_true", help="dump records to stdout")
    parser.add_argument("--upload", action="store_true", help="upload to algolia")
    parser.add_argument("--app-id", default="9H1PGK6NP7", help="algloia app id")
    parser.add_argument(
        "--api-key", default=os.environ.get("ALGOLIA_API_KEY"), help="algloia admin key"
    )
    args = parser.parse_args()

    # Pick the correct version.
    HERE = pathlib.Path(__file__).parent
    with (HERE / ".." / ".." / "VERSION").open() as f:
        version = f.read().strip()
    if "-dev" in version:
        # Dev builds search against a special dev index that is update with every push to master.
        version = "dev"
    elif "-rc" in version:
        # Each release candidate publishes against the actual version without the "-rc" in the name.
        version = version[: version.index("-rc")]

    records = scrape_tree(BUILD, EXCLUDES)

    if args.json:
        json.dump(records, sys.stdout, indent="  ")

    if args.upload:
        if not args.api_key:
            print("--api-key or ALGOLIA_API_KEY required for upload", file=sys.stderr)
            exit(1)
        upload(args.app_id, args.api_key, records, version)

    if not args.upload and not args.json:
        print("scrape was successful, try --json or --upload", file=sys.stderr)
