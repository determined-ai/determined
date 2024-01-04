import json
import logging
import os
import queue
import subprocess
import sys
import tempfile
import threading
import time
from typing import Any, Callable, Dict, List

import boto3
import requests
from packaging import version

import determined as det
import determined.common
from determined.tensorboard import fetchers

TENSORBOARD_TRIGGER_READY_MSG = "TensorBoard contains metrics"

TICK_INTERVAL = 1  # How many seconds to wait on each iteration of our check loop
MAX_WAIT_TIME = 600  # How many seconds to wait for the first metric file to download
TB_RESPONSE_WAIT_TIME = 300  # How many seconds to wait for TensorBoard to initially start up
WORK_QUEUE_MAX_SIZE = 20  # Size of the threading work queue for fetching
FULL_ITERATION_SLEEP_TIME = 20  # How long to wait between a full iteration run (in seconds)
NUM_FETCH_THREADS = 5  # Number of fetching threads to run concurrently
READY_SIGNAL_DELAY = 7  # How many seconds to wait before sending the ready signal

logger = logging.getLogger("determined")


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
        work_queue: queue.Queue = queue.Queue(maxsize=WORK_QUEUE_MAX_SIZE)

        iteration_thread = TBFetchIterationThread(
            fetcher=fetcher, work_queue=work_queue, daemon=True
        )
        fetch_threads = [
            TBFetchThread(
                fetcher=fetcher,
                work_queue=work_queue,
                new_file_callback=tb_fetch_manager.on_file_fetched,
                daemon=True,
            )
            for _ in range(NUM_FETCH_THREADS)
        ]

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
                iteration_thread.start()
                for fetch_thread in fetch_threads:
                    fetch_thread.start()
                while True:
                    raise_if_dead(tensorboard_process)

                    # Check if we have reached a timeout without downloading any files
                    if tb_fetch_manager.get_num_fetched_files() == 0:
                        if time.time() > stop_time:
                            raise RuntimeError("No new files were fetched before the timeout.")
                        else:
                            # TODO: This should trigger an actual task state change (DET-10001).
                            # For now, just print a message to the logs.
                            print("TensorBoard is waiting for metrics.", flush=True)
                    time.sleep(TICK_INTERVAL)

            finally:
                if tensorboard_process.poll() is None:
                    logger.info("Killing tensorboard process")
                    tensorboard_process.kill()


class TBFetchManager:
    """Simple Container Class to manage the state of the TensorBoard remote fetchers"""

    def __init__(self) -> None:
        self._lock = threading.Lock()
        self._ready = False
        self._num_fetched_files = 0

    def on_file_fetched(self) -> None:
        """Runs on each callback from a fetcher to update the state.
        We delay signalling the READY_SIGNAL to give TensorBoard a moment
        to do its internal book keeping and prevent the "no data"
        issues we were previously seeing when signalling ready immediately
        after the first file is fetched.

        TensorBoard >=v2.5.0 has a "load_fast=true" flag to alleviate this problem
        but for backwards compatibility reasons, we cannot use it yet.
        More info here: https://github.com/tensorflow/tensorboard/issues/4784
        """
        with self._lock:
            if not self._ready:
                self._ready = True
                t = threading.Timer(READY_SIGNAL_DELAY, self._emit_ready_signal)
                t.start()
            self._num_fetched_files += 1

    def _emit_ready_signal(self) -> None:
        print(TENSORBOARD_TRIGGER_READY_MSG, flush=True)

    def get_num_fetched_files(self) -> int:
        with self._lock:
            return self._num_fetched_files


class TBFetchIterationThread(threading.Thread):
    """Thread to continuously iterate over the fetchers files and add them to a threading.Queue

    Note: We are making the assumption that there will only be one of these running per process.
    If we add more, then the base fetcher will need to support locking around the _file_records
    dictionary. Defined in <ROOT>/harness/determined/tensorboard/fetchers/base.py
    """

    def __init__(
        self,
        fetcher: fetchers.Fetcher,
        work_queue: queue.Queue,
        *args: Any,
        **kwargs: Any,
    ) -> None:
        self._fetcher = fetcher
        self._work_queue = work_queue
        super().__init__(*args, **kwargs)

    def run(self) -> None:
        while True:
            try:
                for filepath in self._fetcher.list_all_generator():
                    self._work_queue.put(filepath, block=True)
            except Exception as e:
                logger.warning(
                    f"Failure listing TensorBoard files from {self._fetcher}. Error: {e}"
                    f" (retrying in {FULL_ITERATION_SLEEP_TIME}s)...",
                    exc_info=True,
                )
            finally:
                time.sleep(FULL_ITERATION_SLEEP_TIME)


class TBFetchThread(threading.Thread):
    """Thread to continuously read from the queue and fetch the files"""

    def __init__(
        self,
        fetcher: fetchers.Fetcher,
        work_queue: queue.Queue,
        new_file_callback: Callable,
        *args: Any,
        **kwargs: Any,
    ) -> None:
        self._fetcher = fetcher
        self._work_queue = work_queue
        self._new_file_callback = new_file_callback
        super().__init__(*args, **kwargs)

    def run(self) -> None:
        while True:
            try:
                filepath = self._work_queue.get(block=True)
                self._fetcher._fetch(filepath, self._new_file_callback)
            except Exception as e:
                logger.warning(
                    f"Timeout fetching TensorBoard files from {self._fetcher}. Error: {e}"
                    f" (retrying)...",
                    exc_info=True,
                )
                # Put the failed filepath back onto the list
                if filepath:
                    self._work_queue.put(filepath, block=True)


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
