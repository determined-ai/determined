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

import determined as det
import determined.common
from determined.tensorboard import fetchers

TENSORBOARD_TRIGGER_READY_MSG = "TensorBoard contains metrics"
TRIGGER_WAITING_MSG = "TensorBoard waits on metrics"
TICK_INTERVAL = 1
MAX_WAIT_TIME = 600
TB_RESPONSE_WAIT_TIME = 300


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


def raise_if_dead(p: subprocess.Popen) -> None:
    ret_code = p.poll()
    if ret_code is not None:
        raise RuntimeError(f"Tensorboard process died, exit code({ret_code}).")


def start_tensorboard(
    storage_config: Dict[str, Any],
    tb_version: str,
    storage_paths: List[str],
    add_tb_args: List[str],
) -> int:
    """Start Tensorboard and look for new files."""
    with tempfile.TemporaryDirectory() as local_dir:
        # Get fetcher and perform initial fetch
        logger.debug(
            f"Building fetcher...\n"
            f"\tstorage_config: {storage_config}\n"
            f"\tstorage_paths: {storage_paths}\n"
            f"\tlocal_dir: {local_dir}"
        )
        fetcher = fetchers.build(storage_config, storage_paths, local_dir)

        # Build Tensorboard args and launch process.
        tb_args = get_tensorboard_args(tb_version, local_dir, add_tb_args)
        logger.debug(f"tensorboard args: {tb_args}")
        tensorboard_process = subprocess.Popen(tb_args)
        tb_fetch_manager = TBFetchManager()

        with det.util.forward_signals(tensorboard_process):
            try:
                tb_unresponsive_stop_time = time.time() + TB_RESPONSE_WAIT_TIME

                # Wait for the Tensorboard process to start responding before proceeding.
                responsive = False
                while not responsive:
                    raise_if_dead(tensorboard_process)

                    if time.time() > tb_unresponsive_stop_time:
                        raise RuntimeError("Tensorboard wasn't responsive before the timeout.")

                    time.sleep(TICK_INTERVAL)
                    responsive = check_tensorboard_responsive()

                # Continuously loop checking for new files
                stop_time = time.time() + MAX_WAIT_TIME
                while True:
                    raise_if_dead(tensorboard_process)

                    # Check if we have reached a timeout without downloading any files
                    if tb_fetch_manager.num_fetched_files == 0 and time.time() > stop_time:
                        raise RuntimeError("No new files were fetched before the timeout.")

                    time.sleep(TICK_INTERVAL)
                    # TODO: Note that this call is blocking and serial. We won't check
                    # the stop time until this completely finishes
                    fetcher.fetch_new(new_file_callback=tb_fetch_manager.on_file_fetched)

            finally:
                if tensorboard_process.poll() is None:
                    logger.info("Killing tensorboard process")
                    tensorboard_process.kill()


class TBFetchManager:
    def __init__(self) -> None:
        self._ready = False
        self.num_fetched_files = 0

    # TODO: If we support multi-threaded fetching in the future, this will
    # need a lock
    def on_file_fetched(self) -> None:
        if not self._ready:
            self._ready = True
            print(TENSORBOARD_TRIGGER_READY_MSG, flush=True)
        self.num_fetched_files += 1


if __name__ == "__main__":
    tb_version = sys.argv[1]
    storage_config_path = sys.argv[2]
    storage_paths = sys.argv[3].split(",")
    additional_tb_args = sys.argv[4:]

    config = {}  # type: Dict[str, Any]
    with open(storage_config_path) as config_file:
        storage_config = json.load(config_file)

    determined.common.set_logger(determined.common.util.debug_mode())
    logger.debug(
        f"Tensorboard (v{tb_version}) Initializing...\n"
        f"\tstorage_config_path: {storage_config_path}\n"
        f"\tstorage_paths: {storage_paths}\n"
        f"\tadditional_tb_args: {additional_tb_args}\n"
        f"\tstorage_config: {storage_config}"
    )

    ret = start_tensorboard(storage_config, tb_version, storage_paths, additional_tb_args)
    sys.exit(ret)
