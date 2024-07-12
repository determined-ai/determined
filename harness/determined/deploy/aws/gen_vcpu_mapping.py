import argparse
import json
import pathlib
from typing import Dict, Iterable, List, Tuple

import boto3

from determined.common import util


def _fetch_vcpu_mapping() -> Iterable[Tuple[str, Dict]]:
    # Price List api is only available in us-east-1 and ap-southeast-1.
    client = boto3.client("pricing", region_name="us-east-1")
    for page in client.get_paginator("get_products").paginate(
        ServiceCode="AmazonEC2",
        MaxResults=100,
        Filters=[
            {
                "Field": "regionCode",
                "Type": "TERM_MATCH",
                "Value": "us-east-1",
            }
        ],
    ):
        for sku_str in page["PriceList"]:
            sku_data = json.loads(sku_str)
            try:
                attributes = sku_data["product"]["attributes"]
                data = {
                    "instanceType": attributes["instanceType"],
                    "vcpu": int(attributes["vcpu"]),
                }

                if "gpu" in attributes:
                    data["gpu"] = int(attributes["gpu"])

                yield (attributes["instanceType"], data)
            except KeyError:
                pass


def fetch_vcpu_mapping() -> List[Dict]:
    return [v for (_, v) in sorted(dict(_fetch_vcpu_mapping()).items())]


def main(args: argparse.Namespace) -> None:
    data = fetch_vcpu_mapping()
    with args.output_fn.open("w") as fout:
        util.yaml_safe_dump(data, fout)


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("output_fn", type=pathlib.Path, help="output filename")

    args = parser.parse_args()
    main(args)
