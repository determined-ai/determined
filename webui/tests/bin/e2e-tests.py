#!/usr/bin/env python
import argparse
import logging
import os
import pathlib
import signal
import subprocess
import sys
from typing import List

import docker

DOCKER_CYPRESS_IMAGE = "cypress/included:4.3.0"
RESULTS_DIR_NAME = "results"
logger = logging.getLogger("e2e-tests")

root = subprocess.check_output(
    ["git", "rev-parse", "--show-toplevel"], encoding="utf-8"
)[:-1]
root_path = pathlib.Path(root)
webui_dir = root_path.joinpath("webui")
tests_dir = webui_dir.joinpath("tests")

client = docker.from_env()


def run(cmd: List[str], config) -> None:
    logger.info("+ %s", " ".join(cmd))
    subprocess.check_call(cmd, env=config["env"])


def run_cluster_cmd(subcommand: List[str], config):
    run(
        ["det-deploy", "local"]
        + subcommand
        + ["--cluster-name", config["CLUSTER_NAME"]],
        config,
    )


def run_ignore_failure(cmd: List[str], config):
    try:
        run(cmd, config)
    except subprocess.CalledProcessError:
        pass


def pre_e2e_tests(config):
    run_ignore_failure(["rm", "-r", str(tests_dir.joinpath(RESULTS_DIR_NAME))], config)
    run(["docker", "pull", DOCKER_CYPRESS_IMAGE], config)
    run_cluster_cmd(
        [
            "cluster-up",
            "--agents",
            "1",
            "--no-gpu",
            "--delete-db",
            "--no-autorestart",
            "--master-port",
            config["INTEGRATIONS_HOST_PORT"],
        ],
        config,
    )
    test_setup_path = tests_dir.joinpath("bin", "createUserAndExperiments.py")
    run(["python", str(test_setup_path)], config)


def _cypress_container_name(config):
    return config["CLUSTER_NAME"] + "_cypress"


def post_e2e_tests(config):
    clean_up_cypress(config)
    run_cluster_cmd(["cluster-down", "--delete-db"], config)


# _environment_variables passes docker specific environment variables
def _environment_variables(config):
    cluster_name = config["CLUSTER_NAME"]
    master_name = cluster_name + "_determined-master_1"
    env_vars = [
        "--env",
        f"DET_MASTER={master_name}:8080",
    ]

    if config["DISCREET_LOGS"]:
        env_vars.extend(["--env", "DISCREET_LOGS=1"])

    return env_vars


# _cypress_arguments generates an array of cypress arguments.
def _cypress_arguments(cypress_configs, config, use_docker):
    timeout_config = (
        f"defaultCommandTimeout={config['CYPRESS_DEFAULT_COMMAND_TIMEOUT']}"
    )
    config_file_name = "cypress-docker.json" if use_docker else "cypress.json"
    args = [
        "--config-file",
        config_file_name,
        "--config",
        ",".join([timeout_config, *cypress_configs]),
        "--browser",
        "chrome",
        "--headless",
    ]

    if config["CYPRESS_ARGS"]:
        args.extend(config["CYPRESS_ARGS"].split(" "))

    return args


def container_exists(name):
    return any(
        filter(
            lambda container: container.name == name, client.containers.list(all=True)
        )
    )


def clean_up_cypress(config):
    cypress_name = _cypress_container_name(config)
    if container_exists(cypress_name):
        # ensure that the cypress container is stopped and removed
        run(["docker", "container", "rm", "-f", cypress_name], config)


def run_e2e_tests(config):
    base_url_config = f"baseUrl=http://localhost:{config['INTEGRATIONS_HOST_PORT']}"
    cypress_arguments = _cypress_arguments([base_url_config], config, False)

    command = [
        "yarn",
        "--cwd",
        str(tests_dir),
        "run",
        "cypress",
        "run",
        *cypress_arguments,
    ]

    run(
        command, config,
    )


def docker_run_e2e_tests(config):
    cluster_name = config["CLUSTER_NAME"]
    master_name = cluster_name + "_determined-master_1"
    network_name = cluster_name + "_default"
    cypress_name = _cypress_container_name(config)

    env_vars = _environment_variables(config)

    base_url_config = f"baseUrl=http://{master_name}:8080"
    cypress_config = [base_url_config]
    cypress_arguments = _cypress_arguments(cypress_config, config, True)

    command = [
        "docker",
        "run",
        "--name",
        cypress_name,
        f"--network={network_name}",
        "--mount",
        f"type=bind,source={webui_dir},target=/webui",
        "-w",
        "/webui/tests",
        *env_vars,
        DOCKER_CYPRESS_IMAGE,
        *cypress_arguments,
    ]

    run(command, config)


def e2e_tests(config):
    try:
        pre_e2e_tests(config)
        run_e2e_tests(config)
    finally:
        post_e2e_tests(config)


def docker_e2e_tests(config):
    try:
        pre_e2e_tests(config)
        docker_run_e2e_tests(config)
    finally:
        post_e2e_tests(config)


# Defines a one time signal handler that reverts to the original handler after one interception.
def setup_onetime_sig_handler(sig, fn):
    original_handler = signal.getsignal(sig)

    def signal_handler(a, b):
        logger.info("received interrupt request. cleaning up..")
        signal.signal(sig, original_handler)
        fn()
        exit(0)

    signal.signal(sig, signal_handler)
    return signal_handler


def get_config(args):
    config = {}
    config["DISCREET_LOGS"] = args.discreet_logs
    config["INTEGRATIONS_HOST_PORT"] = args.integrations_host_port
    config[
        "CLUSTER_NAME"
    ] = f"determined_integrations_{config['INTEGRATIONS_HOST_PORT']}"
    config["INTEGRATIONS_RESOURCE_SUFFIX"] = (
        "_webui_tests_" + config["INTEGRATIONS_HOST_PORT"]
    )
    config["INTEGRATIONS_NETWORK"] = (
        "determined" + config["INTEGRATIONS_RESOURCE_SUFFIX"]
    )
    config["DET_DOCKER_MASTER_NODE"] = "localhost"
    config["CYPRESS_DEFAULT_COMMAND_TIMEOUT"] = args.cypress_default_command_timeout
    config["CYPRESS_ARGS"] = args.cypress_args

    env = {}
    for var in ["DISPLAY", "PATH", "XAUTHORITY", "TERM"]:
        value = os.environ.get(var)
        if value is not None:
            env[var] = value
    env["INTEGRATIONS_HOST_PORT"] = config["INTEGRATIONS_HOST_PORT"]
    env["INTEGRATIONS_RESOURCE_SUFFIX"] = config["INTEGRATIONS_RESOURCE_SUFFIX"]
    env["DET_MASTER"] = "localhost:" + config["INTEGRATIONS_HOST_PORT"]
    logging.basicConfig(
        level=(args.log_level or "INFO"), format=(args.log_format or "%(message)s")
    )
    config["env"] = env
    return config


def main():
    operation_to_fn = {
        "docker-run-e2e-tests": docker_run_e2e_tests,
        "pre-e2e-tests": pre_e2e_tests,
        "run-e2e-tests": run_e2e_tests,
        "post-e2e-tests": post_e2e_tests,
        "e2e-tests": e2e_tests,
        "docker-e2e-tests": docker_e2e_tests,
    }

    parser = argparse.ArgumentParser(description="Manage e2e tests.")
    help_msg = f"operation must be in {sorted(operation_to_fn.keys())}"
    parser.add_argument("operation", help=help_msg)
    parser.add_argument("--integrations-host-port", default="5300")
    parser.add_argument("--cypress-default-command-timeout", default="4000")
    parser.add_argument("--cypress-args", help="other cypress arguments")
    parser.add_argument("--log-level")
    parser.add_argument("--log-format")
    parser.add_argument("--discreet-logs")
    args = parser.parse_args()

    fn = operation_to_fn.get(args.operation)
    if fn is None:
        logger.error(f"{args.operation} is not a supported operation.")
        parser.print_help()
        sys.exit(1)

    config = get_config(args)

    setup_onetime_sig_handler(signal.SIGINT, lambda: post_e2e_tests(config))
    fn(config)


if __name__ == "__main__":
    main()
