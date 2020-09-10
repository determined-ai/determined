import json
import os
import subprocess
import sys
import time
from typing import Callable, List

import boto3
import requests


def set_s3_region(bucket: str) -> None:
    endpoint_url = os.environ.get("DET_S3_ENDPOINT", None)

    client = boto3.client("s3", endpoint_url=endpoint_url)
    bucketLocation = client.get_bucket_location(Bucket=bucket)

    region = bucketLocation["LocationConstraint"]

    if region is not None:
        # We have observed that in US-EAST-1 the region comes back as None
        # and if AWS_REGION is set to None, tensorboard fails to pull events.
        print(f"Setting AWS_REGION environment variable to {region}.")
        os.environ["AWS_REGION"] = str(region)


def wait_for_tensorboard(max_seconds: float, url: str, still_alive_fn: Callable[[], bool]) -> bool:
    """Return True if the process successfully comes up before a deadline."""

    deadline = time.time() + max_seconds

    while True:
        if time.time() > deadline:
            print(f"TensorBoard did not find metrics within {max_seconds} seconds", file=sys.stderr)
            return False

        if not still_alive_fn():
            print("TensorBoard process died before reporting metrics", file=sys.stderr)
            return False

        time.sleep(1)

        try:
            res = requests.get(url)
            res.raise_for_status()
        except (requests.exceptions.ConnectionError, requests.exceptions.HTTPError):
            continue

        try:
            tags = res.json()
        except ValueError:
            continue

        # TensorBoard will return { trial/<id> : { tag: value } } when data is present.
        if len(tags) == 0:
            print("TensorBoard is awaiting metrics...")
            continue

        for val in tags.values():
            if len(val):
                print("TensorBoard contains metrics")
                return True


def main(args: List[str]) -> int:
    with open("/run/determined/workdir/experiment_config.json") as f:
        exp_conf = json.load(f)

    if exp_conf["checkpoint_storage"]["type"] == "s3":
        set_s3_region(exp_conf["checkpoint_storage"]["bucket"])

    task_id = os.environ["DET_TASK_ID"]
    port = os.environ["TENSORBOARD_PORT"]
    tensorboard_addr = f"http://localhost:{port}/proxy/{task_id}"
    url = f"{tensorboard_addr}/data/plugin/scalars/tags"

    print(f"Running: tensorboard --port{port} --path_prefix=/proxy/{task_id}", *args)
    p = subprocess.Popen(
        ["tensorboard", f"--port={port}", f"--path_prefix=/proxy/{task_id}", *args]
    )

    def still_alive() -> bool:
        return p.poll() is None

    if not wait_for_tensorboard(600, url, still_alive):
        p.kill()

    return p.wait()


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
