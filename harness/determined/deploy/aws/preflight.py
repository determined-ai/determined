import re
import sys
from typing import Any, Dict

import boto3
import pkg_resources
import yaml
from botocore.exceptions import ClientError
from termcolor import colored

from determined.common import util
from determined.deploy.errors import PreflightFailure

from . import constants
from .deployment_types.base import DeterminedDeployment

# There's no reliable way to map instance type to its quota category via an API.
# Lookup quota codes in AWS console at:
# https://us-west-2.console.aws.amazon.com/servicequotas/home/services/ec2/quotas
ON_DEMAND_QUOTA_CODES = {
    "F": "L-74FC7D96",
    "G": "L-DB2E81BA",
    "Inf": "L-1945791B",
    "P": "L-417A185B",
    "X": "L-7295265B",
    None: "L-1216C47A",  # Other / "Standard" instance types.
}

SPOT_QUOTA_CODES = {
    "F": "L-88CF9481",
    "G": "L-3819A6DF",
    "Inf": "L-B5D1601B",
    "P": "L-7212CCBC",
    "X": "L-E3A00192",
    None: "L-34B43A08",
}


def get_instance_type_quota_code(instance_type: str, spot: bool = False) -> str:
    match = re.match("([a-z]+)[0-9]+.+", instance_type)
    if not match:
        raise PreflightFailure("can't detect instance class")

    instance_class = match.group(1).capitalize()
    quota_map = SPOT_QUOTA_CODES if spot else ON_DEMAND_QUOTA_CODES
    quota_code = quota_map.get(instance_class, quota_map[None])

    return quota_code


def fetch_instance_type_quota(boto_session: boto3.session.Session, quota_code: str) -> int:
    try:
        client = boto_session.client("service-quotas")
        quota_data = client.get_service_quota(ServiceCode="ec2", QuotaCode=quota_code)
        return int(quota_data["Quota"]["Value"])
    except ClientError as ex:
        raise PreflightFailure("failed to fetch service quota: %s" % ex)


# CloudFront templates use a set of built-in functions such as
# `!FindInMap`, `!Equals`, `!Ref` etc.
# They can't be parsed by a simple yaml parser, but we can safely ignore them,
# since we only make use of the default parameters section,
# and it doesn't contain any such function calls.
class LoaderIgnoreUnknown(yaml.SafeLoader):
    def ignore_unknown(self, node: Any) -> None:
        return None


LoaderIgnoreUnknown.add_constructor(None, LoaderIgnoreUnknown.ignore_unknown)  # type: ignore


def get_default_cf_parameter(deployment_object: DeterminedDeployment, parameter: str) -> Any:
    with open(deployment_object.template_path) as fin:
        data = yaml.load(fin, Loader=LoaderIgnoreUnknown)

    return data["Parameters"][parameter]["Default"]


def get_cf_parameter(
    det_config: Dict[str, Any], deployment_object: DeterminedDeployment, parameter: str
) -> Any:
    if det_config[parameter] is not None:
        return det_config[parameter]

    return get_default_cf_parameter(deployment_object, parameter)


def check_quotas(det_config: Dict[str, Any], deployment_object: DeterminedDeployment) -> None:
    try:
        boto_session: boto3.session.Session = det_config[constants.cloudformation.BOTO3_SESSION]
        gpu_instance_type = get_cf_parameter(
            det_config, deployment_object, constants.cloudformation.COMPUTE_AGENT_INSTANCE_TYPE
        )
        max_agents = get_cf_parameter(
            det_config, deployment_object, constants.cloudformation.MAX_DYNAMIC_AGENTS
        )
        spot_enabled = get_cf_parameter(
            det_config, deployment_object, constants.cloudformation.SPOT_ENABLED
        )

        quota_code = get_instance_type_quota_code(gpu_instance_type, spot=spot_enabled)
        vcpu_quota = fetch_instance_type_quota(boto_session, quota_code=quota_code)

        mapping_fn = pkg_resources.resource_filename("determined.deploy.aws", "vcpu_mapping.yaml")
        with open(mapping_fn) as fin:
            mapping_data = util.safe_load_yaml_with_exceptions(fin)
            vcpu_mapping = {d["instanceType"]: d for d in mapping_data}

        if gpu_instance_type not in vcpu_mapping:
            raise PreflightFailure("unknown vCPU count for instance type")

        vcpus_required = int(vcpu_mapping[gpu_instance_type]["vcpu"] * max_agents)
    except PreflightFailure as ex:
        print(colored("Failed to check AWS instance quota: %s" % ex, "yellow"))
        return
    except Exception as ex:
        print(colored("Error while checking AWS instance quota: %s" % ex, "yellow"))
        return

    if vcpus_required > vcpu_quota:
        print(
            colored(
                "Insufficient AWS GPU agent instance quota (available: %s, required: %s)"
                % (vcpu_quota, vcpus_required),
                "red",
            )
        )
        print(
            "You can request a quota increase at "
            "https://%s.console.aws.amazon.com/servicequotas/home/services/ec2/quotas"
            % boto_session.region_name
        )
        print("Required quota code: %s" % quota_code)
        print("This check can be skipped via `det deploy --no-preflight-checks ...`")
        sys.exit(1)
