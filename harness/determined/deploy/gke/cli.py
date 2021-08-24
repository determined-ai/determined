import argparse
import json
import subprocess
import sys
from multiprocessing.sharedctypes import Value
from pathlib import Path
from typing import Callable, Dict, Union

from termcolor import cprint

import determined
from determined.common import yaml
from determined.common.declarative_argparse import Arg, ArgGroup, Cmd, Group
from determined.common.util import safe_load_yaml_with_exceptions
from determined.deploy.gke.constants import defaults


def make_spec(task_container_defaults: dict, key: str) -> dict:
    pod_spec = task_container_defaults.get(key)
    if not pod_spec:
        pod_spec = {"apiVersion": "v1", "kind": "Pod"}
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
        raise argparse.ArgumentTypeError(
            "Accelerator must be one of {}".format(valid_accelerator_types)
        )


def validate_location(location: str, isZone: bool = True) -> None:
    try:
        cmd = ["gcloud", "compute"]
        if isZone:
            cmd += ["zones"]
        else:
            cmd += ["regions"]
        cmd += ["describe", location]
        subprocess.check_call(cmd, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    except:
        raise argparse.ArgumentTypeError(
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
    except:
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
    except:
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
    if args.multi_node_pool:
        if args.cpu_node_pool_name is None:
            raise ValueError(
                "--cpu-node-pool-name must be specified if using multiple node pools "
                + "(--multi-node-pool flag is set).",
            )
    if args.gpu_coscheduler and args.preemption:
        raise ValueError(
            "--gpu-coscheduler and --preemptive-scheduler are mutually exclusive and cannot both be"
            " specified",
        )
    if args.cpu_coscheduler and not args.gpu_coscheduler:
        raise ValueError(
            "To enable coscheduling on the CPU Node Pool, coscheduling must be enabled on the GPU"
            " Node Pool. --cpu-coscheduler cannot be specified if --gpu-coscheduler isn't as well."
        )
    if args.cpu_coscheduler and not args.multi_node_pool:
        raise ValueError(
            "To enable coscheduling on the CPU Node Pool, multiple node pools must be used."
            + " --cpu-coscheduler cannot be specified if --multi-node-pool isn't as well."
        )


def create_cluster(args: argparse.Namespace) -> None:
    region = "-".join(args.zone.split("-")[:-1])
    cmd = [
        "gcloud",
        "container",
        "clusters",
        "create",
        args.cluster_name,
        "--region",
        region,
        "--node-locations",
        args.zone,
        "--num-nodes=1",
        "--machine-type={}".format(args.master_machine_type),
    ]
    subprocess.check_call(cmd, stdout=subprocess.DEVNULL)
    create_nodepools(region, args)
    if not args.no_make_bucket:
        cmd = ["gsutil", "mb", "gs://{}".format(args.gcs_bucket)]
        subprocess.check_call(cmd, stdout=subprocess.DEVNULL)


def create_nodepools(region: str, args: argparse.Namespace) -> None:
    cmd = [
        "gcloud",
        "container",
        "node-pools",
        "create",
        args.gpu_node_pool_name,
        "--cluster",
        args.cluster_name,
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
    if args.multi_node_pool:
        cmd += ["--node-taints=gpuAvailable=True:NoSchedule"]
    subprocess.check_call(cmd, stdout=subprocess.DEVNULL)
    cmd = [
        "kubectl",
        "apply",
        "-f",
        defaults.K8S_NVIDIA_DAEMON,
    ]
    subprocess.check_call(cmd, stdout=subprocess.DEVNULL)
    if args.multi_node_pool:
        cmd = [
            "gcloud",
            "container",
            "node-pools",
            "create",
            args.cpu_node_pool_name,
            "--cluster",
            args.cluster_name,
            "--zone",
            region,
        ]
        if args.cpu_coscheduler:
            cmd += ["--num-nodes={}".format(args.max_cpu_nodes)]
        else:
            cmd += [
                "--num-nodes=0",
                "--enable-autoscaling",
                "--min-nodes=0",
                "--max-nodes={}".format(args.max_cpu_nodes),
            ]
        cmd += [
            "--machine-type={}".format(args.agent_machine_type),
            "--scopes=storage-full,cloud-platform",
            "--node-labels=accelerator_type=cpu",
            "--node-taints=gpuAvailable=False:NoSchedule",
        ]
        subprocess.check_call(cmd)


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
    checkpointStorage["bucket"] = args.gcs_bucket
    helm_values["checkpointStorage"] = checkpointStorage
    helm_values["maxSlotsPerPod"] = args.gpus_per_node
    if args.multi_node_pool:
        gpu_pod_spec = make_spec(helm_values["taskContainerDefaults"], "gpuPodSpec")
        gpu_pod_spec["spec"]["tolerations"] = [
            {
                "key": "gpuAvailable",
                "operator": "Equal",
                "value": "True",
                "effect": "NoSchedule",
            }
        ]
        gpu_pod_spec["spec"]["nodeSelector"] = {"accelerator_type": "gpu"}
        helm_values["taskContainerDefaults"]["gpuPodSpec"] = gpu_pod_spec
        cpu_pod_spec = make_spec(helm_values["taskContainerDefaults"], "cpuPodSpec")
        cpu_pod_spec["spec"]["tolerations"] = [
            {
                "key": "gpuAvailable",
                "operator": "Equal",
                "value": "False",
                "effect": "NoSchedule",
            }
        ]
        cpu_pod_spec["spec"]["nodeSelector"] = {"accelerator_type": "cpu"}
        if args.cpu_coscheduler:
            cpu_pod_spec["spec"]["schedulerName"] = "coscheduler"
        helm_values["taskContainerDefaults"]["cpuPodSpec"] = cpu_pod_spec
    with (helm_dir / "values.yaml").open("w") as f:
        yaml.round_trip_dump(helm_values, f)


def handle_up(args: argparse.Namespace) -> None:
    try:
        validate_args(args)
    except Exception as e:
        exc_str = "Argument Error: {}".format(e)
        cprint(exc_str, "red")
        cprint("Failed to create gke cluster", "red")
        sys.exit(1)
    if args.gpu_coscheduler or args.preemption:
        cprint(
            (
                "Autoscaling is not supported with the lightweight coscheduling plugin or with the"
                + " preemptive priority-based scheduler, and so a GPU node pool with {} nodes will"
                + " be statically allocated. This can be changed by using the default Kubernetes"
                + "  scheduler, or specifying a different value for --max-gpu-nodes."
            ).format(args.max_gpu_nodes),
            "yellow",
        )
    if args.cpu_coscheduler:
        cprint(
            (
                "Autoscaling is not supported with the lightweight coscheduling plugin and so a CPU"
                + " node pool with {} nodes will be statically allocated. This can be changed by"
                + "  using the default Kubernetes scheduler, or specifying a different value for"
                + " --max-cpu-nodes."
            ).format(args.max_cpu_nodes),
            "yellow",
        )
    create_cluster(args)
    configure_helm(args)
    if args.cpu_coscheduler:
        cprint(
            (
                "The Priority-Based Gang Scheduler has been enabled for the CPU Agent Pool. To use"
                + " this feature, please specify the following in your experiment config for"
                + " commands or CPU Experiments:"
            ),
            "yellow",
        )
        exp_config = {
            "environment": {
                "podspec": {
                    "metadata": {
                        "labels": {
                            "pod-group.scheduling.sigs.k8s.io/name": "<unique task name>",
                            "pod-group.scheduling.sigs.k8s.io/min-available": "<# of Nodes required>",
                        }
                    }
                }
            }
        }
        cprint(yaml.dump(exp_config), "blue")
        cprint("Replacing the text in the angle brackets (<>).", "yellow")
    cmd = ["helm", "install", "determined-gke", args.helm_dir]
    subprocess.check_call(cmd)


def handle_down(args: argparse.Namespace) -> None:
    validate_location(args.region, isZone=False)
    cprint(
        (
            "Setting kubectl config to cluster {}. Please make sure to run\n`kubectl config "
            + "set-cluster <other_cluster_name>`\nto interact with other deployed clusters."
        ).format(args.cluster_name),
        "yellow",
    )
    cmd = ["kubectl", "config", "set-cluster", args.cluster_name]
    subprocess.check_call(cmd)
    cmd = ["helm", "uninstall", "determined-gke"]
    subprocess.check_call(cmd)
    cmd = [
        "gcloud",
        "container",
        "clusters",
        "delete",
        args.cluster_name,
        "--region",
        args.region,
        "--quiet",
    ]
    subprocess.check_call(cmd)
    print("Succesfully deleted GKE Cluster {}".format(args.cluster_name))


args_description = Cmd(
    "gke",
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
                            "--cluster-name",
                            type=str,
                            default=None,
                            required=True,
                            help="a unique name for the gke cluster",
                        ),
                        Arg(
                            "--gpu-node-pool-name",
                            "--gpu-name",
                            type=str,
                            default=None,
                            required=True,
                            help="a unique name for the GPU node pool",
                        ),
                        Arg(
                            "--gcs-bucket",
                            type=str,
                            default=None,
                            required=True,
                            help="a unique name for the GCS bucket that will store your"
                            " checkpoints",
                        ),
                    ],
                ),
                ArgGroup(
                    "optional named arguments",
                    None,
                    [
                        Arg(
                            "--gpu-type",
                            type=str,
                            default=defaults.GPU_TYPE,
                            required=False,
                            help="accelerator type to use for agents",
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
                            "--no-make-bucket",
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
                            "--multi-node-pool",
                            required=False,
                            help="flag that indicates multiple node pools should be used - one"
                            " for CPU only tasks and one for GPU tasks",
                            action="store_true",
                        ),
                        Arg(
                            "--cpu-coscheduler",
                            required=False,
                            help="Enables the lightweight coscheduling plugin for Kubernetes that"
                            " provides priority-based gang scheduling for the CPU Agent Nodepool."
                            "If this argument is set, cluster autoscaling is disabled, and"
                            " --max-cpu-nodes nodes are statically allocated for the CPU Agent Node"
                            " pool at creation time.",
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
                            "--cluster-name",
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
                    ],
                ),
            ],
        ),
    ],
)
