import json
import sys
import time
from typing import List

from determined_common import api
from kubernetes import client, config


def pwChange(user: str, password: str, entrypoint: str) -> None:
    print(f"Trying to change {user}'s password", flush=True)
    while True:
        try:
            resp = api.post(
                entrypoint,
                "/api/v1/auth/login",
                body={"username": user, "password": ""},
                authenticated=False,
            )

            data = json.loads(resp.text)
            api.post(
                entrypoint,
                f"/api/v1/users/{data['user']['id']}/password",
                body=password,
                headers={"Authorization": f"Bearer {data['token']}"},
                authenticated=False,
            )
            print(f"Sucessfully changed {user}'s password", flush=True)
            return
        except Exception as e:
            print(f"Encountered exception: {e}", flush=True)
            return


def checkPortAlive(entrypoint: str) -> None:
    while True:
        try:
            api.get(entrypoint, "/api/v1/master", authenticated=False)
            return
        except Exception as e:
            print(f"Encountered exception: {e}")
            continue


def getMasterAddress(namespace: str, service_name: str, master_port: str, node_port: str) -> str:
    config.load_incluster_config()
    v1 = client.CoreV1Api()
    target_service = f"determined-master-service-{service_name}"

    if node_port != "true":
        while True:
            services = v1.list_namespaced_service(namespace)
            for svc in services.items:
                if target_service in svc.metadata.name:
                    ingress = svc.status.load_balancer.ingress
                    if ingress is None:
                        time.sleep(1)
                        break
                    if ingress[0].hostname is not None:
                        # use hostname over ip address, if available
                        return f"{ingress[0].hostname}:{master_port}"
                    else:
                        return f"{ingress[0].ip}:{master_port}"

    services = v1.list_namespaced_service(namespace)
    for svc in services.items:
        if target_service in svc.metadata.name:
            entrypoint = f"{svc.spec.cluster_ip}:{master_port}"
            checkPortAlive(entrypoint)

            return entrypoint
    return ""


def main(argv: List[str]) -> None:
    if len(argv) < 5:
        raise Exception("not enough args")
    if argv[4] == "":
        raise Exception("no password supplied")

    for i in range(len(argv)):
        argv[i] = argv[i].strip()
    addr = getMasterAddress(argv[0], argv[1], argv[2], argv[3])
    pwChange("determined", argv[4], addr)
    pwChange("admin", argv[4], addr)


if __name__ == "__main__":
    main(sys.argv[1:])
