import lomond
from lomond import events

url = "http://localhost:8080"

# steal determined token
from determined.experimental import client
client.login(url)
token = client._determined._session._auth.session.token

_state = 0
def late_sub(ws):
    global _state
    _state += 1
    if _state == 3:
        print("------ updating subscription ------------")
        ws.send_binary(
            b'{"add": {"trials": {"trial_ids": [11]}}, "drop": {"trials": {"trial_ids": [1]}}}'
        )

ws = lomond.WebSocket(f"{url}/stream")
ws.add_header(b"Authorization", f"Bearer {token}".encode('utf8'))
for event in ws.connect():
    if isinstance(event, events.Binary):
        print(event.data.decode('utf8'))
        late_sub(ws)
    elif isinstance(event, events.Text):
        print(event.text.strip())
        late_sub(ws)
    elif isinstance(event, events.Ready):
        print("ready")
        ws.send_binary(
            b'{"subscribe": {"trials": {"trial_ids": [1], "since": 1000}},'
            b' "known": {"trials": "1,99-110,1000-8000"}}'
        )
    elif isinstance(event, (events.ConnectFail, events.Rejected, events.ProtocolError)):
        raise Exception(f"connection failed: {event}")
    elif isinstance(event, (events.Closing, events.Disconnected)):
        break
