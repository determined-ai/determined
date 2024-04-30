import argparse
import json
import re
import subprocess
import subprocess as sp
import time
from typing import List

import termcolor

import determined as det

parser = argparse.ArgumentParser(description="Setup a test task")
parser.add_argument("--task-id", help="Task ID to use")
parser.add_argument("--kctl-context", type=str, default="kind-kind", help="Kubectl context to use")
parser.add_argument("--rp", type=str, default="default", help="resource pool to use")
args = parser.parse_args()


def kctl(ctl_args: List[str]) -> List[str]:
    return ["kubectl", "--context", args.kctl_context] + ctl_args


def call_det_api(args: List[str]) -> str:
    out = subprocess.run(
        ["det", "-u", "admin", "dev", "bindings", "call", "-y"] + args,
        check=True,
        text=True,
        stdout=subprocess.PIPE,
    )
    return out.stdout


def get_det_k8s_contexts() -> List[str]:
    # get configured k8s clusters
    mconfig = json.loads(call_det_api(["get_GetMasterConfig"]))["config"]
    rm = mconfig["resource_manager"]
    assert rm["type"] == "kubernetes"
    default_context_name = rm["name"]
    configured_contexts = [default_context_name]
    addl = mconfig.get("additional_resource_managers") or []
    for addl_rm in addl:
        addl_rm = addl_rm["resource_manager"]
        if addl_rm["type"] == "kubernetes":
            configured_contexts.append(addl_rm["name"])
    return configured_contexts


if args.kctl_context:
    contexts = get_det_k8s_contexts()
    assert (
        args.kctl_context in contexts
    ), f"Kubectl context {args.kctl_context} not found in {contexts}"

task_id = args.task_id
if not task_id:
    req_config = {"slots": 0, "resource_pool": args.rp}
    task_id = (
        subprocess.check_output(
            [
                "det",
                "notebook",
                "start",
                "--config",
                f"resources={json.dumps(req_config)}",
                "-d",
                "--no-browser",
            ]
        )
        .decode("utf-8")
        .strip()
    )
print(f"Task ID: {task_id}")

# # log_proc = subprocess.Popen(["det", "task", "logs", task_id, "-f"], stdout=subprocess.PIPE)
# time.sleep(5)
# log_proc = subprocess.Popen(["det", "task", "logs", task_id], stdout=subprocess.PIPE, text=True)
# port_re = re.compile(r"http://localhost:(\d+)/proxy/{task_id}/")
# container_port = None
# while True:
#     line = log_proc.stdout.readline().strip()
#     match = port_re.search(line)
#     if match:
#         container_port = match.group(1)
#         break
# log_proc.terminate()
# print(f"Container Port: {container_port}")
# exit(0)

# find the pod with this task id selector
pod_name = None
while not pod_name:
    print(".", end="", flush=True)
    try:
        log_proc = subprocess.run(
            kctl(
                [
                    "get",
                    "pods",
                    "-l",
                    f"determined.ai/task_id={task_id}",
                    "-o",
                    "jsonpath={.items[0].metadata.name}",
                ]
            ),
            stderr=subprocess.DEVNULL,
            stdout=subprocess.PIPE,
        )
        log_proc.check_returncode()
        if log_proc.stdout:
            pod_name = log_proc.stdout.decode("utf-8").strip()
    except subprocess.CalledProcessError:
        pass
    time.sleep(0.5)
print(f"Pod name: {pod_name}")

pod_ip = None
while not pod_ip:
    print(".", end="", flush=True)
    pod_ip = (
        subprocess.check_output(kctl(["get", "pod", pod_name, "-o", "jsonpath={.status.podIP}"]))
        .decode("utf-8")
        .strip()
    )
    time.sleep(0.5)

pod_port = (
    subprocess.check_output(
        kctl(
            [
                "get",
                "pod",
                pod_name,
                "-o",
                "jsonpath={.spec.containers[0].ports[0].containerPort}",
            ]
        )
    )
    .decode("utf-8")
    .strip()
)

print(f"Pod IP: {pod_ip}")
print(f"Pod Port: {pod_port}")

# wait for the pod to be running
while True:
    print(".", end="", flush=True)
    pod_status = (
        subprocess.check_output(
            kctl(["get", "pod", pod_name, "-o", "jsonpath={.status.phase}"]),
            stderr=subprocess.DEVNULL,
        )
        .decode("utf-8")
        .strip()
    )
    if pod_status == "Running":
        break
    time.sleep(0.5)

local_port = 47777
forward_cmd = kctl(["port-forward", pod_name, f"{local_port}:{pod_port}"])
print(f"Port forward: {' '.join(forward_cmd)}")
time.sleep(3)  # pod ready is not enough
pforward = subprocess.Popen(forward_cmd, stdout=subprocess.PIPE, text=True)
while True:
    line = pforward.stdout.readline()
    if "forwarding from" in line.lower():
        break

# wait for the local port to be ready. nc -vz
while True:
    print(".", end="", flush=True)
    nc_proc = subprocess.run(["nc", "-vz", "localhost", str(local_port)], stderr=subprocess.PIPE)
    if nc_proc.returncode == 0:
        break
    time.sleep(0.5)

interests = [
    ["curl", "-s", f"http://{pod_ip}:{pod_port}/", "-m", "1"],
    ["curl", "-s", f"http://{pod_ip}:{pod_port}/proxy/{task_id}/", "-m", "1"],
    ["curl", "-s", f"http://localhost:{local_port}/", "-m", "1"],
    ["curl", "-s", f"http://localhost:{local_port}/proxy/{task_id}/", "-m", "1"],
    ["curl", "-s", f"http://localhost:{local_port}/lab", "-m", "1"],
    ["curl", "-s", f"http://localhost:{local_port}/lab", "-m", "1", "|", "grep", "'not exist'"],
    ["curl", "-s", f"http://localhost:{local_port}/proxy/{task_id}/lab", "-m", "1"],
    [
        "curl",
        "-s",
        f"http://localhost:{local_port}/proxy/{task_id}/lab",
        "-m",
        "1",
        "|",
        "grep",
        "'not exist'",
    ],
    ["det", "dev", "curl", f"/proxy/{task_id}/", "-s", "-m", "1"],
]
for cmd in interests:
    print(termcolor.colored(f"Running: {' '.join(cmd)}", "yellow"))
    finished_proc = subprocess.run(" ".join(cmd), stdout=subprocess.PIPE, shell=True)
    got_html = "html" in finished_proc.stdout.decode("utf-8")
    return_code = finished_proc.returncode
    print(f"Return code: {return_code}, Got HTML: {got_html}")

input("Press Enter to cleanup")
pforward.terminate()
