import json
import logging
import os
import subprocess
import sys
import time
import uuid
from typing import Callable, List, Tuple
from urllib.parse import urlparse

import boto3
import requests
from requests.exceptions import ConnectionError, HTTPError

TENSORBOARD_TRIGGER_READY_MSG = "TensorBoard contains metrics"
EXP_CONFIG_JSON_PATH = "/run/determined/workdir/experiment_config.json"

logger = logging.getLogger("determined.exec.tensorboard")
formatter = logging.Formatter("%(levelname)s - %(message)s")
stderr_handler = logging.StreamHandler(sys.stderr)
stderr_handler.setFormatter(formatter)
logger.addHandler(stderr_handler)

logger.setLevel(logging.DEBUG)


def get_tensorboard_args(tb_version, tfevents_dir, add_args):
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
    version_parts = tb_version.split(".")
    major = int(version_parts[0])
    minor = int(version_parts[1])

    if major >= 2:
        tensorboard_args.append("--bind_all")
    if major > 2 or major == 2 and minor >= 5:
        tensorboard_args.append("--load_fast=false")

    tensorboard_args.append(f"--logdir={tfevents_dir}")

    return tensorboard_args


def get_tensorboard_url():
    task_id = os.environ["DET_TASK_ID"]
    port = os.environ["TENSORBOARD_PORT"]
    tensorboard_addr = f"http://localhost:{port}/proxy/{task_id}"
    return f"{tensorboard_addr}/data/plugin/scalars/tags"


class S3Fetcher:
    def __init__(self, storage_config, temp_dir):
        req_keys = ["bucket", "access_key", "secret_key", "endpoint_url"]
        for key in req_keys:
            try:
                storage_config[key]
            except KeyError:
                raise ValueError(f"storage_config must define a '{key}'")

        self.client = boto3.client(
            "s3",
            endpoint_url=storage_config["endpoint_url"],
            aws_access_key_id=storage_config["access_key"],
            aws_secret_access_key=storage_config["secret_key"],
        )

        self.bucket = storage_config["bucket"]
        self.temp_dir = temp_dir
        self._tfevents_files = {}

    def _find_keys(self, s3_path, recursive=True):
        """Generates tuples of (s3_key, datetime.datetime)."""

        logger.debug(f"Listing keys at '{s3_path}'")

        key = urlparse(s3_path).path.lstrip("/")

        list_args = {}
        while True:
            list_dict = self.client.list_objects_v2(
                Bucket=self.bucket,
                Prefix=key,
                **list_args,
            )
            logger.debug(f"list_objects_v2 response dict: {list_dict}")

            for s3_obj in list_dict.get("Contents", []):
                yield (s3_obj["Key"], s3_obj["LastModified"])

            list_args.pop("ContinuationToken", None)
            if list_dict["IsTruncated"]:
                list_args["ContinuationToken"] = list_dict["NextContinuationToken"]
                continue

            break

    def download_new_files(self):
        keys_to_download = []

        # Look at all files in our storage location.
        for path in paths:
            logger.debug(f"Looking at path: {path}")

            for key, mtime in self._find_keys(path):
                prev_mtime = self._tfevents_files.get(key, None)

                if prev_mtime is not None and prev_mtime >= mtime:
                    logger.debug(f"File not new '{key}'")
                    continue

                logger.debug(f"Found new file '{key}'")
                keys_to_download.append(key)
                self._tfevents_files[key] = mtime

        # Download the new or updated files.
        for key in keys_to_download:
            bucket_key = f"{self.bucket}/{key}"
            local_path = os.path.join(self.temp_dir, bucket_key)

            dir_path = os.path.dirname(local_path)
            os.makedirs(dir_path, exist_ok=True)

            with open(local_path, "wb+") as local_file:
                self.client.download_fileobj(self.bucket, key, local_file)

            logger.debug(f"Downloaded file to local: {local_path}")


def build_storage_fetcher(temp_dir):
    """Setup Storage Connector"""
    with open(EXP_CONFIG_JSON_PATH) as f:
        exp_conf = json.load(f)

    checkpoint_storage = exp_conf.get("checkpoint_storage", None)
    if checkpoint_storage is None:
        raise ValueError("chechpoint_storage not defined in config")

    storage_type = checkpoint_storage.get("type", None)
    if storage_type is None:
        raise ValueError("checkpoint_storage must define a 'type'")

    storage_fetchers = {
        "s3": S3Fetcher,
        # XXX impl others.
    }

    if storage_type not in storage_fetchers.keys():
        raise NotImplementedError(f"checkpoint_storage.type == {storage_type} not impl")

    return storage_fetchers[storage_type](checkpoint_storage, temp_dir)


def check_for_metrics():
    tensorboard_url = get_tensorboard_url()
    tags = {}
    try:
        # Attempt to retrieve metrics from tensorboard.
        res = requests.get(tensorboard_url)
        res.raise_for_status()
        logger.debug(f"requests.get({tensorboard_url}) -> Response.content: {res.content}")
        tags = res.json()

    except (ConnectionError, HTTPError) as exp:
        logger.warning(f"OK: Tensorboard not responding to HTTP: {exp}")

    except ValueError as exp:
        logger.warning(str(exp))

    except json.JSONDecodeError as exp:
        logger.warning(f"OK: Could not JSONDecode Tensorboard HTTP response: {exp}")

    if len(tags) != 0 and any([len(v) for v in tags.values()]):
        print(TENSORBOARD_TRIGGER_READY_MSG)
        return True

    return False


def start_tensorboard(tb_version, paths, add_tb_args):

    # Create local temporary directory
    script_dir = os.path.dirname(__file__)
    temp_dir = os.path.join(script_dir, f"tb_events_{str(uuid.uuid4())}")
    os.makedirs(temp_dir, mode=0o777, exist_ok=True)

    # Get fetcher and perform initial fetch
    fetcher = build_storage_fetcher(temp_dir)
    fetcher.download_new_files()

    # Build Tensorboard args and launch process.
    tb_args = get_tensorboard_args(tb_version, temp_dir, add_tb_args)
    logger.debug(f"tensorboard args: {tb_args}")
    tensorboard_process = subprocess.Popen(tb_args)

    # Loop state.
    stop_time = time.time() + 600
    tb_has_metrics = False

    try:
        while True:
            # Check if tensorboard process is still alive.
            ret_code = tensorboard_process.poll()
            if ret_code is not None:
                raise RuntimeError(f"Tensorboard process died, exit code({ret_code}).")

            # Check if we have reached a timeout without receiving metrics
            if not tb_has_metrics and time.time() > stop_time:
                raise RuntimeError("We reached the timeout without receiving metrics.")

            if not tb_has_metrics:
                tb_has_metrics = check_for_metrics()

            fetcher.download_new_files()
            time.sleep(1)

    except Exception as exp:
        logger.error(str(exp))

    finally:
        if tensorboard_process.poll() is None:
            tensorboard_process.kill()

    return tensorboard_process.wait()


if __name__ == "__main__":
    tb_version = sys.argv[1]
    paths = sys.argv[2].split(",")
    additional_tb_args = sys.argv[3:]

    ret = start_tensorboard(tb_version, paths, additional_tb_args)
    sys.exit(ret)
