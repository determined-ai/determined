from typing import Dict, Any, List
from enum import Enum
from abc import ABC, abstractmethod
import json  # TODO: Simplejson instead?


class EventPriority(Enum):
    MUST_DELIVER = 1
    DROPPABLE = 2


class EventTrailEvent(ABC):

    @staticmethod
    @abstractmethod
    def version() -> int:
        """
        Version number for this event.
        """
        pass

    @staticmethod
    @abstractmethod
    def priority() -> EventPriority:
        """
        During cleanup, should the harness block exiting until those events have been
        sent? Most events should be DROPPABLE.
        """
        pass

    @staticmethod
    @abstractmethod
    def event_name() -> str:
        pass

    @staticmethod
    @abstractmethod
    def process_batch(batch: List["EventTrailEvent"], master_url: str) -> None:
        pass

    @staticmethod
    @abstractmethod
    def max_batch_size() -> int:
        """
        The largest batch size we can process at one time. -1 indicates no limit to the batch size
        """
        pass

    # TODO: Should this even exist?
    @abstractmethod
    def as_dict(self) -> Dict[str, Any]:
        """
        Just
        """
        pass

    def __str__(self):
        return json.dumps(self.as_dict())







    # def _send_events(self):
    #     event = self.events.pop()
    #     if isinstance(event, TrialInfoEventV1):
    #         post_url = f"{self.master_url}/api/v1/telemetry/trial-info"
    #         send_event(event)
    #
    #         headers = {
    #             'accept': 'application/json',
    #             'Content-Type': 'application/json',
    #             'Cookie': f'auth=v2.public.eyJpZCI6NywidXNlcl9pZCI6MSwiZXhwaXJ5IjoiMjAyMC0xMC0yOVQxMzowNjoxOC4yOTM4NTItMDc6MDAifef4KehbcCjUph10IFTA8vUFeB9wloTn6aGzVsh-xfzuei717VOaWkAmcrfMHOUeEajIH5vjzKHVJAI5UZPI6wY.bnVsbA'
    #         }
    #
    #         # data = '{ "experimentId": 0, "trialId": 0, "trialType": "PYTORCH"}'
    #         # response = requests.post('http://localhost:8080/api/v1/telemetry/trial-info', headers=headers, data=data)
    #
    #         data = {
    #             "experimentId": event.experiment_id,
    #             "trialId": event.trial_id,
    #             "trialFramework": event.framework.value
    #         }
    #         data = json.dumps(data)
    #         response = requests.post(post_url, headers=headers, data=data)
    #         print("Attempted to send event", data, "to", post_url, ". Response was:", response.text)
    #     else:
    #         # TODO: handle this
    #         pass
    #

# TODO: Rename to TrialFrameworkEvent
class TrialInfoEventV1(EventTrailEvent):

    class TrialFramework(Enum):
        PYTORCH = "PYTORCH"
        ESTIMATOR = "ESTIMATOR"
        KERAS = "KERAS"


    @staticmethod
    def event_name() -> str:
        return "TrialInfoEventV1"

    @staticmethod
    def process_batch(batch: List["TrialInfoEventV1"], master_url: str) -> None:
        assert len(batch) == 1, "TrialInfoEventV1 cannot handle batches larger than 1"
        event = batch[0]
        # TODO: Implement
        pass

    @staticmethod
    def max_batch_size() -> int:
        # No batching in this API
        return 1

    def __init__(self, experiment_id: int, trial_id: int, framework: TrialFramework):
        self.experiment_id = experiment_id
        self.trial_id = trial_id
        self.framework = framework

    @staticmethod
    def priority() -> EventPriority:
        return EventPriority.MUST_DELIVER

    @staticmethod
    def version() -> int:
        return 1

    def as_dict(self) -> Dict[str, Any]:
        return {
            "eventType": "TrialInfoEvent",
            "eventVersion": self.version,
            "experimentId": self.experiment_id,
            "trialId": self.trial_id,
            "framework": self.framework.value
        }


