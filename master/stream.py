import lomond
from lomond import events
from string import Template

from determined.experimental import client

url = "localhost:8080"

payload_template = Template('''
{
    "subscribe": {
        "trials": {
            "trial_ids": $trial_ids,
            "since": 1000
        },
        "metrics": {
            "metric_ids": $metric_ids,
            "since": 1000
        }
    },
    "known": {
        "trials": "1,99-110,1000-8000",
        "metrics": "1,99-110,1000-8000"
    }
}
''')

# payload_template = Template('''
# {
#     "subscribe": {
#         "trials": {
#             "trial_ids": $trial_ids,
#             "since": 1000
#         }
#     },
#     "known": {
#         "trials": "1,99-110,1000-8000"
#     }
# }
# ''')

# steal determined token
client.login(url)
print(f"Logged in as: {client._determined._session._auth.session.username}")
token = client._determined._session._auth.session.token

trials = input("Submit array of trial_id values to subscribe to (default: [1]): ")
if trials == "":
    trials = "[1]"

metrics = input("Submit array of metric id values to subscribe to (default: [1]): ")
if metrics == "":
    metrics = "[1]"


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
                payload = payload_template.substitute(trial_ids=trials, metric_ids=metrics)
                # payload = 
                #     '{"subscribe": {"trials": {"trial_ids": '
                #     + str(trials)
                #     + ', "since": 1000}, "metrics": {"metric_ids": '
                #     + str(metrics)
                #     + ', "since": 1000}}, "known": {"trials": "1,99-110,1000-8000", "metrics": "1"}}'
                # )
                ws.send_binary(payload.encode())
            elif isinstance(
                event, (events.ConnectFail, events.Rejected, events.ProtocolError)
            ):
                print("Connection failed")
                raise Exception(f"connection failed: {event}")
            elif isinstance(event, (events.Closing, events.Disconnected)):
                print("Connection closed...attempting reconnect")
                num_retries += 1
                #if num_retries < max_retries:
                    #stream_loop()
                #else:
                 #   print("exceeded max retries")
    except KeyboardInterrupt:
        print("")
    except Exception as e:
        print(e)


stream_loop()
