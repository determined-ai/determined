import os
import subprocess
import sys
from typing import Dict, List

terraform_dir = os.path.join(os.path.dirname(os.path.abspath(__file__)), "terraform")


def deploy(configs: Dict, env: Dict, variables: List) -> None:
    terraform_init(configs, env)
    terraform_apply(configs, env, variables)


def dry_run(configs: Dict, env: Dict, variables: List) -> None:
    terraform_init(configs, env)
    terraform_plan(configs, env, variables)


def terraform_init(configs: Dict, env: Dict) -> None:
    command = ["terraform init"]
    command += [
        "-backend-config='path={}'".format(
            os.path.join(configs["local_state_path"], "terraform.tfstate")
        )
    ]

    command += [terraform_dir]

    output = subprocess.Popen(" ".join(command), env=env, shell=True, stdout=sys.stdout)
    output.wait()


def terraform_plan(configs: Dict, env: Dict, variables: List) -> None:
    command = ["terraform", "plan"]

    for key in configs:
        if key in variables:
            continue
        else:
            command += ["-var='{}={}'".format(key, configs[key])]

    command += ["-input=false"]
    command += [terraform_dir]

    run_command(" ".join(command), env)


def terraform_apply(configs: Dict, env: Dict, variables: List) -> None:
    command = ["terraform", "apply"]

    for key in configs:
        if key in variables:
            continue
        else:
            command += ["-var='{}={}'".format(key, configs[key])]

    command += ["-input=false"]
    command += ["-auto-approve"]
    command += [terraform_dir]

    run_command(" ".join(command), env)


def delete(configs: Dict, env: Dict, variables: List) -> None:
    command = ["terraform", "destroy"]

    for key in configs:
        if key in variables:
            continue
        else:
            command += ["-var='{}={}'".format(key, configs[key])]

    command += ["-input=false"]
    command += ["-auto-approve"]
    command += [terraform_dir]

    run_command(" ".join(command), env)


def run_command(command: str, env: Dict[str, str]) -> None:
    subprocess.check_call(command, env=env, shell=True, stdout=sys.stdout)
