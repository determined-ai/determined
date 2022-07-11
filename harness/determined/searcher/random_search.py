import dataclasses
import enum
import math
import random
import uuid
from typing import Tuple, List, Optional, Dict, Any

from determined.searcher.search_method import SearchMethod, Operation, SearchState, Create


@dataclasses.dataclass
class RandomSearchState(SearchState):
    created_trials: int
    pending_trials: int


class Unit(enum.Enum):
    Records = "records"
    Batches = "batches"
    Epochs = "epochs"
    Unitless = "unitless"
    Unspecified = "unspecified"


@dataclasses.dataclass
class MaxLength:
    unit: Unit
    units: int


@dataclasses.dataclass
class RandomConfig:
    max_length: MaxLength
    max_trials: int
    max_concurrent_trials: int


class RandomSearch(SearchMethod):
    def __init__(self, config: RandomConfig) -> None:
        super(self).__init__(RandomSearchState(0,0))
        # TODO should we create a RandomConfig class?
        self.config = config

    def initial_operations(self) -> Tuple[List[Operation], Optional[str]]:
        initial_trials = self.config.max_trials
        if self.config.max_concurrent_trials > 0:
            initial_trials = min(initial_trials, self.config.max_concurrent_trials)
        operations = []
        for i in range(initial_trials):
            create = Create(
                request_id=uuid.uuid4(),
                trial_seed=random.randint(0, 2**31),
                hparams=
            )
            operations.append(Create())