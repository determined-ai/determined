import json
import logging
import os
import subprocess
import sys
import tempfile
import time
from typing import Any, Dict, List

import boto3
import requests
from packaging import version

import determined.common
from determined.tensorboard import fetchers

TENSORBOARD_TRIGGER_READY_MSG = "TensorBoard contains metrics"
FETCH_INTERVAL = 1
MAX_WAIT_TIME = 600


logger = logging.getLogger("determined.exec.tensorboard")


def set_s3_region() -> None:
    bucket = os.environ.get("AWS_BUCKET")
    if bucket is None:
        return

    endpoint_url = os.environ.get("DET_S3_ENDPOINT_URL")
    client = boto3.client("s3", endpoint_url=endpoint_url)
    bucket_location = client.get_bucket_location(Bucket=bucket)

    region = bucket_location["LocationConstraint"]

    if region is not None:
        # We have observed that in US-EAST-1 the region comes back as None
        # and if AWS_REGION is set to None, tensorboard fails to pull events.
        print(f"Setting AWS_REGION environment variable to {region}.")
        os.environ["AWS_REGION"] = str(region)


def get_tensorboard_args(tb_version: str, tfevents_dir: str, add_args: List[str]) -> List[str]:
    """Build tensorboard startup args.

    Args are added and deprecated at the mercy of tensorboard; all of the below are necessary to
    support versions 1.14, 2.4, and 2.5

    - Tensorboard 2+ no longer exposes all ports. Must pass in "--bind_all" to expose localhost
    - Tensorboard 2.5.0 introduces an experimental feature (default load_fast=true)
    which prevents multiple plugins from loading correctly.
    """
    task_id = os.environ["DET_TASK_ID"]
    port = os.environ["TENSORBOARD_PORT"]

    tensorboard_args = [
        "tensorboard",
        f"--port={port}",
        f"--path_prefix=/proxy/{task_id}",
        *add_args,
    ]

    # Version dependant args
    if version.parse(tb_version) >= version.parse("2"):
        tensorboard_args.append("--bind_all")
    if version.parse(tb_version) >= version.parse("2.5"):
        tensorboard_args.append("--load_fast=false")

    tensorboard_args.append(f"--logdir={tfevents_dir}")

    return tensorboard_args


def get_tensorboard_url() -> str:
    task_id = os.environ["DET_TASK_ID"]
    port = os.environ["TENSORBOARD_PORT"]
    tensorboard_addr = f"http://localhost:{port}/proxy/{task_id}"
    return tensorboard_addr


def check_tensorboard_responsive() -> bool:
    # Ensure Tensorboard is responding to HTTP request to prevent 502 from master.
    tensorboard_url = get_tensorboard_url()
    try:
        # Attempt HTTP request to Tensorboard.
        res = requests.get(tensorboard_url)
        res.raise_for_status()
        return True

    except (requests.exceptions.ConnectionError, requests.exceptions.HTTPError, ValueError) as exp:
        logger.warning(f"Tensorboard not responding to HTTP: {exp}")

    return False


def start_tensorboard(
    config: Dict[str, Any],
    tb_version: str,
    storage_paths: List[str],
    add_tb_args: List[str],
) -> int:
    """Start Tensorboard and look for new files."""

    stop_time = time.time() + MAX_WAIT_TIME
    triggered = False
    responsive = False

    with tempfile.TemporaryDirectory() as local_dir:

        # Get fetcher and perform initial fetch
        fetcher = fetchers.build(config, storage_paths, local_dir)
        num_fetched_files = fetcher.fetch_new()

        # Build Tensorboard args and launch process.
        tb_args = get_tensorboard_args(tb_version, local_dir, add_tb_args)
        logger.debug(f"tensorboard args: {tb_args}")
        tensorboard_process = subprocess.Popen(tb_args)

        try:
            while True:
                ret_code = tensorboard_process.poll()
                if ret_code is not None:
                    raise RuntimeError(f"Tensorboard process died, exit code({ret_code}).")

                # Check if we have reached a timeout without receiving metrics
                if num_fetched_files == 0 and time.time() > stop_time:
                    raise RuntimeError("No new files were fetched before the timeout.")

                if not responsive:
                    if time.time() > stop_time:
                        raise RuntimeError("Tensorboard wasn't responsive before the timeout.")
                    responsive = check_tensorboard_responsive()

                if responsive and not triggered and num_fetched_files > 0:
                    print(TENSORBOARD_TRIGGER_READY_MSG)
                    triggered = True

                time.sleep(FETCH_INTERVAL)
                num_fetched_files += fetcher.fetch_new()

        finally:
            if tensorboard_process.poll() is None:
                logger.info("Killing tensorboard process")
                tensorboard_process.kill()

        return tensorboard_process.wait()


if __name__ == "__main__":
    tb_version = sys.argv[1]
    config_path = sys.argv[2]
    storage_paths = sys.argv[3].split(",")
    additional_tb_args = sys.argv[4:]

    config = {}
    with open(config_path) as config_file:
        config = json.load(config_file)

    determined.common.set_logger(determined.common.util.debug_mode())

    ret = start_tensorboard(config, tb_version, storage_paths, additional_tb_args)
    sys.exit(ret)
