import json
import os
import shutil
import subprocess
import sys
import time
from pathlib import Path
from typing import Any, Dict, List, Optional

import googleapiclient.discovery
from google.auth.exceptions import DefaultCredentialsError
from google.cloud import storage
from googleapiclient.errors import HttpError
from tabulate import tabulate
from termcolor import colored

from determined import util
from determined.cli import render
from determined.cli.errors import CliError
from determined.common import util as common_util
from determined.deploy import healthcheck

from .preflight import check_quota

TF_VARS_FILE = "terraform.tfvars.json"
TF_STATE_FILE = "terraform.tfstate"
TF_GCS_CONFIG = """
terraform {
  backend "gcs" {
    bucket = "%s"
    prefix = "%s"
  }
}
"""


def deploy(configs: Dict, env: Dict, variables_to_exclude: List, dry_run: bool = False) -> None:
    set_validate_gcp_credentials(configs)
    if not configs.get("no_preflight_checks"):
        check_quota(configs)

    terraform_init(configs, env)
    if dry_run:
        terraform_plan(configs, env, variables_to_exclude)
    else:
        terraform_apply(configs, env, variables_to_exclude)


def dry_run(configs: Dict, env: Dict, variables_to_exclude: List) -> None:
    return deploy(configs, env, variables_to_exclude, dry_run=True)


def terraform_dir(configs: Dict) -> str:
    return os.path.join(configs["local_state_path"], "terraform")


def get_terraform_vars_file_path(configs: Dict) -> str:
    return os.path.join(configs["local_state_path"], TF_VARS_FILE)


def terraform_read_variables(vars_file_path: str) -> Dict:
    if not os.path.exists(vars_file_path):
        print(f"ERROR: Terraform variables file does not exist: {vars_file_path}")
        sys.exit(1)

    with open(vars_file_path, "r") as f:
        variables = json.load(f)
        assert isinstance(variables, dict), "expected a dict of variables"
        return variables


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
    configs["det_version_key"] = det_version.replace(".", "-")[:12].rstrip("-")

    # Track the default zone in configuration variables. This is needed
    # during deprovisioning.
    if "zone" not in configs:
        configs["zone"] = f"{configs['region']}-b"

    vars_file_path = get_terraform_vars_file_path(configs)

    tf_vars = {k: configs[k] for k in configs if k not in variables_to_exclude}
    with open(vars_file_path, "w") as f:
        json.dump(tf_vars, f)

    return vars_file_path


def terraform_init(configs: Dict, env: Dict) -> None:
    # Copy module definitions to local state directory. By using the local state
    # path as the current working directory and copying module definitions to it
    # we don't have to rely on users running `det deploy gcp up/down` from
    # different directories or with different Python environments.
    if os.path.exists(terraform_dir(configs)):
        util.rmtree_nfs_safe(terraform_dir(configs))

    shutil.copytree(
        os.path.join(os.path.dirname(os.path.abspath(__file__)), "terraform"),
        terraform_dir(configs),
    )

    state_bucket = configs.get("tf_state_gcs_bucket_name")
    if state_bucket:
        with (Path(terraform_dir(configs)) / "override.tf").open("w") as fout:
            fout.write(TF_GCS_CONFIG % (state_bucket, configs["cluster_id"]))

    command = ["terraform", "init"]
    if not state_bucket:
        command += [
            "-backend-config=path={}".format(
                os.path.join(configs["local_state_path"], "terraform.tfstate")
            )
        ]

    run_command(command, env, cwd=terraform_dir(configs))


def terraform_plan(configs: Dict, env: Dict, variables_to_exclude: List) -> None:
    vars_file_path = terraform_write_variables(configs, variables_to_exclude)

    command = ["terraform", "plan"]
    command += ["-input=false"]
    command += [f"-var-file={vars_file_path}"]

    run_command(command, env, cwd=terraform_dir(configs))


def terraform_apply(configs: Dict, env: Dict, variables_to_exclude: List) -> None:
    vars_file_path = terraform_write_variables(configs, variables_to_exclude)

    command = ["terraform", "apply"]
    command += ["-input=false"]
    command += ["-auto-approve"]
    command += [f"-var-file={vars_file_path}"]

    run_command(command, env, cwd=terraform_dir(configs))


def terraform_output(configs: Dict, env: Dict, variable_name: str) -> Any:
    if configs.get("tf_state_gcs_bucket_name"):
        command = [
            "terraform",
            "state",
            "pull",
        ]
        state_data_json = subprocess.check_output(command, env=env, cwd=terraform_dir(configs))
        state_data = json.loads(state_data_json)
        return state_data.get("outputs", {}).get(variable_name, {}).get("value")

    state_file_path = os.path.join(configs["local_state_path"], TF_STATE_FILE)
    command = [
        "terraform",
        "output",
        f"--state={state_file_path}",
        "--json",
        variable_name,
    ]

    # `terraform output` can take state file path, but it doesn't like
    # TF_DATA_DIR env variable.
    env_sanitized = {k: v for (k, v) in env.items() if k != "TF_DATA_DIR"}
    json_value = subprocess.check_output(command, env=env_sanitized)
    return json.loads(json_value)


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


def list_instances(compute: Any, tf_vars: Dict, filter_expr: str) -> Any:
    """Get list of instances for this deployment matching the given filter."""
    result = (
        compute.instances()
        .list(project=tf_vars.get("project_id"), zone=tf_vars.get("zone"), filter=filter_expr)
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
    filter_expr = f'name="{master_name(tf_vars)}"'
    instances = list_instances(compute, tf_vars, filter_expr)

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
    filter_expr = f'labels.managed-by="{master_name(tf_vars)}"'
    agent_instances = list_instances(compute, tf_vars, filter_expr)
    delete_instances(compute, tf_vars, agent_instances)


def delete(configs: Dict, env: Dict, no_prompt: bool) -> None:
    """Deprovision a given deployment.

    The order of operations for deprovisioning is:
      1. Stop master so that no more dynamic agents can be provisioned.
      2. Terminate all dynamic agents (which aren't managed by Terraform).
      3. Destroy all Terraform-managed resources.
    """
    vars_file_path = get_terraform_vars_file_path(configs)
    tf_vars = terraform_read_variables(vars_file_path)

    set_gcp_credentials_env(tf_vars)

    compute = googleapiclient.discovery.build("compute", "v1")

    stop_master(compute, tf_vars)
    terminate_running_agents(compute, tf_vars)

    command = ["terraform", "destroy"]

    command += ["-input=false"]

    if no_prompt:
        command += ["-auto-approve"]

    command += [f"-var-file={vars_file_path}"]

    run_command(command, env, cwd=terraform_dir(configs))


def run_command(command: List[str], env: Dict[str, str], cwd: Optional[str] = None) -> None:
    subprocess.check_call(command, env=env, stdout=sys.stdout, cwd=cwd)


def set_validate_gcp_credentials(
    configs: Optional[Dict] = None, keypath: Optional[str] = None
) -> None:
    """Sets and validates GCP Credentials.
    - If det_configs are available, then uses that to set credentials
    - Else if only keypath is available, then uses it set credentials
    - If none are available/provided, validates if credentials are set and valid
    """
    if keypath:
        os.environ["GOOGLE_APPLICATION_CREDENTIALS"] = keypath
    if configs:
        vars_file_path = get_terraform_vars_file_path(configs)
        # Try to load google credentials from terraform vars when present.
        if os.path.exists(vars_file_path):
            tf_vars = terraform_read_variables(vars_file_path)
            set_gcp_credentials_env(tf_vars)

    try:
        googleapiclient.discovery.build("compute", "v1")
    except DefaultCredentialsError as exc:
        err = (
            colored("Unable to locate GCP credentials.", "red")
            + " Please set "
            + colored("GOOGLE_APPLICATION_CREDENTIALS", "yellow")
            + " or explicitly create credentials "
            + "and re-run the application. "
            + "For more information, please see "
            + "https://docs.determined.ai/latest/sysadmin-deploy-on-gcp/install-gcp.html#credential"
            + "s and "
            + "https://cloud.google.com/docs/authentication/getting-started"
        )
        raise CliError(err) from exc


def set_gcp_credentials_env(tf_vars: Dict) -> None:
    keypath = tf_vars.get("keypath")
    if keypath:
        os.environ["GOOGLE_APPLICATION_CREDENTIALS"] = keypath


def wait_for_master(configs: Dict, env: Dict, timeout: int = 300) -> None:
    master_url = terraform_output(configs, env, "Web-UI")
    healthcheck.wait_for_master_url(master_url, timeout)


def check_or_create_gcsbucket(project_id: str, keypath: Optional[str] = None) -> None:
    set_validate_gcp_credentials(keypath=keypath)
    bucket_name = project_id + "-determined-deploy"
    storage_service = googleapiclient.discovery.build("storage", "v1")
    try:
        storage_service.buckets().get(bucket=bucket_name).execute()
    except HttpError as err:
        if err.resp.status == 404:
            request_body = {
                "name": bucket_name,
            }
            storage_service.buckets().insert(project=project_id, body=request_body).execute()
        else:
            raise


def list_clusters(bucket_name: str, project_id: str, print_format: str = "table") -> None:
    set_validate_gcp_credentials()
    storage_client = storage.Client(project=project_id)
    blobs = storage_client.list_blobs(bucket_name)
    cluster_list = [["Cluster ID"]]
    for blob in blobs:
        json_data_string = blob.download_as_string()
        json_data = json.loads(json_data_string)
        if json_data.get("resources"):
            cluster_list.append([blob.name[:-16]])
    cluster_json = {"Clusters": [dict(zip(cluster_list[0], row)) for row in cluster_list[1:]]}

    if print_format == "json":
        render.print_json(cluster_json)
    elif print_format == "yaml":
        cluster_yaml = common_util.yaml_safe_dump(cluster_json)
        print(cluster_yaml)
    else:
        print(tabulate(cluster_list, headers="firstrow"))
    return
