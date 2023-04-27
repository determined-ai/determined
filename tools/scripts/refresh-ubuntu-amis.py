#!/usr/bin/env python3

"""
Fetch the latest official Ubuntu AMIs that we hardcode into our system.

This can update all *_master_ami and *_bastion_ami tags in bumpenvs.yaml file.

This can also update the environments-packer.json in the environments repo.
"""

import argparse
import json
import re
import sys
from typing import Dict, List, Union

import requests
from ruamel import yaml


def get_ubuntu_ami(table: List[List[str]], release: str, region: str) -> Union[None, str]:
    def filters(line: List[str]) -> bool:
        return all(
            [
                line[0] == release,
                # Only use EBS, not instance-store.
                line[4] == "ebs-ssd",
                line[5] == "amd64",
                line[6] == region,
                # Only use HVM virtualization, not paravirtualization.
                line[10] == "hvm",
            ]
        )

    results = [item for item in table if filters(item)]

    if len(results) > 1:
        print(f"Found multiple AMIs for {region}!", file=sys.stderr)
    if len(results) == 0:
        print(f"Failed to find AMI for {region}!", file=sys.stderr)
        return None

    return results[0][7]


def update_tag_for_image_type(subconf: Dict[str, str], new_tag: str) -> bool:
    if new_tag == subconf["new"]:
        return False

    subconf["old"] = subconf["new"]
    subconf["new"] = new_tag
    return True


def update_bumpenvs_yaml(table: List[List[str]], path: str) -> None:
    with open(path) as f:
        bumpenvs_conf = yaml.safe_load(f)

    # All master-amis and bastion-amis in bumpenvs.yaml are updated.
    # The agent-amis are updated after each rebuild of the environments repo.
    for image_type, subconf in bumpenvs_conf.items():
        if image_type.endswith("_master_ami"):
            region = image_type[: -len("_master_ami")].replace("_", "-")
            # Master AMIs are based on Focal.
            new_ami = get_ubuntu_ami(table, "focal", region)
            if new_ami is not None:
                update_tag_for_image_type(subconf, new_ami)

        if image_type.endswith("_bastion_ami"):
            region = image_type[: -len("_bastion_ami")].replace("_", "-")
            # Bastion AMIs are based on Focal.
            new_ami = get_ubuntu_ami(table, "focal", region)
            if new_ami is not None:
                update_tag_for_image_type(subconf, new_ami)

    with open(path, "w") as f:
        yaml.dump(bumpenvs_conf, f)


def update_packer_json(table: List[List[str]], path: str) -> None:
    with open(path) as f:
        packer_conf = json.load(f)

    # There are two specific keys we set in the environments-packer.json file.
    new_ami = get_ubuntu_ami(table, "focal", "us-west-2")
    packer_conf["variables"]["aws_base_image"] = new_ami
    new_ami = get_ubuntu_ami(table, "focal", "us-gov-west-1")
    packer_conf["variables"]["gov_aws_base_image"] = new_ami

    with open(path, "w") as f:
        json.dump(packer_conf, f, indent="  ")
        # json.dump() leaves off the final newline.
        f.write("\n")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter
    )
    parser.add_argument("--bumpenvs-yaml", metavar="/PATH/TO/BUMPENVS.YAML")
    parser.add_argument("--packer-json", metavar="/PATH/TO/ENIRONMENTS-PACKER.JSON")
    args = parser.parse_args()

    if not args.bumpenvs_yaml and not args.packer_json:
        parser.print_help(sys.stderr)
        sys.exit(1)

    release = "focal"
    req_url = f"https://cloud-images.ubuntu.com/query/{release}/server/released.current.txt"
    gov_req_url = (
        f"https://cloud-images.ubuntu.com/query.govcloud/{release}/server/released.current.txt"
    )

    req = requests.get(req_url)
    req.raise_for_status()
    table = [re.split(r"\t", row) for row in re.split(r"\n", req.text)[:-1]]
    gov_req = requests.get(gov_req_url)
    gov_req.raise_for_status()
    gov_table = [re.split(r"\t", row) for row in re.split(r"\n", gov_req.text)[:-1]]
    table += gov_table

    if args.bumpenvs_yaml:
        update_bumpenvs_yaml(table, args.bumpenvs_yaml)

    if args.packer_json:
        update_packer_json(table, args.packer_json)
