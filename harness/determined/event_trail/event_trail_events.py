from typing import Dict, Any
from enum import Enum
from abc import ABC, abstractmethod
import json  # TODO: Simplejson instead?


class EventPriority(Enum):
    MUST_DELIVER = 1
    DROPPABLE = 2


class EventTrailEvent(ABC):


    @property
    @abstractmethod
    def priority(self) -> EventPriority:
        pass


    @property
    @abstractmethod
    def version(self) -> int:
        pass

    @abstractmethod
    def as_dict(self) -> Dict[str, Any]:
        pass

    def __str__(self):
        return json.dumps(self.as_dict())

    # TODO: Should the delivery code be included inside of the EventTrailEvent? It makes it easier
    #       to know how to add a new event, but it makes batching sends harder.


class TrialInfoEventV1(EventTrailEvent):
    class TrialFramework(Enum):
        PYTORCH = 1
        ESTIMATOR = 2
        KERAS = 3

    def __init__(self, experiment_id: int, trial_id: int, framework: TrialFramework):
        self.experiment_id = experiment_id
        self.trial_id = trial_id
        self.framework = framework

    @property
    def priority(self) -> EventPriority:
        return EventPriority.MUST_DELIVER

    @property
    def version(self) -> int:
        return 1

    def as_dict(self) -> Dict[str, Any]:
        return {
            "event_type": "TrialInfoEventV1",
            "version": self.version,
            "experiment_id": self.experiment_id,
            "trial_id": self.trial_id,
            "framework": str(self.framework)
        }



EventTypes = [
    TrialInfoEventV1
]