import time
import uuid
from string import Template

import lomond
from lomond import events

from determined.experimental import client

url = "localhost:8080"

# steal determined token
client.login(url)
print(f"Logged in as: {client._determined._session._auth.session.username}")
token = client._determined._session._auth.session.token


def create_payload(trials=None, experiments=None) -> bytes:
    sync_id = uuid.uuid4()
    payload_template = Template(
        """
    {
        "sync_id": "$sync_id",
        "subscribe": {
            "trials": {
                "trial_ids":      [$trial_ids],
                "experiment_ids": [$experiment_ids],
                "since": 0
            },
            "metrics": {
                "trial_ids":      [$trial_ids],
                "since": 0
            },
            "experiments": {
                "experiment_ids": [$experiment_ids],
                "since": 0
            }
        },
        "known": {
            "trials":  "$trial_ids",
            "experiments": "$experiment_ids"
        }
    }
    """
    )

    if trials is None:
        trials = input("Submit trials to subscribe to (default: 1,2,3): ") or "1,2,3"
    if experiments is None:
        experiments = (
            input("Submit experiments to subscribe to (default: 1,2,3): ") or "1,2,3"
        )

    startupMsg = payload_template.substitute(
        sync_id=sync_id,
        trial_ids=trials,
        experiment_ids=experiments,
    )
    print(f"startupMsg ({sync_id}) created")
    return startupMsg.encode()


def stream_loop():
    max_retries = 5
    num_retries = 0

    ws = lomond.WebSocket(f"http://{url}/stream")
    ws.add_header(b"Authorization", f"Bearer {token}".encode("utf8"))
    try:
        for event in ws.connect():
            if isinstance(event, events.Binary):
                print(event.data.decode("utf8"))
            elif isinstance(event, events.Text):
                print(event.text.strip())
            elif isinstance(event, events.Ready):
                print("ready")
                ws.send_binary(create_payload())
                # ws.send_binary(create_payload())
            elif isinstance(
                event, (events.ConnectFail, events.Rejected, events.ProtocolError)
            ):
                print("Connection failed")
                raise Exception(f"connection failed: {event}")
            elif isinstance(event, (events.Closing, events.Disconnected)):
                print("Connection closed...attempting reconnect")
                num_retries += 1
                if num_retries < max_retries:
                    stream_loop()
                else:
                    print("exceeded max retries")
    except KeyboardInterrupt:
        print("")
    except Exception as e:
        print(e)


stream_loop()
