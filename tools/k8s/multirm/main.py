#!/usr/bin/env python

import json
import re
import subprocess as sp
import threading
import time
from typing import Any, Dict, List, NamedTuple, Optional, Tuple, Union

import fire
import termcolor
import websocket

DEFAULT_SERVICE_PORT = 4000
DEFAULT_LOCAL_PORT = 47777

TARGET_CONTEXT = sp.check_output(["kubectl", "config", "current-context"], text=True).strip()

PodConfig = NamedTuple("PodConfig", [("name", str), ("image", str), ("port", int)])
PodDetails = NamedTuple("PodDetails", [("name", str), ("ip", str), ("port", int)])
NamedAddr = NamedTuple("NamedAddr", [("name", str), ("host", str), ("port", int)])


def get_valid_contexts() -> List[str]:
    contexts = run_command("kubectl config get-contexts -o name", capture=True).stdout
    return contexts.strip().split("\n")


def check_context(k8s_context: str) -> None:
    contexts = get_valid_contexts()
    assert (
        k8s_context in contexts
    ), f"Kubectl context {TARGET_CONTEXT} not found in det contexts {contexts}"


def run_command(command: Union[str, List[str]], capture: bool = False) -> sp.CompletedProcess:
    """execute shell command."""
    if isinstance(command, list):
        command = " ".join(command)
    result = sp.run(command, shell=True, text=True, capture_output=capture)
    if result.returncode != 0:
        print(f"Error: {result.stderr}")
    return result


def kctl(ctl_args: List[str]) -> List[str]:
    return ["kubectl", "--context", TARGET_CONTEXT] + ctl_args


def call_det_api(args: List[str]) -> str:
    out = sp.run(
        ["det", "-u", "admin", "dev", "bindings", "call"] + args,
        check=True,
        text=True,
        stdout=sp.PIPE,
    )
    return out.stdout


def is_jl_page(html: str) -> bool:
    """check if a page is a JupyterLab page."""
    if "html" not in html:
        return False
    if "<title>JupyterLab</ti" not in html:
        return False
    if "not exist" in html:
        return False
    return True


def cleanup_cluster():
    """
    kill all det pods and services
    """
    input(f"Press Enter to cleanup {TARGET_CONTEXT}")
    run_command(kctl(["delete", "pods", "--all"]))
    run_command(kctl(["delete", "services", "--all"]))
    run_command(kctl(["delete", "deployments", "--all"]))


def can_connect_ws(url: str) -> bool:
    connection_stable: bool = False

    def on_open(ws: websocket.WebSocketApp) -> None:
        nonlocal connection_stable
        print("ws: Connection opened")
        connection_stable = True
        threading.Timer(2.0, ws.close).start()

    def on_close(ws: websocket.WebSocketApp) -> None:
        print("ws: Connection closed")

    def on_error(ws: websocket.WebSocketApp, error: Any) -> None:
        print("ws: Error occurred:", error)

    det_token = sp.run(
        "det dev auth-token", shell=True, text=True, capture_output=True
    ).stdout.strip()
    ws: websocket.WebSocketApp = websocket.WebSocketApp(
        url,
        on_open=on_open,
        on_close=on_close,
        on_error=on_error,
        header=["Authorization: Bearer " + det_token],
    )
    ws.run_forever()
    print("ws url", url)
    print("connection_stable", connection_stable)

    return connection_stable


def run_notebook_tests(task_id) -> None:
    checks = [["det", "dev", "curl", f"/proxy/{task_id}/lab", "-s", "-m", "1"]]
    for cmd in checks:
        print(termcolor.colored(f"Running: {' '.join(cmd)}", "yellow"))
        proc = sp.run(cmd, stdout=sp.PIPE, text=True)
        output = proc.stdout
        got_html = "html" in output
        return_code = proc.returncode
        print(f"Return code: {return_code}, Got HTML: {got_html}")
        print(f"is notebook page", is_jl_page(output))

    url = f"ws://localhost:8080/proxy/{task_id}/api/events/subscribe"
    if not can_connect_ws(url):
        print(f"Failed to connect to {url}")


def run_and_capture_long_running(cmd: List[str], wait=3) -> Tuple[str, str]:
    print(f"Running: {' '.join(cmd)}")
    proc = sp.Popen(cmd, stdout=sp.PIPE, stderr=sp.PIPE, text=True)
    time.sleep(wait)
    if proc.poll() is not None:
        print("Tunnel process exited unexpectedly")
        return "", ""
    proc.terminate()
    assert proc.stdout
    out = proc.stdout.read()
    err = proc.stderr.read() if proc.stderr else ""
    return out, err


def run_shell_tests(task_id) -> None:
    # py -m determined.cli.tunnel http://localhost:8080 task_id
    """
    ws.connect: Connecting(url='ws://localhost:8080/proxy/9f063848-5c81-4116-a381-8b3dddb36ffb/')
    ws.connect: Connected(url='ws://localhost:8080/proxy/9f063848-5c81-4116-a381-8b3dddb36ffb/')
    ws.connect: Ready(response=<response HTTP/1.1 101 Swit
    ready
    ws.connect: Poll()
    ws.connect: Binary(data=b'SSH-2.0-OpenSSH_8.2p1 Ub' + 18 bytes)
    SSH-2.0-OpenSSH_8.2p1 Ubuntu-4ubuntu0.11
    ws.connect: Poll()
    ws.connect: Poll()
    """
    cmd = ["python", "-m", "determined.cli.tunnel", "http://localhost:8080", task_id]
    out, err = run_and_capture_long_running(cmd)
    assert "OpenSSH" in out, f"failed to tunnel {out}, {err}"
    # out, err = run_and_capture_long_running(["det", "shell", "open", task_id])
    # assert "root@" in out, f"failed to open shell {out}, {err}"


def run_det_tests(task_id) -> None:
    nbs = call_det_api(["get_GetNotebooks"])
    shells = call_det_api(["get_GetShells"])
    if task_id in nbs:
        run_notebook_tests(task_id)
    elif task_id in shells:
        run_shell_tests(task_id)


def _run_tests(pod_ip: str, pod_port, task_id: str, local_port, prefix: str = "") -> None:
    """
    prefix needs trailing slash
    """
    interests = [
        ["curl", "-s", f"http://{pod_ip}:{pod_port}/{prefix}", "-m", "1"],
        [
            "curl",
            "-s",
            f"http://{pod_ip}:{pod_port}/{prefix}proxy/{task_id}/",
            "-m",
            "1",
        ],
        ["curl", "-s", f"http://localhost:{local_port}/{prefix}", "-m", "1"],
        [
            "curl",
            "-s",
            f"http://localhost:{local_port}/{prefix}proxy/{task_id}/",
            "-m",
            "1",
        ],
        ["curl", "-s", f"http://localhost:{local_port}/{prefix}lab", "-m", "1"],
        [
            "curl",
            "-s",
            f"http://localhost:{local_port}/{prefix}lab",
            "-m",
            "1",
            "|",
            "grep",
            "'not exist'",
        ],
        [
            "curl",
            "-s",
            f"http://localhost:{local_port}/{prefix}proxy/{task_id}/lab",
            "-m",
            "1",
        ],
        [
            "curl",
            "-s",
            f"http://localhost:{local_port}/{prefix}proxy/{task_id}/lab",
            "-m",
            "1",
            "|",
            "grep",
            "'not exist'",
        ],
        [
            "curl",
            "-s",
            f"https://localhost:443/{prefix}proxy/{task_id}/lab",
            "-m",
            "1",
            "--insecure",
        ],
        [
            "curl",
            "-s",
            f"https://localhost:443/{prefix}proxy/{task_id}/lab",
            "-m",
            "1",
        ],
        [
            "curl",
            "-s",
            f"http://localhost:80/{prefix}proxy/{task_id}/lab",
            "-m",
            "1",
        ],
        ["det", "dev", "curl", f"/proxy/{task_id}/", "-s", "-m", "1"],
    ]
    for cmd in interests:
        print(termcolor.colored(f"Running: {' '.join(cmd)}", "yellow"))
        finished_proc = sp.run(" ".join(cmd), stdout=sp.PIPE, shell=True)
        out = finished_proc.stdout.decode("utf-8")
        got_html = "html" in out
        return_code = finished_proc.returncode
        print(f"Return code: {return_code}, Got HTML: {got_html}, isnb: {is_jl_page(out)}")

    run_det_tests(task_id)


def run_tests(task_id: str, include_ingress_prefix=True):
    pod = get_pod_details(task_id)
    prefix = ""
    if include_ingress_prefix:
        prefix = f"det-{task_id[:8]}/"
    _run_tests(pod.ip, pod.port, task_id, DEFAULT_LOCAL_PORT, prefix=prefix)


def create_test_tasks(rp: str) -> Dict[str, str]:
    task_ids = {
        "notebook": "",
        "tensorboard": "",
        "shell": "",
    }
    req_config = {"slots": 0, "resource_pool": rp}
    task_id = (
        sp.check_output(
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
    task_ids["notebook"] = task_id
    task_id = (
        sp.check_output(
            [
                "det",
                "shell",
                "start",
                "--config",
                f"resources={json.dumps(req_config)}",
                "-d",
            ]
        )
        .decode("utf-8")
        .strip()
    )
    task_ids["shell"] = task_id
    # task_id = (
    #     sp.check_output(
    #         [
    #             "det",
    #             "tensorboard",
    #             "start",
    #             "1",
    #             "--config",
    #             f"resources={json.dumps(req_config)}",
    #             "-d",
    #         ]
    #     )
    #     .decode("utf-8")
    #     .strip()
    # )
    # task_ids["tensorboard"] = task_id
    print(task_ids)
    return task_ids


def get_pod_details(task_id: str) -> PodDetails:
    pod_name = None
    while not pod_name:
        print(".", end="", flush=True)
        try:
            log_proc = sp.run(
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
                stderr=sp.DEVNULL,
                stdout=sp.PIPE,
            )
            log_proc.check_returncode()
            if log_proc.stdout:
                pod_name = log_proc.stdout.decode("utf-8").strip()
        except sp.CalledProcessError:
            pass
        time.sleep(0.5)
    print(f"Pod name: {pod_name}")

    pod_ip = None
    while not pod_ip:
        print(".", end="", flush=True)
        pod_ip = (
            sp.check_output(kctl(["get", "pod", pod_name, "-o", "jsonpath={.status.podIP}"]))
            .decode("utf-8")
            .strip()
        )
        time.sleep(0.5)

    pod_port = (
        sp.check_output(
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
    print(f"Pod IP: {pod_ip}, Pod Port: {pod_port}")
    return PodDetails(pod_name, pod_ip, int(pod_port))


def wait_for_pod(pod_name: str):
    while True:
        print(".", end="", flush=True)
        pod_status = (
            sp.check_output(
                kctl(["get", "pod", pod_name, "-o", "jsonpath={.status.phase}"]),
                stderr=sp.DEVNULL,
            )
            .decode("utf-8")
            .strip()
        )
        if pod_status == "Running":
            break
        time.sleep(0.5)


def kctl_forward(pod_name: str, local_port: int, pod_port) -> sp.Popen:
    forward_cmd = kctl(["port-forward", pod_name, f"{local_port}:{pod_port}"])
    print(f"Port forward: {' '.join(forward_cmd)}")
    time.sleep(3)  # pod ready is not enough
    pforward = sp.Popen(forward_cmd, stdout=sp.PIPE, text=True)
    while True:
        if pforward.stdout:
            line = pforward.stdout.readline()
            if "forwarding from" in line.lower():
                break
    return pforward


def wait_for_port(host: str, port: int):
    print(f"Waiting for port {port} on {host}")
    while True:
        print(".", end="", flush=True)
        nc_proc = sp.run(["nc", "-vz", host, str(port)], stderr=sp.PIPE)
        if nc_proc.returncode == 0:
            break
        time.sleep(0.5)


def kubectl_apply(file: str):
    """apply kubectl file."""
    cmd = kctl(["apply", "--validate=strict", "-f", file])
    run_command(" ".join(cmd))


def filter_pods(name_regex: str) -> List[str]:
    """filter pods by name."""
    pod_names: List[str] = (
        run_command(
            "kubectl get pods --all-namespaces -o jsonpath='{.items[*].metadata.name}'",
            capture=True,
        )
        .stdout.strip()
        .split(" ")
    )
    return [name for name in pod_names if name_regex in name]


def create_deployment(image: str, name: str, port: int) -> None:
    """create k8s deployment."""
    deployment_manifest = f"""
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {name}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: {name}
  template:
    metadata:
      labels:
        app: {name}
    spec:
      containers:
      - name: {name}
        image: {image}
        ports:
        - containerPort: {port}
"""
    with open("deployment.yaml", "w") as file:
        file.write(deployment_manifest)


def create_service(app_name: str, port: int, target_port: int, selector: Optional[str] = "") -> str:
    if not selector:
        selector = f"app: {app_name}"
    """create k8s service."""
    service_manifest = f"""
apiVersion: v1
kind: Service
metadata:
  name: {app_name}-service
spec:
  type: ClusterIP
  selector:
    {selector}
    # app: name
  ports:
    - protocol: TCP
      port: {port}
      targetPort: {target_port}
"""
    with open("service.yaml", "w") as file:
        file.write(service_manifest)
    run_command("kubectl apply -f service.yaml")
    return f"{app_name}-service"


def download_files_from_pod(pod_name: str, src: str, dest: str) -> None:
    """download files from a pod."""
    run_command(
        kctl(["cp", f"{pod_name}:{src}", dest]),
        capture=True,
    )


def create_ingress(ingress_name: str = "det") -> str:
    """ensure there is an ingress."""
    file_exists = run_command("ls ingress.yaml", capture=True).returncode == 0
    full_name = f"{ingress_name}-ingress"

    # check if ingress.networking.k8s.io/{full_name} exists

    proc = run_command(kctl(["get", "ingress", full_name]), capture=True)
    ingress_exists = proc.returncode == 0
    if ingress_exists and file_exists:
        print(f"Ingress {full_name} exists")
        return full_name

    ingress_manifest = f"""
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: {full_name}
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /$2
    nginx.ingress.kubernetes.io/upgrade-proxy: "true"  # Important for WebSocket
    nginx.ingress.kubernetes.io/proxy-read-timeout: "3600"
    nginx.ingress.kubernetes.io/proxy-send-timeout: "3600"
    nginx.org/ssl-services: "tls-passthrough-service"
    nginx.org/ssl-backends: "true"
    nginx.ingress.kubernetes.io/backend-protocol: "HTTPS"
    # traefik.ingress.kubernetes.io/router.entrypoints: web

    # traefik.ingress.kubernetes.io/router.entrypoints: web
    #  # (/|$)(.*)
spec:
  rules:
  - http:
      paths:
"""
    with open("ingress.yaml", "w") as file:
        file.write(ingress_manifest)
    return full_name


def _add_route(path: str, service_name: str) -> None:
    "add a route to existing ingress.yaml"
    service_port = DEFAULT_SERVICE_PORT
    with open("ingress.yaml", "r") as file:
        ingress_manifest = file.read()
        new_route = f"""
- path: {path}(/|$)(.*)
  # pathType: Prefix
  pathType: ImplementationSpecific
  backend:
    service:
      name: {service_name}
      port:
        number: {service_port}
"""
        # remove first and last empty lines
        TAB = " " * 2
        new_route = new_route.replace("\n", f"\n{TAB * 4}")
        ingress_manifest = ingress_manifest.replace("paths:", f"paths:\n{new_route}")
        # remove empty lines w/ just space
        ingress_manifest = re.sub(r"\n\s*\n", "\n", ingress_manifest)

    # save
    with open("ingress.yaml", "w") as file:
        file.write(ingress_manifest)


def add_route(name: str) -> None:
    "add a route to existing ingress.yaml"
    _add_route(f"/{name}", f"{name}-service")


def report() -> None:
    """report ingress, pods, and services."""
    run_command("kubectl get pods,services,ingress")


def create_http_test_setup() -> None:
    podConfig = PodConfig("httptest", "crccheck/hello-world", 8000)
    create_ingress()
    create_deployment(podConfig.image, podConfig.name, podConfig.port)
    kubectl_apply("deployment.yaml")
    create_service(podConfig.name, DEFAULT_SERVICE_PORT, podConfig.port)
    _add_route("/", f"{podConfig.name}-service")
    kubectl_apply("ingress.yaml")
    report()


def create_test_setup() -> None:
    """create a test setup with two pods.
    minikube: minikube addons enable ingress
    minikube tunnel
    """
    configs = [
        ("wstest", "ksdn117/web-socket-test", 8010),
        ("httptest", "crccheck/hello-world", 8000),
    ]
    pods = [PodConfig(*config) for config in configs]
    create_ingress()
    for name, image, port in pods:
        create_deployment(image, name, port)
        kubectl_apply("deployment.yaml")
        create_service(name, DEFAULT_SERVICE_PORT, port)
        add_route(name)
    kubectl_apply("ingress.yaml")
    report()


def demo_kport_forward(
    rp: str = "default", task_id: Optional[str] = None, context: Optional[str] = None
) -> None:
    if context:
        check_context(context)
        global TARGET_CONTEXT
        TARGET_CONTEXT = context
    if not task_id:
        task_id = create_test_tasks(rp)["notebook"]
    pod = get_pod_details(task_id)
    try:
        pforward = kctl_forward(pod.name, DEFAULT_LOCAL_PORT, pod.port)
        wait_for_port("localhost", DEFAULT_LOCAL_PORT)
        _run_tests(pod.ip, pod.port, task_id, DEFAULT_LOCAL_PORT)
        input("Press Enter to cleanup")
    finally:
        pforward.terminate()


def wait_for_ingress(ingress_name: str) -> None:
    time.sleep(3)
    while True:
        print(".", end="", flush=True)
        try:
            pass
            break
        except sp.CalledProcessError:
            pass
        time.sleep(0.5)


def setup_ingress_for_task(task_id: str) -> Tuple[str, PodDetails]:
    print(f"Setting up ingress for task {task_id}")
    pod = get_pod_details(task_id)
    app_name = f"det-{task_id[:8]}"
    create_service(app_name, DEFAULT_SERVICE_PORT, pod.port, f"determined.ai/task_id: {task_id}")
    ingress_name = create_ingress()
    wait_for_ingress(ingress_name)
    # ingress = get_ingress_details(ingress_name)
    # print(ingress)
    _add_route(f"/{app_name}", f"{app_name}-service")
    kubectl_apply("ingress.yaml")
    run_det_tests(task_id)
    return app_name, pod


def demo_ingress_flow(
    rp: str = "default", task_id: Optional[str] = None, context: Optional[str] = None
) -> None:
    if context:
        check_context(context)
        global TARGET_CONTEXT
        TARGET_CONTEXT = context
    if not task_id:
        task_id = create_test_tasks(rp)["notebook"]
    app_name, pod = setup_ingress_for_task(task_id)
    wait_for_port("localhost", DEFAULT_LOCAL_PORT)
    report()
    _run_tests(pod.ip, pod.port, task_id, DEFAULT_LOCAL_PORT, prefix=f"{app_name}/")


if __name__ == "__main__":
    fire.Fire(
        {
            "demo_kport_forward": demo_kport_forward,
            "demo_ingress_flow": demo_ingress_flow,
            "cleanup_cluster": cleanup_cluster,
            "create_http_test_setup": create_http_test_setup,
            "setup_ingress_for_task": setup_ingress_for_task,
            "run_det_tests": run_det_tests,
            "run_tests": run_tests,
            "create_test_tasks": create_test_tasks,
        }
    )
