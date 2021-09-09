#!/usr/bin/env python
import argparse
from contextlib import contextmanager
import logging
import time
import os
import pathlib
import requests
import subprocess
import sys
from typing import List

logger = logging.getLogger("e2e-tests")

root = subprocess.check_output(
    ["git", "rev-parse", "--show-toplevel"], encoding="utf-8"
)[:-1]
root_path = pathlib.Path(root)
webui_dir = root_path.joinpath("webui")
tests_dir = webui_dir.joinpath("tests")
reports_dir = tests_dir.joinpath("reports")
logs_dir = reports_dir.joinpath("logs")
videos_dir = reports_dir.joinpath("videos")
test_cluster_dir = tests_dir.joinpath("test-cluster")
devcluster_config_path = test_cluster_dir / 'devcluster.yaml'

CLEAR = "\033[39m"
BLUE = "\033[94m"
LOG_COLOR = BLUE


def run(cmd: List[str], config) -> None:
    logger.info(f"+ {' '.join(cmd)}")
    return subprocess.check_call(cmd, env=config["env"])


def run_forget(cmd: List[str], logfile, config) -> subprocess.Popen:
    out_target = logfile if logfile is not None else sys.stdout
    return subprocess.Popen(cmd, env=config["env"], stdout=out_target)


def run_ignore_failure(cmd: List[str], config):
    try:
        run(cmd, config)
    except subprocess.CalledProcessError:
        pass


def setup_reports_dir(config):
    run_ignore_failure(["rm", "-r", str(reports_dir)], config)
    run(["mkdir", "-p", str(logs_dir)], config)
    run(["mkdir", "-p", str(videos_dir)], config)


def is_cluster_up(config):
    try:
        r = requests.get(config['DET_MASTER'] + '/api/v1/master')
        return r.status_code < 300 and r.status_code >= 200
    except:
        return False


def wait_until(condition, timeout=1, *args):
    interval = 0.5
    start = time.time()
    while time.time() - start < timeout:
        if condition(*args):
            return True
        time.sleep(interval)
    return False


def setup_cluster(config, logfile=None):
    logger.info("setting up the cluster..")
    cluster_process = run_forget(
        ["devcluster", "-1", "-c", devcluster_config_path], logfile, config)
    if not wait_until(is_cluster_up, 60, config):
        raise Exception(f"cluster {config['DET_MASTER']} is unreachable")
    logger.info(f"cluster pid: {cluster_process.pid}")
    return cluster_process


def teardown_cluster(config, pid):
    logger.info("tearing down the cluster..")
    run_ignore_failure(["kill", str(pid)], config)


@contextmanager
def det_cluster(config):
    try:
        log_path = str(test_cluster_dir.joinpath("cluster.stdout.log"))
        with open(log_path, "w") as f:
            cluster_process = setup_cluster(config, f)
            yield
    finally:
        teardown_cluster(config, cluster_process.pid)


def pre_e2e_tests(config):
    if not is_cluster_up(config):
        raise Exception(f"cluster {config['DET_MASTER']} is unreachable")
    setup_reports_dir(config)
    run(
        ["python", str(tests_dir.joinpath("bin", "createUserAndExperiments.py"))],
        config,
    )


def run_e2e_tests(config):
    """ depends on:
    1. a brand new, exclusive cluster at config['DET_MASTER']
    2. pre_e2e_tests() to have seeded that cluster recently* """
    logger.info(f"testing against http://{config['DET_MASTER']}")
    command = [
        "npx",
        "gauge",
        "run",
        "--env",
        "ci",
        "specs"
    ]
    run(command, config)


def run_dev_tests(config):
    run(["npx", "gauge", "run", "--env", "dev", "specs"], config)


def e2e_tests(config):
    with det_cluster(config):
        pre_e2e_tests(config)
        run_e2e_tests(config)
        cleanup_video_artifacts(config)


def dev_tests(config):
    with det_cluster(config):
        pre_e2e_tests(config)
        run_dev_tests(config)
        cleanup_video_artifacts(config)


def cleanup_video_artifacts(config):
    items = os.listdir(videos_dir)
    for item in items:
        if item.endswith(tuple([".jpeg", ".jpg", ".png"])):
            os.remove(os.path.join(videos_dir, item))


def get_config(args):
    config = {}
    config["DET_PORT"] = args.det_port
    config["CLUSTER_NAME"] = f"det_test_{args.det_port}"
    config["DET_MASTER"] = f"{args.det_host}:{args.det_port}"

    env = os.environ.copy()
    env["DET_MASTER"] = config["DET_MASTER"]
    env["DET_PROJ"] = str(root_path)
    logging.basicConfig(
        level=(args.log_level or "INFO"),
        format=(args.log_format or f"{LOG_COLOR}%(message)s{CLEAR}"),
    )
    config["env"] = env
    return config


def main():
    operation_to_fn = {
        "setup-test-cluster": setup_cluster,
        "teardown-test-cluster": teardown_cluster,
        "pre-e2e-tests": pre_e2e_tests,
        "run-e2e-tests": run_e2e_tests,
        "run-dev-tests": run_dev_tests,
        "e2e-tests": e2e_tests,
        "dev-tests": dev_tests,
    }

    parser = argparse.ArgumentParser(description="Manage e2e tests.")
    help_msg = f"operation must be in {sorted(operation_to_fn.keys())}"
    parser.add_argument("operation", help=help_msg)
    parser.add_argument("--det-port", default="8081", help="det master port")
    parser.add_argument(
        "--det-host",
        default="http://localhost",
        help="det master address eg localhost or 192.168.1.2",
    )
    parser.add_argument("--log-level")
    parser.add_argument("--log-format")
    args = parser.parse_args()

    fn = operation_to_fn.get(args.operation)
    if fn is None:
        logger.error(f"{args.operation} is not a supported operation.")
        parser.print_help()
        sys.exit(1)

    config = get_config(args)

    fn(config)


if __name__ == "__main__":
    main()
