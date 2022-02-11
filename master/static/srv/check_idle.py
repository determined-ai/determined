import enum
import logging
import os
from time import sleep

import requests
from determined.common import api
from determined.common.api import certs


class IdleType(enum.Enum):
    KERNELS_OR_TERMINALS = 1
    KERNEL_CONNECTIONS = 2
    ACTIVITY = 3


last_activity = None


def is_idle(request_address, mode):
    try:
        kernels = requests.get(request_address + "/api/kernels").json()
        terminals = requests.get(request_address + "/api/terminals").json()
        sessions = requests.get(request_address + "/api/sessions").json()
    except Exception as err:
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



def main():
    port = os.environ["NOTEBOOK_PORT"]
    notebook_id = os.environ["DET_TASK_ID"]
    notebook_server = f"http://127.0.0.1:{port}/proxy/{notebook_id}"
    master_url = os.environ["DET_MASTER"]
    cert = certs.default_load(master_url)
    try:
        idle_type = IdleType[os.environ["NOTEBOOK_IDLE_TYPE"].upper()]
    except KeyError:
        logging.warning(
            "unknown idle type '%s', using default value", os.environ["NOTEBOOK_IDLE_TYPE"]
        )
        idle_type = IdleType.KERNELS_OR_TERMINALS

    while True:
        sleep(1)

        try:
            api.put(
                master_url,
                f"/api/v1/notebooks/{notebook_id}/report_idle",
                {
                    "notebook_id": notebook_id,
                    "idle": is_idle(notebook_server, idle_type),
                },
                cert=cert,
            )
        except Exception as e:
            logging.warning("ignoring error communicating with master", exc_info=True)


if __name__ == "__main__":
    main()
