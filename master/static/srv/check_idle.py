import enum
import logging
import os
import socket
import time
from typing import Tuple

import requests

from determined.common import api
from determined.common.api import certs


class IdleType(enum.Enum):
    KERNELS_OR_TERMINALS = 1
    KERNEL_CONNECTIONS = 2
    ACTIVITY = 3


REPORT_IDLE_INTERVAL = 30

last_activity = None


def wait_for_jupyter(addr: Tuple[str, int]) -> None:
    """
    Avoid logging enormous stacktraces when the requests library attempts to connect to a server
    that isn't accepting connections yet.  This is expected as jupyter startup time might take
    longer than a second, and we don't want to generate scary logs for expected behavior.
    """
    i = 0
    while True:
        with socket.socket() as s:
            try:
                s.connect(addr)
                # Connection worked, we're done here.
                return
            except ConnectionError as e:
                if (i + 1) % 10 == 0:
                    # Every 10 seconds without reaching jupyter, start telling the user.
                    # This is beyond the range of expected startup times.
                    logging.warning(f"jupyter is still not reachable at {addr}")
            time.sleep(1)
            i += 1


def is_idle(request_address: str, mode: IdleType) -> bool:
    try:
        kernels = requests.get(request_address + "/api/kernels", verify=False).json()
        terminals = requests.get(request_address + "/api/terminals", verify=False).json()
        sessions = requests.get(request_address + "/api/sessions", verify=False).json()
    except Exception:
        logging.warning("Cannot get notebook kernel status", exc_info=True)
        return False

    if mode == IdleType.KERNELS_OR_TERMINALS:
        return len(kernels) == 0 and len(terminals) == 0 and len(sessions) == 0
    elif mode == IdleType.KERNEL_CONNECTIONS:
        # Unfortunately, the terminals API doesn't return a connection count.
        return all(k["connections"] == 0 for k in kernels)
    elif mode == IdleType.ACTIVITY:
        global last_activity

        old_last_activity = last_activity
        if kernels or terminals:
            last_activity = max(x["last_activity"] for x in kernels + terminals)
        no_busy_kernels = all(k["execution_state"] != "busy" for k in kernels)

        return no_busy_kernels and (last_activity == old_last_activity)
    return False


def main() -> None:
    requests.packages.urllib3.disable_warnings()  # type: ignore
    port = os.environ["NOTEBOOK_PORT"]
    notebook_id = os.environ["DET_TASK_ID"]
    notebook_server = f"https://127.0.0.1:{port}/proxy/{notebook_id}"
    master_url = os.environ["DET_MASTER"]
    cert = certs.default_load(master_url)
    try:
        idle_type = IdleType[os.environ["NOTEBOOK_IDLE_TYPE"].upper()]
    except KeyError:
        logging.warning(
            "unknown idle type '%s', using default value", os.environ["NOTEBOOK_IDLE_TYPE"]
        )
        idle_type = IdleType.KERNELS_OR_TERMINALS

    wait_for_jupyter(("127.0.0.1", int(port)))

    while True:
        try:
            idle = is_idle(notebook_server, idle_type)
            api.put(
                master_url,
                f"/api/v1/notebooks/{notebook_id}/report_idle",
                {"notebook_id": notebook_id, "idle": idle},
                cert=cert,
            )
        except Exception:
            logging.warning("ignoring error communicating with master", exc_info=True)
        time.sleep(REPORT_IDLE_INTERVAL)


if __name__ == "__main__":
    main()
