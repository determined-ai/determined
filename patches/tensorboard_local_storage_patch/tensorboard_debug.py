import json
import re
import os
import subprocess
import sys
import time
import uuid
import logging
from typing import Callable, List, Tuple
from requests.exceptions import (ConnectionError, HTTPError)

import boto3
import requests


TENSORBOARD_TRIGGER_READY_MSG = "TensorBoard contains metrics"
EXP_CONFIG_JSON_PATH = "/run/determined/workdir/experiment_config.json"

logger = logging.getLogger("determined.exec.tensorboard")
formatter = logging.Formatter("%(levelname)s - %(message)s")
stderr_handler = logging.StreamHandler(sys.stderr)
stderr_handler.setFormatter(formatter)
logger.addHandler(stderr_handler)

# logger.setLevel(logging.DEBUG)


def get_tb_args(tb_version, tfevents_dir, add_args):
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
        *add_args
    ]

    # Version dependant args
    version_parts = tb_version.split(".")
    major = int(version_parts[0])
    minor = int(version_parts[1])

    bind_all = False
    load_fast = True

    if major >= 2:
        bind_all = True
    if major > 2 or major == 2 and minor >= 2:
        load_fast = False

    if bind_all:
        tensorboard_args.append("--bind_all")
    if not load_fast:
        tensorboard_args.append("--load_fast=false")

    tensorboard_args.append(f"--logdir={tfevents_dir}")

    return tensorboard_args


def get_tensorboard_url():
    task_id = os.environ["DET_TASK_ID"]
    port = os.environ["TENSORBOARD_PORT"]
    tensorboard_addr = f"http://localhost:{port}/proxy/{task_id}"
    return f"{tensorboard_addr}/data/plugin/scalars/tags"


class S3Connector:
    def __init__(self, storage_config):
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

        self.delimiter = "/"
        self._s3_path_regex = re.compile(r"^s3://(?P<bucket>[^/]+)/(?P<key>.+)$")
        self._path_type = re.compile(r"^(?P<type>[^:]+):/(?P<path>/.+)$")

    def _path_to_bucket_key(self, s3_path):
        m = re.match(self._s3_path_regex, s3_path)
        if m is None:
            raise ValueError(f"{s3_path} is not s3://bucket/path")

        return m.group("bucket"), m.group("key")

    def split_path_type(self, s3_path):
        m = re.match(self._path_type, s3_path)
        if m is None:
            raise ValueError(f"{s3_path} is not s3://bucket/path")

        return m.group("type"), m.group("path")

    def to_local(self, s3_path, local_path, make_dirs=True):
        bucket, key = self._path_to_bucket_key(s3_path)

        if make_dirs:
            dir_path = os.path.dirname(local_path)
            os.makedirs(dir_path, exist_ok=True)

        with open(local_path, "wb+") as local_file:
            self.client.download_fileobj(bucket, key, local_file)

        logger.debug(f"Downloaded file to local: {local_path}")

    def from_local(self, local_path, s3_path):
        bucket, key = self._path_to_bucket_key(s3_path)
        raise NotImplementedError("Impl me")

    def list_files(self, s3_path, recursive=True):
        # Generates tuples of (s3_key, datetime.datetime)
        bucket, key = self._path_to_bucket_key(s3_path)

        logger.debug(f"Listing files at '{s3_path}'")

        list_args = {"Bucket": bucket, "Prefix": key}
        if not recursive:
            list_args["Delimiter"] = self.delimiter

        # XXX Does not handle "directories", i.e., prefixes
        while True:
            list_dict = self.client.list_objects_v2(**list_args)
            logger.debug(f"list_objects_v2 response dict: {list_dict}")

            for s3_obj in list_dict.get("Contents", []):
                key_path = f"s3://{bucket}/{s3_obj['Key']}"
                key_mtime = s3_obj["LastModified"]
                yield (key_path, key_mtime)

            list_args.pop("ContinuationToken", None)
            if list_dict["IsTruncated"]:
                list_args["ContinuationToken"] = list_dict["NextContinuationToken"]
                continue

            break


def build_storage_connector():
    """Setup Storage Connector"""
    with open(EXP_CONFIG_JSON_PATH) as f:
        exp_conf = json.load(f)

    checkpoint_storage = exp_conf.get("checkpoint_storage", None)
    if checkpoint_storage is None:
        raise ValueError("chechpoint_storage not defined in config")

    storage_type = checkpoint_storage.get("type", None)
    if storage_type is None:
        raise ValueError("checkpoint_storage must define a 'type'")

    storage_connectors = {
        "s3": S3Connector,
        # XXX impl others.
    }

    if storage_type not in storage_connectors.keys():
        raise NotImplementedError(
            f"checkpoint_storage.type == {storage_type} not impl"
        )

    return storage_connectors[storage_type](checkpoint_storage)


def start_tensorboard(tb_version, paths, add_tb_args):

    storage = build_storage_connector()
    tensorboard_url = get_tensorboard_url()

    # Create local temporary directory
    script_dir = os.path.dirname(__file__)
    temp_dir = os.path.join(script_dir, f"tb_events_{str(uuid.uuid4())}")
    os.makedirs(temp_dir, mode=0o777, exist_ok=True)

    def launch_tensorboard():
        tb_args = get_tb_args(tb_version, temp_dir, add_tb_args)
        logger.debug(f"tensorboard args: {tb_args}")
        return subprocess.Popen(tb_args)

    # Loop state
    tfevents_files = {}  # {filepath -> datetime}, update on iter
    stop_time = time.time() + 600
    tensorboard_process = None
    tb_has_metrics = False

    while True:
        s3_paths_to_download = []

        # Look at all files in our storage location.
        for path in paths:
            logger.debug(f"Looking at path: {path}")

            for s3_path, mtime in storage.list_files(path):
                prev_mtime = tfevents_files.get(s3_path, None)

                if prev_mtime is not None and prev_mtime >= mtime:
                    logger.debug(f"File not new '{s3_path}'")
                    continue

                logger.debug(f"Found new file '{s3_path}'")
                s3_paths_to_download.append(s3_path)
                tfevents_files[s3_path] = mtime

        # XXX Do we need to delete files no longer in storage?

        # Download the new or updated files.
        for s3_path in s3_paths_to_download:
            _, s3_path_part = storage.split_path_type(s3_path)
            local_path = os.path.join(temp_dir, s3_path_part.lstrip("/"))
            # XXX Could potentially async these calls for concurrency.
            storage.to_local(s3_path, local_path)

        # Launch tensorboard if it isn't started.
        if tensorboard_process is None:
            tensorboard_process = launch_tensorboard()
            if not isinstance(tensorboard_process, subprocess.Popen):
                raise RuntimeError("Failed to start Tensorboard subprocess")

        # Check if tensorboard process is still alive.
        ret_code = tensorboard_process.poll()
        if ret_code is not None:
            logger.error(f"Tensorboard process died, exit code({ret_code}).")
            return ret_code

        # Attempt to retrieve metrics from tensorboard.
        try:
            res = requests.get(tensorboard_url)
            res.raise_for_status()
            logger.debug(
                f"requests.get({tensorboard_url}) -> Response.content: {res.content}"
            )

            tags = res.json()

            if len(tags) == 0:
                raise ValueError("No metrics available, len(tags) == 0")

            if any([len(v) for v in tags.values()]):
                logger.info("Tensorboard has metrics!")
                print(TENSORBOARD_TRIGGER_READY_MSG)
                tb_has_metrics = True

        except (ConnectionError, HTTPError) as exp:
            logger.warning(f"OK: Tensorboard not responding to HTTP: {exp}")

        except ValueError as exp:
            logger.warning(str(exp))

        except json.JSONDecodeError as exp:
            logger.warning(f"OK: Could not JSONDecode Tensorboard HTTP response: {exp}")

        # Check if we have reached a timeout without receiving metrics
        if not tb_has_metrics and time.time() > stop_time:
            logger.error("We have reached the timeout without receiving metrics.")
            tensorboard_process.kill()
            return 1

        time.sleep(1)

    return 1


if __name__ == "__main__":
    tb_version = sys.argv[1]
    paths = sys.argv[2].split(",")
    additional_tb_args = sys.argv[3:]

    ret = start_tensorboard(tb_version, paths, additional_tb_args)
    sys.exit(ret)
