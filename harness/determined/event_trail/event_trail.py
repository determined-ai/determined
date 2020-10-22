import threading
import time
from typing import Any, Callable, Dict, Generator, Union, List, Type
from determined.event_trail.event_trail_events import EventTrailEvent, EventPriority, TrialInfoEventV1
from enum import Enum


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
    return f"{str(color)}{s}{str(bcolors.ENDC)}"


def send_event(event: EventTrailEvent):
    log_line = colorize(f'EventSenderThread sending: {str(event)}', bcolors.OKGREEN)
    print(log_line)


class EventTrailThread(threading.Thread):
    """
    Background thread for sending events to the master asynchronously
    """

    def __init__(self) -> None:
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
            send_event(event)
        else:
            # TODO: handle this
            pass

    def _cleanup_on_shutdown(self):
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
    def __init__(self) -> None:
        pass

    def enqueue_for_async_send(self, event: EventTrailEvent):
        pass

    def __enter__(self) -> "NoOpEventTrail":
        return self

    def __exit__(self, *arg: Any) -> None:
        pass


# These type must implement enqueue_for_async_send(event: EventTrialEvent) and __enter__/__exit__
TypeEventTrailThread = Union[EventTrailThread, NoOpEventTrail]


def create_event_trail_thread(noop=False) -> TypeEventTrailThread:
    if noop:
        return NoOpEventTrail()
    else:
        return EventTrailThread()
