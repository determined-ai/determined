#!/usr/bin/env python
import argparse
from contextlib import contextmanager
import logging
import time
import os
import pathlib
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
results_dir = tests_dir.joinpath("results")
test_cluster_dir = tests_dir.joinpath("test-cluster")

CLUSTER_CMD_PREFIX = ["make", "-C", str(test_cluster_dir)]

CLEAR = "\033[39m"
BLUE = "\033[94m"
LOG_COLOR = BLUE


def run(cmd: List[str], config) -> None:
    logger.info(f"+ {' '.join(cmd)}")
    return subprocess.check_call(cmd, env=config["env"])


def run_forget(cmd: List[str], logfile, config) -> None:
    return subprocess.Popen(cmd, stdout=logfile)


def run_ignore_failure(cmd: List[str], config):
    try:
        run(cmd, config)
    except subprocess.CalledProcessError:
        pass


def setup_results_dir(config):
    run_ignore_failure(["rm", "-r", str(results_dir)], config)
    run(["mkdir", "-p", str(results_dir)], config)


def setup_cluster(logfile, config):
    logger.info("setting up the cluster..")
    run(CLUSTER_CMD_PREFIX + ["start-db"], config)
    cluster_process = run_forget(CLUSTER_CMD_PREFIX + ["run"], logfile, config)
    time.sleep(5)  # FIXME add a ready check for master
    logger.info(f"cluster pid: {cluster_process.pid}")
    return cluster_process


def teardown_cluster(config):
    logger.info("tearing down the cluster..")
    # FIXME
    run_ignore_failure(["pkill", "determined"], config)
    run_ignore_failure(["pkill", "run-server"], config)

    run(CLUSTER_CMD_PREFIX + ["stop-db"], config)


@contextmanager
def det_cluster(config):
    try:
        log_path = str(test_cluster_dir.joinpath("cluster.stdout.log"))
        with open(log_path, "w") as f:
            yield setup_cluster(f, config)

    finally:
        teardown_cluster(config)


def pre_e2e_tests(config):
    # TODO add a check for cluster condition
    setup_results_dir(config)
    run(
        ["python", str(tests_dir.joinpath("bin", "createUserAndExperiments.py"))],
        config,
    )


# _cypress_arguments generates an array of cypress arguments.
def _cypress_arguments(cypress_configs, config):
    base_url_config = f"baseUrl=http://{config['DET_MASTER']}"
    timeout_config = (
        f"defaultCommandTimeout={config['CYPRESS_DEFAULT_COMMAND_TIMEOUT']}"
    )
    args = [
        "--config-file",
        "cypress.json",
        "--config",
        ",".join([timeout_config, base_url_config, *cypress_configs]),
        "--browser",
        "chrome",
        "--headless",
    ]

    if config["CYPRESS_ARGS"]:
        args.extend(config["CYPRESS_ARGS"].split(" "))

    return args


def run_e2e_tests(config):
    """ depends on:
    1. a brand new, exclusive cluster at config['DET_MASTER']
    2. pre_e2e_tests() to have seeded that cluster recently* """
    logger.info(f"testing against http://{config['DET_MASTER']}")
    cypress_arguments = _cypress_arguments([], config)
    command = [
        "npx",
        "cypress",
        "run",
        *cypress_arguments,
    ]
    run(command, config)


def cypress_open(config):
    base_url_config = f"baseUrl=http://{config['DET_MASTER']}"
    run(["npx", "cypress", "open", "--config", base_url_config], config)


def e2e_tests(config):
    setup_results_dir(config)
    with det_cluster(config):
        pre_e2e_tests(config)
        run_e2e_tests(config)


def dev_tests(config):
    with det_cluster(config):
        pre_e2e_tests(config)
        cypress_open(config)


def get_config(args):
    config = {}
    config["DET_PORT"] = args.det_port
    config["CLUSTER_NAME"] = f"det_test_{args.det_port}"
    config["DET_MASTER"] = f"{args.det_host}:{args.det_port}"
    config["CYPRESS_DEFAULT_COMMAND_TIMEOUT"] = args.cypress_default_command_timeout
    config["CYPRESS_ARGS"] = args.cypress_args

    env = {}
    for var in ["DISPLAY", "PATH", "XAUTHORITY", "TERM"]:
        value = os.environ.get(var)
        if value is not None:
            env[var] = value
    env["DET_MASTER"] = config["DET_MASTER"]
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
        "cypress-open": cypress_open,
        "e2e-tests": e2e_tests,
        "dev-tests": dev_tests,
    }

    parser = argparse.ArgumentParser(description="Manage e2e tests.")
    help_msg = f"operation must be in {sorted(operation_to_fn.keys())}"
    parser.add_argument("operation", help=help_msg)
    parser.add_argument("--det-port", default="8081", help="det master port")
    parser.add_argument(
        "--det-host",
        default="localhost",
        help="det master address eg localhost or 192.168.1.2",
    )
    parser.add_argument("--cypress-default-command-timeout", default="4000")
    parser.add_argument("--cypress-args", help="other cypress arguments")
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
