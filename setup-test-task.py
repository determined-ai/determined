import argparse
import re
import subprocess
import time

import termcolor

parser = argparse.ArgumentParser(description="Setup a test task")
# optionally get a task id
parser.add_argument("--task-id", help="Task ID to use")
args = parser.parse_args()

task_id = args.task_id
if not task_id:
    task_id = (
        subprocess.check_output(
            ["det", "notebook", "start", "--config", 'resources={"slots":0}', "-d", "--no-browser"]
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
            [
                "kubectl",
                "get",
                "pods",
                "-l",
                f"determined.ai/task_id={task_id}",
                "-o",
                "jsonpath={.items[0].metadata.name}",
            ],
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
        subprocess.check_output(
            ["kubectl", "get", "pod", pod_name, "-o", "jsonpath={.status.podIP}"]
        )
        .decode("utf-8")
        .strip()
    )
    time.sleep(0.5)

pod_port = (
    subprocess.check_output(
        [
            "kubectl",
            "get",
            "pod",
            pod_name,
            "-o",
            "jsonpath={.spec.containers[0].ports[0].containerPort}",
        ]
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
            ["kubectl", "get", "pod", pod_name, "-o", "jsonpath={.status.phase}"]
        )
        .decode("utf-8")
        .strip()
    )
    if pod_status == "Running":
        break
    time.sleep(0.5)

local_port = 47777
forward_cmd = ["kubectl", "port-forward", pod_name, f"{local_port}:{pod_port}"]
print(f"Port forward: {' '.join(forward_cmd)}")
pforward = subprocess.Popen(forward_cmd, stdout=subprocess.PIPE, text=True)
while True:
    line = pforward.stdout.readline()
    if "forwarding from" in line.lower():
        break

interests = [
    ["curl", "-s", f"http://{pod_ip}:{pod_port}/", "-m", "1"],
    ["curl", "-s", f"http://{pod_ip}:{pod_port}/proxy/{task_id}/", "-m", "1"],
    ["curl", "-s", f"http://localhost:{local_port}/", "-m", "1"],
    ["curl", "-s", f"http://localhost:{local_port}/proxy/{task_id}/", "-m", "1"],
    ["curl", "-s", f"http://localhost:{local_port}/lab", "-m", "1"],
    ["curl", "-s", f"http://localhost:{local_port}/proxy/{task_id}/lab", "-m", "1"],
    ["det", "dev", "curl", f"/proxy/{task_id}/", "-s", "-m", "1"],
]
for cmd in interests:
    print(termcolor.colored(f"Running: {' '.join(cmd)}", "yellow"))
    finished_proc = subprocess.run(cmd, stdout=subprocess.PIPE)
    got_html = "html" in finished_proc.stdout.decode("utf-8")
    return_code = finished_proc.returncode
    print(f"Return code: {return_code}, Got HTML: {got_html}")

pforward.terminate()
