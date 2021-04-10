#!/usr/bin/env python3

"""
Fetch the latest official Ubuntu AMIs that we hardcode into our system.

This will update all *_master_ami and *_bastion_ami tags in bumpenvs.yaml file.

Usage: refresh-ubuntu-amis.py path/to/bumpenvs.yaml
"""

import sys
from typing import Dict, List

import requests
import yaml

get_cache = {}  # type: Dict[str, str]


def cacheable_get(url: str) -> str:
    global get_cache
    if url not in get_cache:
        req = requests.get(url)
        req.raise_for_status()
        get_cache[url] = req.text
    return get_cache[url]


def get_ubuntu_ami(release: str, region: str) -> str:
    resp = cacheable_get(
        f"https://cloud-images.ubuntu.com/query/{release}/server/released.current.txt"
    )

    ami_lines = [line.split("\t") for line in resp.splitlines()]

    def filters(line: List[str]) -> bool:
        return all(
            [
                # Only use EBS, not instance-store.
                line[4] == "ebs-ssd",
                line[5] == "amd64",
                line[6] == region,
                # Only use HVM virtualization, not paravirtualization.
                line[10] == "hvm",
            ]
        )

    results = [line for line in ami_lines if filters(line)]

    assert (
        len(results) == 1
    ), f"expected one match to {release}/{region} but got {results}"

    return results[0][7]


def update_tag_for_image_type(subconf: Dict[str, str], new_tag: str) -> bool:
    if new_tag == subconf["new"]:
        return False

    subconf["old"] = subconf["new"]
    subconf["new"] = new_tag
    return True


if __name__ == "__main__":
    if len(sys.argv) < 2:
        print(__doc__, file=sys.stderr)
        sys.exit(1)

    path = sys.argv[1]

    with open(path) as f:
        conf = yaml.safe_load(f)

    for image_type, subconf in conf.items():
        if image_type.endswith("_master_ami"):
            region = image_type.rstrip("_master_ami").replace("_", "-")
            # Master AMIs are based on Focal.
            new_ami = get_ubuntu_ami("focal", region)
            update_tag_for_image_type(subconf, new_ami)

        if image_type.endswith("_bastion_ami"):
            region = image_type.rstrip("_bastion_ami").replace("_", "-")
            # Bastion AMIs are based on Focal.
            new_ami = get_ubuntu_ami("focal", region)
            update_tag_for_image_type(subconf, new_ami)

    with open(path, "w") as f:
        yaml.dump(conf, f, sort_keys=True)
