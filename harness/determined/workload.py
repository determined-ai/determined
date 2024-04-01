import abc
import enum
from typing import Any, Callable, Dict, Iterator, Optional, Tuple, Union, cast

from determined.common import check


class Workload:
    @enum.unique
    class Kind(enum.Enum):
        RUN_STEP = 1
        COMPUTE_VALIDATION_METRICS = 2
        CHECKPOINT_MODEL = 3

    def __init__(
        self,
        kind: Kind,
        e_id: int,
        t_id: int,
        s_id: int,
        num_batches: int,
        total_batches_processed: int,
    ) -> None:
        self.kind = kind
        self.experiment_id = e_id
        self.trial_id = t_id
        self.step_id = s_id
        self.num_batches = num_batches
        self.total_batches_processed = total_batches_processed

    def __eq__(self, other: object) -> bool:
        if type(self) is not type(other):
            return False

        return self.__dict__ == other.__dict__

    def __hash__(self) -> int:
        return hash((self.kind, self.experiment_id, self.trial_id, self.step_id))

    def __repr__(self) -> str:
        extra = f" ({self.num_batches} Batches)" if self.kind == self.Kind.RUN_STEP else ""
        return f"<{self.kind.name}{extra}: ({self.experiment_id},{self.trial_id},{self.step_id})>"

    def __json__(self) -> Dict[str, Any]:
        return self.__dict__

    @staticmethod
    def from_json(data: Dict[str, Any]) -> "Workload":
        check.check_in(data["kind"], Workload.Kind.__members__)
        return Workload(
            Workload.Kind[data["kind"]],
            data["experiment_id"],
            data["trial_id"],
            data["step_id"],
            data["num_batches"],
            data["total_batches_processed"],
        )


"""Metrics is the general structure of metrics used in response messages throughout the harness."""
Metrics = Dict[str, Any]


class InvalidHP:
    """Workload canceled because an InvalidHP was raised."""

    pass


"""Every Workload needs a Response, which is either a Metrics object or an InvalidHP."""
Response = Union[Metrics, InvalidHP]


"""
ResponseFunc is a closure for returning a response message from a lower layer to a higher layer.
Since all messages are synchronous, the response function must be called.
"""
ResponseFunc = Callable[[Response], None]


"""
Stream describes the main message passing interface between layers of the harness.  Higher layers
will yield workloads to lower layers, with closures to be called for the response.  Yielding a
response closure alongside the workload only works because the messaging paradigm in the harness is
synchronous.
"""
Stream = Iterator[Tuple[Workload, ResponseFunc]]


class Source(metaclass=abc.ABCMeta):
    """
    Source is a simple interface that most layers of the harness will implement. The only
    layer of the harness which should definitely not be a WorkloadSource is the final layer
    (probably a TrialController), since the final layer will only consume messages.

    Generally, a layer of the harness will be initialized with the Stream produced by the
    layer above it and without any idea of what layers come below it. This keeps a strong isolation
    between layers of the harness and improves the plugability and testability of each layer.
    """

    @abc.abstractmethod
    def __iter__(self) -> Stream:
        """
        Generate tuples of (workload, response closure) to pass to the next layer down.
        """
        pass


class WorkloadResponseInterceptor:
    """
    WorkloadResponseInterceptor is a class that can send some precanned workload messages, as the
    TrialWorkloadManager might send to the TrialController, but then intercept the responses and
    offer them via .result().

    This is basically syntactic sugar to make writing unit tests feel more declarative even when
    unit tests need to be written as WorkloadIterators (i.e. generator coroutines).

    Example usage:

        def make_workloads():
            interceptor = WorkloadResponseInterceptor()

            # Yield some workload message to the TrialController.
            yield from interceptor.send(my_workload)

            # Check that the result is appropriate.
            check.is_reasonable(interceptor.result())

            ...

        # Create a Trial to read this stream of workloads.
        controller = MyTrialController(..., make_workloads())

        # Run the workloads to completion.
        controller.run()
    """

    def __init__(self) -> None:
        self._response = None  # type: Optional[Response]

    def _respond(self, resp: Response) -> None:
        """Capture a response from the trial controller."""
        check.is_none(self._response, "_respond() was called twice by the TrialController")
        self._response = resp

    def send(self, workload: Workload) -> Stream:
        """Yield a workload with our _respond() function so we can intercept the response."""
        self._response = None
        yield workload, self._respond

    def result(self) -> Response:
        """Read the WorkloadResponse from the TrialController (only call once per send)."""
        check.is_not_none(self._response, "_respond() was not called by the TrialController.")
        out = self._response
        self._response = None
        return cast(Response, out)

    def metrics_result(self) -> Metrics:
        """Identical to result but disallow workload.InvalidHP responses."""
        check.is_not_none(self._response, "_respond() was not called by the TrialController.")
        check.is_instance(self._response, dict, "unexpected InvalidHP response.")
        return cast(Metrics, self._response)


def ignore_workload_response(*_: Any) -> None:
    return


def train_workload(
    step_id: int,
    exp_id: int = 1,
    trial_id: int = 1,
    num_batches: int = 1,
    total_batches_processed: int = 0,
) -> Workload:
    return Workload(
        Workload.Kind.RUN_STEP,
        exp_id,
        trial_id,
        step_id,
        num_batches,
        total_batches_processed,
    )


def validation_workload(
    step_id: int = 1,
    exp_id: int = 1,
    trial_id: int = 1,
    total_batches_processed: int = 0,
) -> Workload:
    return Workload(
        Workload.Kind.COMPUTE_VALIDATION_METRICS,
        exp_id,
        trial_id,
        step_id,
        0,
        total_batches_processed,
    )


def checkpoint_workload(
    step_id: int = 1, exp_id: int = 1, trial_id: int = 1, total_batches_processed: int = 0
) -> Workload:
    return Workload(
        Workload.Kind.CHECKPOINT_MODEL,
        exp_id,
        trial_id,
        step_id,
        0,
        total_batches_processed,
    )
