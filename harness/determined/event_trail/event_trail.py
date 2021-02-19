import threading
import time
from typing import Any, Callable, Dict, Generator, Union, List, Type
from determined.event_trail.event_trail_events import EventTrailEvent, EventPriority, TrialInfoEventV1
from enum import Enum
import requests
import json



class bcolors(Enum):
    HEADER = '\033[95m'
    OKBLUE = '\033[94m'
    OKCYAN = '\033[96m'
    OKGREEN = '\033[92m'
    WARNING = '\033[93m'
    FAIL = '\033[91m'
    ENDC = '\033[0m'
    BOLD = '\033[1m'
    UNDERLINE = '\033[4m'


def colorize(s: str, color: bcolors):
    return f"{color.value}{s}{str(bcolors.ENDC.value)}"


def send_event(event: EventTrailEvent):
    log_line = colorize(f'EventSenderThread sending: {str(event)}', bcolors.OKBLUE)
    print(log_line)


class EventTrailThread(threading.Thread):
    """
    Background thread for sending events to the master asynchronously
    """

    def __init__(self, master_address, master_port, use_tls) -> None:
        self.master_address = master_address
        self.master_port = master_port
        self.use_tls = use_tls
        use_tls = False  # Failing on local tests
        self.master_url = f"http{'s' if use_tls else ''}://{master_address}:{master_port}"
        self.events = []  # type: List[EventTrailEvent]
        self.quitting = False
        self.main_event_loop_exited = False
        super().__init__()
        print("EventSenderThread init")

    def run(self) -> None:
        while True:
            print("EventSenderThread While Loop")
            if self.quitting:
                break

            if len(self.events) > 0:
                print("EventSenderThread While Loop Events Exist")
                self._send_events()

            if len(self.events) == 0:
                time.sleep(1)

        self.main_event_loop_exited = True

    def enqueue_for_async_send(self, event: EventTrailEvent):
        self.events.append(event)
        print("EventSenderThread enqueue_for_async_send", str(event), len(self.events))

    def _send_events(self):
        event = self.events.pop()
        if isinstance(event, TrialInfoEventV1):
            post_url = f"{self.master_url}/api/v1/telemetry/trial-info"
            send_event(event)

            headers = {
                'accept': 'application/json',
                'Content-Type': 'application/json',
                'Cookie': f'auth=v2.public.eyJpZCI6NywidXNlcl9pZCI6MSwiZXhwaXJ5IjoiMjAyMC0xMC0yOVQxMzowNjoxOC4yOTM4NTItMDc6MDAifef4KehbcCjUph10IFTA8vUFeB9wloTn6aGzVsh-xfzuei717VOaWkAmcrfMHOUeEajIH5vjzKHVJAI5UZPI6wY.bnVsbA'
            }

            # data = '{ "experimentId": 0, "trialId": 0, "trialType": "PYTORCH"}'
            # response = requests.post('http://localhost:8080/api/v1/telemetry/trial-info', headers=headers, data=data)

            data = {
                "experimentId": event.experiment_id,
                "trialId": event.trial_id,
                "trialFramework": event.framework.value
            }
            data = json.dumps(data)
            response = requests.post(post_url, headers=headers, data=data)
            print("Attempted to send event", data, "to", post_url, ". Response was:", response.text)
        else:
            # TODO: handle this
            pass

    def _cleanup_on_shutdown(self):
        # If not many events, send them
        # Otherwise, send any MUST_DELIVER events
        print("EventSenderThread - cleaning up")
        pass

    def __enter__(self) -> "EventTrailThread":
        print("EventSenderThread.__enter__")
        self.start()
        print("Finished EventSenderThread.__enter__")
        return self

    def __exit__(self, *arg: Any) -> None:
        self.quitting = True
        while not self.main_event_loop_exited:
            time.sleep(0.01)
        self._cleanup_on_shutdown()



class NoOpEventTrail:
    """
    No-op equivalent of EventTrailThread (for measuring performance impact of EventTrailThread)
    """
    def __init__(self, master_address, master_port, use_tls) -> None:
        pass

    def enqueue_for_async_send(self, event: EventTrailEvent):
        pass

    def __enter__(self) -> "NoOpEventTrail":
        return self

    def __exit__(self, *arg: Any) -> None:
        pass


# These type must implement enqueue_for_async_send(event: EventTrialEvent) and __enter__/__exit__
TypeEventTrailThread = Union[EventTrailThread, NoOpEventTrail]


def create_event_trail_thread(master_address, master_port, use_tls, noop=False) -> TypeEventTrailThread:
    if noop:
        return NoOpEventTrail(master_address, master_port, use_tls)
    else:
        return EventTrailThread(master_address, master_port, use_tls)
