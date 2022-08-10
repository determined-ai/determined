import dataclasses
import json
import uuid
from abc import abstractmethod
from enum import Enum
from typing import Any, Dict, List, Optional, Set, Type

from determined.common.api import bindings
from determined.common.experimental import Checkpoint


@dataclasses.dataclass
class SearcherState:
    failures: Set[uuid.UUID]
    trial_progress: Dict[uuid.UUID, float]
    trials_closed: Set[uuid.UUID]
    trials_created: Set[uuid.UUID]
    last_event_id: Optional[int] = None
    experiment_completed: bool = False

    def __init__(self) -> None:
        self.failures = set()
        self.trial_progress = {}
        self.trials_closed = set()
        self.trials_created = set()

    def to_dict(self) -> Dict[str, Any]:
        return {
            "failures": [str(f) for f in self.failures],
            "trialProgress": {str(k): v for k, v in self.trial_progress.items()},
            "trialsClosed": [str(t) for t in self.trials_closed],
            "trialsCreated": [str(t) for t in self.trials_created],
            "lastEventId": self.last_event_id,
            "experimentCompleted": self.experiment_completed,
        }

    def from_dict(self, d: Dict[str, Any]) -> None:
        self.failures = set(uuid.UUID(f) for f in d.get("failures", []))
        self.trial_progress = {uuid.UUID(k): v for k, v in d.get("trialProgress", dict()).items()}
        self.trials_closed =  set(uuid.UUID(t) for t in d.get("trialsClosed", []))
        self.trials_created = set(uuid.UUID(t) for t in d.get("trialsCreated", []))
        self.last_event_id = d.get("lastEventId")
        self.experiment_completed = d.get("experimentCompleted", False)


class ExitedReason(Enum):
    ERRORED = "ERRORED"
    USER_CANCELED = "USER_CANCELED"
    INVALID_HP = "INVALID_HP"
    INIT_INVALID_HP = "INIT_INVALID_HP"

    @classmethod
    def _from_bindings(
        cls, bindings_exited_reason: bindings.v1TrialExitedEarlyExitedReason
    ) -> "ExitedReason":
        if (
            bindings_exited_reason
            == bindings.v1TrialExitedEarlyExitedReason.EXITED_REASON_INVALID_HP
        ):
            return cls.INVALID_HP
        if (
            bindings_exited_reason
            == bindings.v1TrialExitedEarlyExitedReason.EXITED_REASON_USER_REQUESTED_STOP
        ):
            return cls.USER_CANCELED
        if (
            bindings_exited_reason
            == bindings.v1TrialExitedEarlyExitedReason.EXITED_REASON_INIT_INVALID_HP
        ):
            return cls.INIT_INVALID_HP
        raise RuntimeError(f"Invalid exited reason: {bindings_exited_reason}")


class Operation:
    @abstractmethod
    def _to_searcher_operation(self) -> bindings.v1SearcherOperation:
        pass


class ValidateAfter(Operation):
    def __init__(self, request_id: uuid.UUID, length: int) -> None:
        super().__init__()
        self.request_id = request_id
        self.length = length

    def _to_searcher_operation(self) -> bindings.v1SearcherOperation:
        return bindings.v1SearcherOperation(
            validateAfter=bindings.v1ValidateAfterOperation(
                requestId=str(self.request_id),
                length=str(self.length),
            )
        )


class Close(Operation):
    def __init__(self, request_id: uuid.UUID):
        super().__init__()
        self.request_id = request_id

    def _to_searcher_operation(self) -> bindings.v1SearcherOperation:
        return bindings.v1SearcherOperation(
            closeTrial=bindings.v1CloseTrialOperation(requestId=str(self.request_id))
        )


class Progress(Operation):
    def __init__(self, progress: float):
        super().__init__()
        self.progress = progress

    def _to_searcher_operation(self) -> bindings.v1SearcherOperation:
        return bindings.v1SearcherOperation(
            searcherProgress=bindings.v1SearcherProgressOperation(progress=self.progress)
        )


class Shutdown(Operation):
    def __init__(self) -> None:
        super().__init__()

    def _to_searcher_operation(self) -> bindings.v1SearcherOperation:
        return bindings.v1SearcherOperation(shutdown=bindings.v1ShutdownOperation())


class Create(Operation):
    def __init__(
        self,
        request_id: uuid.UUID,
        hparams: Dict[str, Any],
        checkpoint: Optional[Checkpoint],
    ) -> None:
        super().__init__()
        self.request_id = request_id
        self.hparams = json.dumps(hparams)
        self.checkpoint = checkpoint

    def _to_searcher_operation(self) -> bindings.v1SearcherOperation:
        return bindings.v1SearcherOperation(
            createTrial=bindings.v1CreateTrialOperation(
                hyperparams=self.hparams, requestId=str(self.request_id)
            )
        )


class SearchMethod:
    def __init__(self) -> None:
        self._searcher_state = SearcherState()

    @property
    def searcher_state(self) -> SearcherState:
        return self._searcher_state

    @abstractmethod
    def initial_operations(self) -> List[Operation]:
        """
        initial_operations returns a set of initial operations that the searcher
        would like to take.
        """
        pass

    @abstractmethod
    def on_trial_created(self, request_id: uuid.UUID) -> List[Operation]:
        """
        on_trial_created informs the searcher that a trial has been created
        as a result of Create operation.
        """
        pass

    @abstractmethod
    def on_validation_completed(self, request_id: uuid.UUID, metric: float) -> List[Operation]:
        """
        on_validation_completed informs the searcher that the validation workload
        initiated by the same searcher has completed. It returns any new operations
        as a result of this workload completing.
        """
        pass

    @abstractmethod
    def on_trial_closed(self, request_id: uuid.UUID) -> List[Operation]:
        """
        trialClosed informs the searcher that the trial has been closed as a result of a Close
        operation.
        """
        pass

    @abstractmethod
    def progress(self) -> float:
        """
        progress returns experiment progress as a float between 0 and 1.
        """
        pass

    @abstractmethod
    def on_trial_exited_early(
        self,
        request_id: uuid.UUID,
        exited_reason: ExitedReason,
    ) -> List[Operation]:
        """
        on_trial_exited_early informs the searcher that the trial has exited earlier than expected.
        """
        pass

    def save_checkpoint(self, event_id: int) -> None:
        """
        This is optionally implemented to save a checkpoint indexed by event id.
        It will be called by the ``SearchRunner`` after receiving operations
        from the ``SearchMethod``
        """
        pass

    def load_checkpoint(self, event_id: int) -> None:
        """
        This is optionally implemented to load a checkpoint indexed by event id.
        It will be called by the ``SearchRunner`` before processing new searcher events
        from the master.
        """
        pass
