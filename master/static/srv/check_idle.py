import requests
import json
import os
from time import sleep
from determined.common import api

ACTIVE = False
IDLE = True


def get_execution_state(request_address):
    try:
        kernels_res = requests.get(request_address + "/api/kernels")
        kernels_data = json.loads(kernels_res.text)

        terminals_res = requests.get(request_address + "/api/terminals")
        terminals_data = json.loads(terminals_res.text)

        sessions_res = requests.get(request_address + "/api/sessions")
        sessions_data = json.loads(sessions_res.text)
    except Exception as err:
        print("Cannot get notebook kernel status", err)
        return ACTIVE

    if len(kernels_data) == 0 and len(terminals_data) == 0 and len(sessions_data) == 0:
        return IDLE
    return ACTIVE

def main():
    while True:
        sleep(1)
        url = "http://127.0.0.1:" + str(os.environ["NOTEBOOK_PORT"]) + "/proxy/"+ str(os.environ["DET_TASK_ID"])
        try:
            api.put(
                str(os.environ["DET_MASTER_HOST"]),
                "/api/v1/notebooks/{notebook_id}/report_idle",
                {
                    "notebook_id": str(os.environ["DET_TASK_ID"]),
                    "idle": get_execution_state(url),
                },
            )
        except Exception as e:
            print(e)


if __name__ == "__main__":
    main()
