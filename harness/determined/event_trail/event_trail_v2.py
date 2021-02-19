from determined.event_trail.event_trail_events import EventTrailEvent, EventPriority
from determined.event_trail.event_trail_events import TrialInfoEventV1
from typing import List, Any, Callable, Union
import threading
import time

EVENT_TYPES = [
    TrialInfoEventV1
]


class SingleTrail:
    """
    Handle the queue, locks, and sending events for a single type of event.
    EventTrailThread will have one SingleTrail per event type.
    """

    def __init__(
            self,
            event_type_name: str,
            processor_fn: Callable[[List, str], None],
            priority: EventPriority,
            max_batch_size: int,
            master_url: str
    ):
        self.event_type_name = event_type_name
        self.q = []
        self.lock = threading.Lock()
        self.processor_fn = processor_fn
        self.priority = priority
        self.max_batch_size = max_batch_size
        self.master_url = master_url

    def enqueue(self, event):
        with self.lock:
            self.q.append(event)

    def pop_batch(self) -> List[EventTrailEvent]:
        with self.lock:
            if self.max_batch_size == -1 or self.max_batch_size >= len(self.q):
                ret = self.q
                self.q = []
            else:
                ret = self.q[:self.max_batch_size]
                del self.q[:self.max_batch_size]
        return ret

    def length(self):
        with self.lock:
            return len(self.q)

    def send_events_if_present(self) -> bool:
        batch = self.pop_batch()
        if len(batch) > 0:
            self.processor_fn(batch, self.master_url)
            return True
        return False

    def send_all(self):
        while self.length() > 0:
            self.send_events_if_present()


class EventTrailThread(threading.Thread):
    """
    Background thread for sending events to the master asynchronously
    """

    def __init__(self, master_address, master_port, use_tls, debug_logs=False) -> None:
        self.verbose = debug_logs
        self.log("Creating EventTrailThread")

        scheme = "https" if use_tls else "http"
        self.master_url = f"{scheme}://{master_address}:{master_port}"
        self.log("Master URL is", self.master_url)

        self.trails = {}
        for EVENT_TYPE in EVENT_TYPES:
            event_type_name = EVENT_TYPE.event_name()
            self.trails[event_type_name] = SingleTrail(event_type_name,
                                                       EVENT_TYPE.process_batch,
                                                       EVENT_TYPE.priority(),
                                                       EVENT_TYPE.max_batch_size(),
                                                       self.master_url)
        self.quitting = False
        self.lock = threading.Lock()
        super().__init__()

    def log(self, *s):
        if self.verbose:
            print("[EventTrailThread]", *s)

    def run(self) -> None:
        with self.lock:
            while True:
                if self.quitting:
                    break

                should_sleep = True
                for trail in self.trails.values():
                    sent_event = trail.send_events_if_present()
                    if sent_event:
                        should_sleep = False

                if should_sleep:
                    time.sleep(1)

    def enqueue_for_async_send(self, event: EventTrailEvent):
        event_type_name = event.event_name()
        self.log("Enqueuing", event_type_name, event.as_dict())
        self.trails[event_type_name].enqueue(event)

    def _cleanup_on_shutdown(self):
        for trail in self.trails.values():
            if trail.priority == EventPriority.MUST_DELIVER:
                trail.send_all()

    def __enter__(self) -> "EventTrailThread":
        self.start()
        return self

    def __exit__(self, *arg: Any) -> None:
        self.quitting = True
        with self.lock:
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


if __name__ == '__main__':
    with EventTrailThread(master_address="localhost", master_port="8080", use_tls=False) as event_trail:
        pass
