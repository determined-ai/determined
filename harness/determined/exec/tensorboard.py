import os
import subprocess
import sys
import time
from typing import Callable, List, Tuple

import boto3
import requests


def set_s3_region() -> None:
    bucket = os.environ.get("AWS_BUCKET")
    if bucket is None:
        return

    endpoint_url = os.environ.get("DET_S3_ENDPOINT_URL", None)
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


def get_tensorboard_version(version: str) -> Tuple[str, str]:
    """
    Gets the version of the tensorboard package currently installed. Used
    by downstream processes to determine args passed in.
    :return: version in the form of (major, minor) tuple
    """

    major, minor, _ = version.split(".")

    return major, minor


def get_tensorboard_args(args: List[str]) -> List[str]:
    """
    Builds tensorboard startup args from args passed in from tensorboard-entrypoint.sh
    Args are added and deprecated at the mercy of tensorboard; all of the below are necessary to
    support versions 1.14, 2.4, and 2.5

    - If multiple directories are specified and the tensorboard version is > 1,
    use legacy logdir_spec behavior

    - Tensorboard 2+ no longer exposes all ports. Must pass in "--bind_all" to expose localhost

    - Tensorboard 2.5.0 introduces an experimental feature (default load_fast=true)
    which prevents multiple plugins from loading correctly.
    """
    task_id = os.environ["DET_TASK_ID"]
    port = os.environ["TENSORBOARD_PORT"]

    version = args.pop(0)

    # logdir is the second argument passed in from tensorboard_manager.go. If multiple directories
    # are specified and the tensorboard version is > 1, use legacy logdir_spec behavior. NOTE:
    # legacy logdir_spec behavior is not supported by many tensorboard plugins
    logdir = args.pop(0)

    tensorboard_args = [
        "tensorboard",
        f"--port={port}",
        f"--path_prefix=/proxy/{task_id}",
        *args,
    ]

    major, minor = get_tensorboard_version(version)

    if major == "2":
        tensorboard_args.append("--bind_all")
        if minor >= "5":
            tensorboard_args.append("--load_fast=false")
        if len(logdir.split(",")) > 1:
            tensorboard_args.append(f"--logdir_spec={logdir}")
            return tensorboard_args

    tensorboard_args.append(f"--logdir={logdir}")
    return tensorboard_args


def main(args: List[str]) -> int:
    set_s3_region()

    task_id = os.environ["DET_TASK_ID"]
    port = os.environ["TENSORBOARD_PORT"]
    tensorboard_addr = f"http://localhost:{port}/proxy/{task_id}"
    url = f"{tensorboard_addr}/data/plugin/scalars/tags"
    tensorboard_args = get_tensorboard_args(args)

    print(f"Running: {tensorboard_args}")
    p = subprocess.Popen(tensorboard_args)

    def still_alive() -> bool:
        return p.poll() is None

    if not wait_for_tensorboard(600, url, still_alive):
        p.kill()

    return p.wait()


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
