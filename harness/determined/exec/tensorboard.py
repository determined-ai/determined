import argparse
import json
import os
import sys
import time
from typing import List

import boto3
import requests


def set_s3_region(bucket: str) -> None:
    client = boto3.client("s3")
    bucketLocation = client.get_bucket_location(Bucket=bucket)

    region = bucketLocation["LocationConstraint"]
    print(f"export AWS_REGION={region}")


def poll_tensorboard() -> str:
    task_id = os.environ["DET_TASK_ID"]
    port = os.environ["TENSORBOARD_PORT"]
    tensorboard_addr = f"http://localhost:{port}/proxy/{task_id}"

    while True:
        time.sleep(5)

        try:
            res = requests.get(f"{tensorboard_addr}/data/plugin/scalars/tags")
            tags = res.json()
        except Exception:
            continue

        # TensorBoard will return { trial/<id> : { tag: value } } when data is present.
        if len(tags) == 0:
            continue

        for val in tags.values():
            if len(val):
                return "TensorBoard contains metrics"


def main(args: List[str]) -> None:
    parser = argparse.ArgumentParser(description="Determined AI Tensorboard Entrypoint")
    parser.add_argument("command", type=str, choices=["hdfs", "s3", "service_ready"])

    conf = parser.parse_args(args)

    with open("/run/determined/workdir/experiment_config.json") as f:
        exp_conf = json.load(f)

    if conf.command == "s3":
        if exp_conf["checkpoint_storage"]["type"] == "s3":
            set_s3_region(exp_conf["checkpoint_storage"]["bucket"])

    elif conf.command == "service_ready":
        print(poll_tensorboard())
    else:
        raise Exception("Unknown Command")


if __name__ == "__main__":
    main(sys.argv[1:])
