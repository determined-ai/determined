import json
import os
import shutil
import subprocess
import sys
import time
from typing import Any, Dict, List, Optional

import googleapiclient.discovery

TF_VARS_FILE = "terraform.tfvars.json"


def deploy(configs: Dict, env: Dict, variables_to_exclude: List) -> None:
    terraform_init(configs, env)
    terraform_apply(configs, env, variables_to_exclude)


def dry_run(configs: Dict, env: Dict, variables_to_exclude: List) -> None:
    terraform_init(configs, env)
    terraform_plan(configs, env, variables_to_exclude)


def terraform_dir(configs: Dict) -> str:
    return os.path.join(configs["local_state_path"], "terraform")


def terraform_read_variables(vars_file_path: str) -> Dict:
    if not os.path.exists(vars_file_path):
        print(f"ERROR: Terraform variables file does not exist: {vars_file_path}")
        sys.exit(1)

    with open(vars_file_path, "r") as f:
        vars = json.load(f)
        assert isinstance(vars, dict), "expected a dict of variables"
        return vars


def terraform_write_variables(configs: Dict, variables_to_exclude: List) -> str:
    """Write out given config object as a Terraform variables JSON file.

    Persist variables to Terraform state directory.  These variables are used
    on apply / plan, and are required for deprovisioning.
    """
    det_version = configs.get("det_version")
    if not det_version or not isinstance(det_version, str):
        print("ERROR: Determined version missing or invalid")
        sys.exit(1)

    # Add GCP-friendly version key to configs. We persist this since it's used
    # across the cluster lifecycle: to name resources on provisioning, and to
    # filter for the master and dynamic agents on deprovisioning.
    configs["det_version_key"] = det_version.replace(".", "-")[0:8]

    # Track the default zone in configuration variables. This is needed
    # during deprovisioning.
    if "zone" not in configs:
        configs["zone"] = f"{configs['region']}-b"

    vars_file_path = os.path.join(configs["local_state_path"], TF_VARS_FILE)

    tf_vars = {k: configs[k] for k in configs if k not in variables_to_exclude}
    with open(vars_file_path, "w") as f:
        json.dump(tf_vars, f)

    return vars_file_path


def terraform_init(configs: Dict, env: Dict) -> None:
    # Copy module definitions to local state directory. By using the local state
    # path as the current working directory and copying module definitions to it
    # we don't have to rely on users running "det-deploy gcp up/down" from
    # different directories or with different Python environments.
    if os.path.exists(terraform_dir(configs)):
        shutil.rmtree(terraform_dir(configs))

    shutil.copytree(
        os.path.join(os.path.dirname(os.path.abspath(__file__)), "terraform"),
        terraform_dir(configs),
    )

    command = ["terraform init"]
    command += [
        "-backend-config='path={}'".format(
            os.path.join(configs["local_state_path"], "terraform.tfstate")
        )
    ]

    command += [terraform_dir(configs)]

    output = subprocess.Popen(" ".join(command), env=env, shell=True, stdout=sys.stdout)
    output.wait()


def terraform_plan(configs: Dict, env: Dict, variables_to_exclude: List) -> None:
    vars_file_path = terraform_write_variables(configs, variables_to_exclude)

    command = ["terraform", "plan"]

    command += ["-input=false"]
    command += [f"-var-file={vars_file_path}"]
    command += [terraform_dir(configs)]

    run_command(" ".join(command), env)


def terraform_apply(configs: Dict, env: Dict, variables_to_exclude: List) -> None:
    vars_file_path = terraform_write_variables(configs, variables_to_exclude)

    command = ["terraform", "apply"]

    command += ["-input=false"]
    command += ["-auto-approve"]
    command += [f"-var-file={vars_file_path}"]
    command += [terraform_dir(configs)]

    run_command(" ".join(command), env)


def wait_for_operations(compute: Any, tf_vars: Dict, operations: List) -> bool:
    """Wait up to ~15 minutes to confirm that all operations have completed."""

    # Track operation statuses
    statuses = [None] * len(operations)  # type: List[Optional[bool]]

    for _ in range(200):
        for i, operation in enumerate(operations):
            if statuses[i] is None:
                result = (
                    compute.zoneOperations()
                    .get(
                        project=tf_vars.get("project_id"),
                        zone=tf_vars.get("zone"),
                        operation=operation,
                    )
                    .execute()
                )
                if result["status"] == "DONE":
                    statuses[i] = True

        # Short circuit and return True iff all operations have succeeded
        if all(status for status in statuses):
            return True

        time.sleep(5)

    # We don't have success for all operations and have run out of time
    return False


def list_instances(compute: Any, tf_vars: Dict, filter: str) -> Any:
    """Get list of instances for this deployment matching the given filter."""
    result = (
        compute.instances()
        .list(project=tf_vars.get("project_id"), zone=tf_vars.get("zone"), filter=filter)
        .execute()
    )
    return result["items"] if "items" in result else []


def delete_instances(compute: Any, tf_vars: Dict, instances: List) -> None:
    """Terminate provided instances in this deployment."""
    instance_names = [instance["name"] for instance in instances]
    if instance_names:
        print(f"Terminating instances: {', '.join(instance_names)}")
        print("This may take a few minutes...")
        operations = []
        for instance_name in instance_names:
            response = delete_instance(compute, tf_vars, instance_name)
            operations.append(response["name"])

        succeeded = wait_for_operations(compute, tf_vars, operations)
        if succeeded:
            print(f"Successfully terminated instances: {', '.join(instance_names)}...")
        else:
            print(
                f"\nWARNING: Unable to confirm instance termination: {', '.join(instance_names)}\n"
            )


def delete_instance(compute: Any, tf_vars: Dict, instance_name: str) -> Any:
    """Terminate instance with given name (resource ID)."""
    return (
        compute.instances()
        .delete(project=tf_vars.get("project_id"), zone=tf_vars.get("zone"), instance=instance_name)
        .execute()
    )


def stop_instance(compute: Any, tf_vars: Dict, instance_name: str) -> Any:
    """Stop instance with given name (resource ID)."""
    return (
        compute.instances()
        .stop(project=tf_vars.get("project_id"), zone=tf_vars.get("zone"), instance=instance_name)
        .execute()
    )


def master_name(tf_vars: Dict) -> str:
    """Construct master name for provided Terraform deployment."""
    return f"det-master-{tf_vars.get('cluster_id')}-{tf_vars.get('det_version_key')}"


def stop_master(compute: Any, tf_vars: Dict) -> None:
    """Stop the master, waiting for operation to complete."""
    filter = f'name="{master_name(tf_vars)}"'
    instances = list_instances(compute, tf_vars, filter)

    if len(instances) == 0:
        print(f"WARNING: Unable to locate master: {master_name(tf_vars)}")
    elif len(instances) > 1:
        print(f"ERROR: Found more than one master named {master_name(tf_vars)}")
        sys.exit(1)
    else:
        instance_name = instances[0]["name"]
        print(f"Stopping master instance: {instance_name}...")
        response = stop_instance(compute, tf_vars, instance_name)
        succeeded = wait_for_operations(compute, tf_vars, [response["name"]])
        if succeeded:
            print(f"Successfully stopped master instance: {instance_name}")
        else:
            print(f"\nWARNING: Unable to confirm master instance stopped: {instance_name}\n")


def terminate_running_agents(compute: Any, tf_vars: Dict) -> None:
    """Terminate all dynamic agents, waiting for operation to complete."""
    filter = f'labels.managed-by="{master_name(tf_vars)}"'
    agent_instances = list_instances(compute, tf_vars, filter)
    delete_instances(compute, tf_vars, agent_instances)


def delete(configs: Dict, env: Dict) -> None:
    """Deprovision a given deployment.

    The order of operations for deprovisioning is:
      1. Stop master so that no more dynamic agents can be provisioned.
      2. Terminate all dynamic agents (which aren't managed by Terraform).
      3. Destroy all Terraform-managed resources.
    """
    vars_file_path = os.path.join(configs["local_state_path"], TF_VARS_FILE)
    tf_vars = terraform_read_variables(vars_file_path)

    keypath = tf_vars.get("keypath")
    if keypath:
        os.environ["GOOGLE_APPLICATION_CREDENTIALS"] = keypath

    compute = googleapiclient.discovery.build("compute", "v1")

    stop_master(compute, tf_vars)
    terminate_running_agents(compute, tf_vars)

    command = ["terraform", "destroy"]

    command += ["-input=false"]
    command += ["-auto-approve"]
    command += [f"-var-file={vars_file_path}"]
    command += [terraform_dir(configs)]

    run_command(" ".join(command), env)


def run_command(command: str, env: Dict[str, str]) -> None:
    subprocess.check_call(command, env=env, shell=True, stdout=sys.stdout)
