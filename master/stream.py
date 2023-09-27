import lomond
from lomond import events

from determined.experimental import client

url = "localhost:8080"


# steal determined token
client.login(url)
print(f"Logged in as: {client._determined._session._auth.session.username}")
token = client._determined._session._auth.session.token

trials = input("Submit array of trial_id values to subscribe to (default: [1]): ")
if trials == "":
    trials = "[1]"

max_retries = 5
num_retries = 0


def stream_loop():
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
                payload = (
                    '{"subscribe": {"trials": {"trial_ids": '
                    + str(trials)
                    + ', "since": 1000}}, "known": {"trials": "1,99-110,1000-8000"}}'
                )
                ws.send_binary(payload.encode())
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
