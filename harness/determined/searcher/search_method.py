import dataclasses
import uuid
from abc import abstractmethod
from typing import Optional, Tuple, List, Dict, Any

from determined.common.experimental import Checkpoint


@dataclasses.dataclass
class SearchState:
    checkpoint_id: Optional[str]


class Operation:
    def __init__(self) -> None:
        pass


class ValidateAfter(Operation):
    pass


class Close(Operation):
    pass


class Shutdown(Operation):
    pass


class Create(Operation):

    def __init__(
        self,
        request_id: uuid.UUID,
        trial_seed: int,
        hparams: Dict[str, Any],
        checkpoint: Optional[Checkpoint],
        workload_sequencer_type: str,
    ) -> None:
        self._request_id = request_id
        self._trial_seed = trial_seed
        self._hparams = hparams
        self._checkpoint = checkpoint
        self._workload_sequencer_type = workload_sequencer_type


class SearchMethod:
    def __init__(self, search_state: SearchState) -> None:
        self._search_state = SearchState(None)

    @property
    def searcher_state(self) -> SearchState:
        return self._search_state

    @abstractmethod
    def initial_operations(self) -> Tuple[List[Operation], Optional[str]]:
        pass




