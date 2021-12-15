import argparse
import json
import subprocess
import sys
from pathlib import Path
from typing import Any, Dict, Optional, Union, cast

from termcolor import cprint

import determined
from determined.common import yaml
from determined.common.declarative_argparse import Arg, ArgGroup, Cmd
from determined.common.util import safe_load_yaml_with_exceptions
from determined.deploy.gke.constants import defaults


def make_spec(task_container_defaults: Dict[str, Any], key: str) -> Dict[str, Union[Dict, str]]:
    pod_spec = task_container_defaults.get(key)  # type: Optional[Dict]
    if not pod_spec:
        pod_spec = {"apiVersion": "v1", "kind": "Pod"}
    pod_spec = cast(Dict[str, Union[Dict, str]], pod_spec)
    if not pod_spec.get("spec"):
        pod_spec["spec"] = {}
    return pod_spec


def validate_accelerator_type(s: str) -> None:
    json_value = subprocess.check_output(
        ["gcloud", "compute", "accelerator-types", "list", "--format=json(name)"]
    )
    json_names = json.loads(json_value)
    valid_accelerator_types = {accelerator["name"] for accelerator in json_names}

    if s not in valid_accelerator_types:
        raise ValueError("Accelerator must be one of {}".format(valid_accelerator_types))


def validate_location(location: str, isZone: bool = True) -> None:
    try:
        cmd = ["gcloud", "compute"]
        if isZone:
            cmd += ["zones"]
        else:
            cmd += ["regions"]
        cmd += ["describe", location]
        subprocess.check_call(cmd, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    except subprocess.CalledProcessError:
        raise ValueError(
            "The specified {} {} was not found".format("zone" if isZone else "region", location)
        )


def validate_machine_type(machine_type: str, zone: str) -> None:
    try:
        subprocess.check_call(
            [
                "gcloud",
                "compute",
                "machine-types",
                "describe",
                machine_type,
                "--zone",
                zone,
            ],
            stdout=subprocess.DEVNULL,
        )
    except subprocess.CalledProcessError:
        raise ValueError(
            "The specified machine type {} was not found in the specified zone {}.".format(
                machine_type, zone
            ),
        )


def validate_accelerator_zone(args: argparse.Namespace, zone: str) -> None:
    try:
        subprocess.check_call(
            [
                "gcloud",
                "compute",
                "accelerator-types",
                "describe",
                args.gpu_type,
                "--zone",
                zone,
            ],
            stdout=subprocess.DEVNULL,
        )
    except subprocess.CalledProcessError:
        raise ValueError(
            "The specified accelerator type {} was not found in the specified zone {}.".format(
                args.gpu_type, zone
            ),
        )


def validate_args(args: argparse.Namespace) -> None:
    validate_location(args.zone, isZone=True)
    validate_accelerator_type(args.gpu_type)

    if args.master_machine_type != "n1-standard-16":
        validate_machine_type(args.master_machine_type, args.zone)

    if args.agent_machine_type != "n1-standard-32":
        validate_machine_type(args.agent_machine_type, args.zone)

    validate_accelerator_zone(args, args.zone)

    if args.gpu_coscheduler and args.preemption:
        raise ValueError(
            "--gpu-coscheduler and --preemptive-scheduler are mutually exclusive and cannot both be"
            " specified",
        )

    if args.gcs_bucket_name is None:
        args.gcs_bucket_name = args.cluster_id + "-checkpoints"
    if args.agent_node_pool_name is None:
        args.agent_node_pool_name = args.cluster_id + "-gpu-pool"
    if args.cpu_node_pool_name is None:
        args.cpu_node_pool_name = args.cluster_id + "-cpu-pool"
    if not Path(args.helm_dir).exists():
        raise ValueError("Please specify valid --helm-dir")


def create_cluster(args: argparse.Namespace) -> None:
    region = args.zone.rsplit("-", 1)[0]
    cmd = [
        "gcloud",
        "container",
        "clusters",
        "create",
        args.cluster_id,
        "--region",
        region,
        "--node-locations",
        args.zone,
        "--num-nodes=1",
        "--machine-type={}".format(args.master_machine_type),
    ]
    subprocess.check_call(cmd, stdout=subprocess.DEVNULL)

    create_nodepools(region, args)

    if not args.no_managed_bucket:
        cmd = ["gsutil", "mb", "gs://{}".format(args.gcs_bucket_name)]
        subprocess.check_call(cmd, stdout=subprocess.DEVNULL)


def create_gpu_nodepool(region: str, args: argparse.Namespace) -> None:
    cmd = [
        "gcloud",
        "container",
        "node-pools",
        "create",
        args.agent_node_pool_name,
        "--cluster",
        args.cluster_id,
        "--accelerator",
        "type={},count={}".format(args.gpu_type, args.gpus_per_node),
        "--zone",
        region,
    ]
    if args.gpu_coscheduler or args.preemption:
        cmd += ["--num-nodes={}".format(args.max_gpu_nodes)]
    else:
        cmd += [
            "--num-nodes=0",
            "--enable-autoscaling",
            "--min-nodes=0",
            "--max-nodes={}".format(args.max_gpu_nodes),
        ]
    cmd += [
        "--machine-type={}".format(args.agent_machine_type),
        "--scopes=storage-full,cloud-platform",
        "--node-labels=accelerator_type=gpu",
    ]
    if args.multiple_node_pools:
        cmd += ["--node-taints=gpuAvailable=True:NoSchedule"]
    subprocess.check_call(cmd, stdout=subprocess.DEVNULL)


def create_cpu_nodepool(region: str, args: argparse.Namespace) -> None:
    cmd = [
        "gcloud",
        "container",
        "node-pools",
        "create",
        args.cpu_node_pool_name if args.multiple_node_pools else args.agent_node_pool_name,
        "--cluster",
        args.cluster_id,
        "--zone",
        region,
        "--num-nodes=0",
        "--enable-autoscaling",
        "--min-nodes=0",
        "--max-nodes={}".format(args.max_cpu_nodes),
        "--machine-type={}".format(args.agent_machine_type),
        "--scopes=storage-full,cloud-platform",
    ]

    if args.multiple_node_pools:
        cmd += [
            "--node-labels=accelerator_type=cpu",
            "--node-taints=gpuAvailable=False:NoSchedule",
        ]
    subprocess.check_call(cmd)


def create_nodepools(region: str, args: argparse.Namespace) -> None:
    create_gpu_nodepool(region, args)
    if args.multiple_node_pools or args.cpu_only:
        create_cpu_nodepool(region, args)

    cmd = [
        "kubectl",
        "apply",
        "-f",
        defaults.K8S_NVIDIA_DAEMON,
    ]
    subprocess.check_call(cmd, stdout=subprocess.DEVNULL)


def configure_helm(args: argparse.Namespace) -> None:
    helm_dir = Path(args.helm_dir)
    with (helm_dir / "Chart.yaml").open() as f:
        helm_chart = safe_load_yaml_with_exceptions(f)
    if args.det_version:
        helm_chart["appVersion"] = args.det_version
    elif "dev" in helm_chart["appVersion"]:
        # Preserve user overridden appVersion in helm chart unless it includes dev in the version.
        helm_chart["appVersion"] = determined.__version__
    if args.gpu_coscheduler:
        helm_chart["defaultScheduler"] = "coscheduler"
    elif args.preemption:
        helm_chart["defaultScheduler"] = "preemption"
    with (helm_dir / "Chart.yaml").open("w") as f:
        yaml.round_trip_dump(helm_chart, f)
    with (helm_dir / "values.yaml").open() as f:
        helm_values = safe_load_yaml_with_exceptions(f)
    checkpointStorage = {}
    checkpointStorage["saveExperimentBest"] = helm_values["checkpointStorage"].get(
        "saveExperimentBest", 0
    )
    checkpointStorage["saveTrialBest"] = helm_values["checkpointStorage"].get("saveTrialBest", 1)
    checkpointStorage["saveTrialLatest"] = helm_values["checkpointStorage"].get(
        "saveTrialLatest", 1
    )
    checkpointStorage["type"] = "gcs"
    checkpointStorage["bucket"] = args.gcs_bucket_name
    helm_values["checkpointStorage"] = checkpointStorage
    helm_values["maxSlotsPerPod"] = args.gpus_per_node

    if args.multiple_node_pools:
        gpu_pod_spec = make_spec(helm_values["taskContainerDefaults"], "gpuPodSpec")
        gpu_spec = cast(Dict, gpu_pod_spec["spec"])
        gpu_spec["tolerations"] = [
            {
                "key": "gpuAvailable",
                "operator": "Equal",
                "value": "True",
                "effect": "NoSchedule",
            }
        ]
        gpu_spec["nodeSelector"] = {"accelerator_type": "gpu"}

        helm_values["taskContainerDefaults"]["gpuPodSpec"] = gpu_pod_spec

        cpu_pod_spec = make_spec(helm_values["taskContainerDefaults"], "cpuPodSpec")
        cpu_spec = cast(Dict, cpu_pod_spec["spec"])
        cpu_spec["tolerations"] = [
            {
                "key": "gpuAvailable",
                "operator": "Equal",
                "value": "False",
                "effect": "NoSchedule",
            }
        ]
        cpu_spec["nodeSelector"] = {"accelerator_type": "cpu"}

        helm_values["taskContainerDefaults"]["cpuPodSpec"] = cpu_pod_spec

    with (helm_dir / "values.yaml").open("w") as f:
        yaml.round_trip_dump(helm_values, f)


def handle_up(args: argparse.Namespace) -> None:
    try:
        validate_args(args)
    except ValueError as e:
        exc_str = "Argument Error: {}".format(e)
        cprint(exc_str, "red")
        cprint("Failed to create gke cluster", "red")
        sys.exit(1)
    if args.gpu_coscheduler or args.preemption:
        cprint(
            (
                "Autoscaling is not supported with the lightweight coscheduling plugin or with the"
                " preemptive priority-based scheduler, and so a GPU node pool with {} nodes will"
                " be statically allocated. This can be changed by using the default Kubernetes"
                "  scheduler, or specifying a different value for --max-gpu-nodes."
            ).format(args.max_gpu_nodes),
            "yellow",
        )
    create_cluster(args)
    configure_helm(args)
    cmd = ["helm", "install", "determined-gke", args.helm_dir]
    subprocess.check_call(cmd)


def handle_down(args: argparse.Namespace) -> None:
    validate_location(args.region, isZone=False)
    cprint(
        (
            "Setting kubectl config to cluster {}. Please make sure to run\n`kubectl config "
            "set-cluster <other_cluster_name>`\nto interact with other deployed clusters."
        ).format(args.cluster_id),
        "yellow",
    )
    cmd = ["kubectl", "config", "set-cluster", args.cluster_id]
    subprocess.check_call(cmd)
    cmd = [
        "gcloud",
        "container",
        "clusters",
        "delete",
        args.cluster_id,
        "--region",
        args.region,
        "--quiet",
    ]
    subprocess.check_call(cmd)

    if not args.no_managed_bucket:
        if args.gcs_bucket_name is None:
            args.gcs_bucket_name = args.cluster_id + "-checkpoints"

        cmd = ["gsutil", "rm", "-r", "gs://{}".format(args.gcs_bucket_name)]
        subprocess.check_call(cmd, stdout=subprocess.DEVNULL)

    print("Succesfully deleted GKE Cluster {}".format(args.cluster_id))


args_description = Cmd(
    "gke-experimental",
    None,
    "GKE help",
    [
        Cmd(
            "up",
            handle_up,
            "create gke cluster",
            [
                ArgGroup(
                    "required named arguments",
                    None,
                    [
                        Arg(
                            "--cluster-id",
                            type=str,
                            default=None,
                            required=True,
                            help="a unique name for the gke cluster",
                        ),
                    ],
                ),
                ArgGroup(
                    "optional named arguments",
                    None,
                    [
                        Arg(
                            "--agent-node-pool-name",
                            "--gpu-node-pool-name",
                            type=str,
                            default=None,
                            help="a unique name for the GPU node pool",
                        ),
                        Arg(
                            "--gcs-bucket-name",
                            type=str,
                            default=None,
                            help="a unique name for the GCS bucket that will store your"
                            " checkpoints",
                        ),
                        Arg(
                            "--gpu-type",
                            type=str,
                            default=defaults.GPU_TYPE,
                            required=False,
                            help="accelerator type to use for agents",
                        ),
                        Arg(
                            "--cpu-only",
                            required=False,
                            help="Flag to create a CPU Only Determined Instance.",
                            action="store_true",
                        ),
                        Arg(
                            "--gpus-per-node",
                            type=int,
                            default=defaults.GPUS_PER_NODE,
                            required=False,
                            help="number of GPUs per node",
                        ),
                        Arg(
                            "--helm-dir",
                            type=str,
                            default="helm/charts/determined",
                            required=False,
                            help="directory containing Helm Chart, values.yaml and templates.",
                        ),
                        Arg(
                            "--det-version",
                            type=str,
                            default=None,
                            help=argparse.SUPPRESS,
                        ),
                        Arg(
                            "--no-managed-bucket",
                            required=False,
                            help="flag that indicates GCS checkpointing bucket already exists",
                            action="store_true",
                        ),
                        Arg(
                            "--zone",
                            type=str,
                            default=defaults.ZONE,
                            help="zone to create cluster in",
                        ),
                        Arg(
                            "--master-machine-type",
                            type=str,
                            default=defaults.MASTER_MACHINE_TYPE,
                            help="machine type to use for master node group",
                        ),
                        Arg(
                            "--agent-machine-type",
                            "--machine-type",
                            type=str,
                            default=defaults.AGENT_MACHINE_TYPE,
                            help="machine type to use for agent node group",
                        ),
                        Arg(
                            "--max-gpu-nodes",
                            "--max-nodes",
                            type=int,
                            default=defaults.MAX_GPU_NODES,
                            help="maximum number of nodes for the GPU node group",
                        ),
                        Arg(
                            "--max-cpu-nodes",
                            type=int,
                            default=defaults.MAX_CPU_NODES,
                            help="maximum number of nodes for the CPU node group",
                        ),
                        Arg(
                            "--cpu-node-pool-name",
                            type=str,
                            default=None,
                            help="a unique name for the GPU node pool",
                        ),
                        Arg(
                            "--multiple-node-pools",
                            required=False,
                            help="flag that indicates multiple node pools should be used - one"
                            " for CPU only tasks and one for GPU tasks",
                            action="store_true",
                        ),
                        Arg(
                            "--gpu-coscheduler",
                            "--coscheduler",
                            required=False,
                            help="Enables the lightweight coscheduling plugin for Kubernetes that"
                            " provides priority-based gang scheduling for the GPU Agent Nodepool."
                            "If this argument is set, cluster autoscaling is disabled, and"
                            " --max-gpu-nodes nodes are statically allocated for the GPU Agent Node"
                            " pool at creation time.",
                            action="store_true",
                        ),
                        Arg(
                            "--preemption",
                            "--preemptive-scheduler",
                            required=False,
                            help="Enables the priority-based scheduler with preemption on the GPU"
                            " Agent Nodepool. If this argument is set, cluster autoscaling is"
                            " disabled, and --max-gpu-nodes nodes are statically allocated for the "
                            " GPU Agent Node pool at creation time.",
                            action="store_true",
                        ),
                    ],
                ),
            ],
        ),
        Cmd(
            "down",
            handle_down,
            "delete gke cluster",
            [
                ArgGroup(
                    "required named arguments",
                    None,
                    [
                        Arg(
                            "--cluster-id",
                            type=str,
                            default=None,
                            required=True,
                            help="the gke cluster to delete",
                        )
                    ],
                ),
                ArgGroup(
                    "optional named arguments",
                    None,
                    [
                        Arg(
                            "--region",
                            type=str,
                            default="us-west1",
                            help="region containing cluster to delete",
                        ),
                        Arg(
                            "--no-managed-bucket",
                            required=False,
                            help="GCS checkpointing bucket is managed externally",
                            action="store_true",
                        ),
                        Arg(
                            "--gcs-bucket-name",
                            type=str,
                            default=None,
                            help="a unique name for the GCS bucket that will store your"
                            " checkpoints",
                        ),
                    ],
                ),
            ],
        ),
    ],
)
