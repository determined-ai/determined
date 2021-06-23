import sys
from typing import Dict

import googleapiclient.discovery
import googleapiclient.errors
from termcolor import colored

from determined.deploy.errors import PreflightFailure

ON_DEMAND_QUOTA_CODES = {
    "nvidia-tesla-t4": "NVIDIA_T4_GPUS",
    "nvidia-tesla-v100": "NVIDIA_V100_GPUS",
    "nvidia-tesla-p100": "NVIDIA_P100_GPUS",
    "nvidia-tesla-p4": "NVIDIA_P4_GPUS",
    "nvidia-tesla-k80": "NVIDIA_K80_GPUS",
    "nvidia-tesla-a100": "NVIDIA_A100_GPUS",
}

PREEMPTIBLE_QUOTA_CODES = {k: "PREEMPTIBLE_" + v for (k, v) in ON_DEMAND_QUOTA_CODES.items()}


def check_quota(configs: Dict) -> None:
    print("Checking quota...\n")
    try:
        try:
            compute = googleapiclient.discovery.build("compute", "v1")
            r = (
                compute.regions()
                .get(project=configs["project_id"], region=configs["region"])
                .execute()
            )
        except googleapiclient.errors.Error as ex:
            raise PreflightFailure("failed to fetch quota info: %s" % ex)

        if "quotas" not in r:
            raise PreflightFailure("no quota info available")

        mapping = PREEMPTIBLE_QUOTA_CODES if configs["preemptible"] else ON_DEMAND_QUOTA_CODES
        quota_code = mapping[configs["gpu_type"]]
        quota = next((q for q in r["quotas"] if q["metric"] == quota_code), None)
        if quota is None:
            raise PreflightFailure("can't find quota metric %s" % quota_code)

        gpu_quota = quota["limit"] - quota["usage"]
        gpu_required = configs["gpu_num"] * configs["max_dynamic_agents"]
    except PreflightFailure as ex:
        print(colored("Failed to check GCP instance quota: %s" % ex, "yellow"))
        return
    except Exception as ex:
        print(colored("Error while checking GCP instance quota: %s" % ex, "yellow"))
        return

    if gpu_required > gpu_quota:
        print(
            colored(
                "Insufficient GCP GPU agent instance quota (available: %s, required: %s)"
                % (gpu_quota, gpu_required),
                "red",
            )
        )
        print(
            "See details on requesting a quota increase at: "
            "https://cloud.google.com/compute/quotas#requesting_additional_quota"
        )
        print("Required quota type: %s" % quota_code)
        print("This check can be skipped via `det deploy --no-preflight-checks ...`")
        sys.exit(1)
