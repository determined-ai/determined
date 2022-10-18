import dataclasses
import json
import pathlib
import uuid
from abc import abstractmethod
from enum import Enum
from typing import Any, Dict, List, Optional, Set, Tuple

from determined.common import experimental
from determined.common.api import bindings

STATE_FILE = "state"


@dataclasses.dataclass
class SearcherState:
    """
    Mutable Searcher state.

    Search runners maintain this state that can be used by a ``SearchMethod``
    to inform event handling. In other words, this state can be taken into account
    when deciding which operations to return from your event handler. Do not
    modify ``SearcherState`` in your ``SearchMethod``. If your hyperparameter
    tuning algorithm needs additional state variables, add those variable to your
    ``SearchMethod`` implementation.

    Attributes:
        failures: number of failed trials
        trial_progress: progress of each trial as a number between 0.0 and 1.0
        trials_closed: set of completed trials
        trials_created: set of created trials
    """

    failures: Set[uuid.UUID]
    trial_progress: Dict[uuid.UUID, float]
    trials_closed: Set[uuid.UUID]
    trials_created: Set[uuid.UUID]
    last_event_id: int = 0
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
            "experimentId": self.experiment_id,
            "experimentCompleted": self.experiment_completed,
        }

    def from_dict(self, d: Dict[str, Any]) -> None:
        self.failures = {uuid.UUID(f) for f in d.get("failures", [])}
        self.trial_progress = {uuid.UUID(k): v for k, v in d.get("trialProgress", {}).items()}
        self.trials_closed = {uuid.UUID(t) for t in d.get("trialsClosed", [])}
        self.trials_created = {uuid.UUID(t) for t in d.get("trialsCreated", [])}
        self.last_event_id = d.get("lastEventId", 0)
        self.experiment_id = d.get("experimentId")
        self.experiment_completed = d.get("experimentCompleted", False)


class ExitedReason(Enum):
    ERRORED = "ERRORED"
    USER_CANCELED = "USER_CANCELED"
    INVALID_HP = "INVALID_HP"

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
            == bindings.v1TrialExitedEarlyExitedReason.EXITED_REASON_UNSPECIFIED
        ):
            return cls.ERRORED
        raise RuntimeError(f"Invalid exited reason: {bindings_exited_reason}")


class Operation:
    """
    Abstract base class for all Operations
    """

    @abstractmethod
    def _to_searcher_operation(self) -> bindings.v1SearcherOperation:
        pass


class ValidateAfter(Operation):
    """
    Operation signaling the trial to train until its total units trained
    equals the specified length, where the units (batches, epochs, etc.)
    are specified in the searcher section of the experiment configuration
    """

    def __init__(self, request_id: uuid.UUID, length: int) -> None:
        super().__init__()
        self.request_id = request_id
        self.length = length

    def _to_searcher_operation(self) -> bindings.v1SearcherOperation:
        return bindings.v1SearcherOperation(
            trialOperation=bindings.v1TrialOperation(
                validateAfter=bindings.v1ValidateAfterOperation(
                    requestId=str(self.request_id), length=str(self.length)
                ),
            )
        )


class Close(Operation):
    """
    Operation closing the specified trial
    """

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
            setSearcherProgress=bindings.v1SetSearcherProgressOperation(progress=self.progress)
        )


class Shutdown(Operation):
    """
    Operation shutting the experiment down
    """

    def __init__(self) -> None:
        super().__init__()

    def _to_searcher_operation(self) -> bindings.v1SearcherOperation:
        return bindings.v1SearcherOperation(shutDown=bindings.v1ShutDownOperation())


class Create(Operation):
    """
    Operation creating a trial with a specified combination of hyperparameter values
    """

    def __init__(
        self,
        request_id: uuid.UUID,
        hparams: Dict[str, Any],
        checkpoint: Optional[experimental.Checkpoint],
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
    """
    The implementation of a custom hyperparameter tuning algorithm.

    To implement your specific hyperparameter tuning approach, subclass ``SearchMethod``
    overriding the event handler methods. Each event handler, except ``progress`` returns a list of
    operations (``List[Operation]``) that will be submitted to master for processing.

    .. note::

        Do not modify ``searcher_state`` passed into event handlers.
    """

    @abstractmethod
    def initial_operations(self, searcher_state: SearcherState) -> List[Operation]:
        """
        Returns a set of initial operations that the searcher will perform.

        Currently, we support the following operations:

        - Create - starts a new trial with a unique trial id and a set of hyperparameter
          values,
        - ValidateAfter - sets number of steps (i.e., batches or epochs) after which a validation
          is run for a trial with a given id,
        - Close - closes a trial with a given id,
        - Shutdown - closes the experiment.
        """
        pass

    @abstractmethod
    def on_trial_created(
        self, searcher_state: SearcherState, request_id: uuid.UUID
    ) -> List[Operation]:
        """
        Informs the searcher that a trial has been created
        as a result of Create operation.
        """
        pass

    @abstractmethod
    def on_validation_completed(
        self, searcher_state: SearcherState, request_id: uuid.UUID, metric: float, train_length: int
    ) -> List[Operation]:
        """
        Informs the searcher that the validation workload
        initiated by the same searcher has completed after training for ``train_length`` units.
        It returns any new operations as a result of this workload completing.
        """
        pass

    @abstractmethod
    def on_trial_closed(
        self, searcher_state: SearcherState, request_id: uuid.UUID
    ) -> List[Operation]:
        """
        Informs the searcher that a trial has been closed as a result of a Close
        operation.
        """
        pass

    @abstractmethod
    def progress(self, searcher_state: SearcherState) -> float:
        """
        Returns experiment progress as a float between 0 and 1.
        """
        pass

    @abstractmethod
    def on_trial_exited_early(
        self,
        searcher_state: SearcherState,
        request_id: uuid.UUID,
        exited_reason: ExitedReason,
    ) -> List[Operation]:
        """
        Informs the searcher that a trial has exited earlier than expected.
        """
        pass

    def save(
        self, searcher_state: SearcherState, path: pathlib.Path, *, experiment_id: int
    ) -> None:
        """
        Saves the searcher state and the search method state.
        It will be called by the ``SearchRunner`` after receiving operations
        from the ``SearchMethod``
        """
        searcher_state_file = path.joinpath(STATE_FILE)
        d = searcher_state.to_dict()
        d["experimentId"] = experiment_id
        with searcher_state_file.open("w") as f:
            json.dump(d, f)

        self.save_method_state(path)

    def save_method_state(self, path: pathlib.Path) -> None:
        """
        Saves method-specific state
        """
        pass

    def load(self, path: pathlib.Path) -> Tuple[SearcherState, int]:
        """
        Loads searcher state and method-specific state.
        """
        searcher_state_file = path.joinpath(STATE_FILE)
        with searcher_state_file.open("r") as f:
            state_dict = json.load(f)
            searcher_state = SearcherState()
            searcher_state.from_dict(state_dict)
            experiment_id = state_dict["experimentId"]  # type: int

        self.load_method_state(path)
        return searcher_state, experiment_id

    def load_method_state(
        self,
        path: pathlib.Path,
    ) -> None:
        """
        Loads method-specific search state.
        """
        pass
