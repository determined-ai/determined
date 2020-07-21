import abc
from enum import Enum, unique
from typing import Any, Callable, Dict, Iterator, List, Optional, Tuple, Union, cast

from determined_common import check
from determined_common.types import ExperimentID, StepID, TrialID


class Workload:
    @unique
    class Kind(Enum):
        RUN_STEP = 1
        COMPUTE_VALIDATION_METRICS = 2
        CHECKPOINT_MODEL = 3
        TERMINATE = 4

    def __init__(
        self,
        kind: Kind,
        e_id: ExperimentID,
        t_id: TrialID,
        s_id: StepID,
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
    def from_json(dict: Dict[str, Any]) -> "Workload":
        check.check_in(dict["kind"], Workload.Kind.__members__)
        return Workload(
            Workload.Kind[dict["kind"]],
            dict["experiment_id"],
            dict["trial_id"],
            dict["step_id"],
            dict["num_batches"],
            dict["total_batches_processed"],
        )


"""Metrics is the general structure of metrics used in response messages throughout the harness."""
Metrics = Dict[str, Any]


class Skipped:
    """Skipped is used in place of Metrics when a workload is ignored by a lower layer."""

    pass


"""Every Workload needs a Response, which is either a Metrics object or a SkippedWorkload."""
Response = Union[Metrics, Skipped]


"""
ResponseFunc is a closure for returning a response message from a lower layer to a higher layer.
Since all messages are synchronous, the response function must be called.
"""
ResponseFunc = Callable[[Response], None]


"""
Args is auxiliary information relevant to a workload which does not come from the master, such as
the path to a checkpoint directory for a trial to save to.
"""
Args = List[Any]


"""
Stream describes the main message passing interface between layers of the harness.  Higher layers
will yield workloads to lower layers, with closures to be called for the response.  Yielding a
response closure alongside the workload only works because the messaging paradigm in the harness is
synchronous.
"""
Stream = Iterator[Tuple[Workload, Args, ResponseFunc]]


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
        Generate tuples of (workload, workload args, response closure) to pass to the next layer
        down.
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
            yield from interceptor.send(my_workload, my_workload_args)

            # Check that the result is appropriate.
            check.is_reasonable(interceptor.result())

            ...

            # Close the TrialController.
            yield from interceptor.send(terminate_workload, [])

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

    def send(self, workload: Workload, workload_args: Args) -> Stream:
        """Yield a workload with our _respond() function so we can intercept the response."""
        self._response = None
        yield workload, workload_args, self._respond

    def result(self) -> Response:
        """Read the WorkloadResponse from the TrialController (only call once per send)."""
        check.is_not_none(self._response, "_respond() was not called by the TrialController.")
        out = self._response
        self._response = None
        return cast(Response, out)

    def metrics_result(self) -> Metrics:
        """Identical to result but disallow workload.Skipped responses."""
        check.is_not_none(self._response, "_respond() was not called by the TrialController.")
        check.is_instance(self._response, dict, "unexpected SkippedWorkload response.")
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
        ExperimentID(exp_id),
        TrialID(trial_id),
        StepID(step_id),
        num_batches,
        total_batches_processed,
    )


def validation_workload(
    step_id: int = 1, exp_id: int = 1, trial_id: int = 1, total_batches_processed: int = 0,
) -> Workload:
    return Workload(
        Workload.Kind.COMPUTE_VALIDATION_METRICS,
        ExperimentID(exp_id),
        TrialID(trial_id),
        StepID(step_id),
        0,
        total_batches_processed,
    )


def checkpoint_workload(
    step_id: int = 1, exp_id: int = 1, trial_id: int = 1, total_batches_processed: int = 0
) -> Workload:
    return Workload(
        Workload.Kind.CHECKPOINT_MODEL,
        ExperimentID(exp_id),
        TrialID(trial_id),
        StepID(step_id),
        0,
        total_batches_processed,
    )


def terminate_workload(step_id: int = 1, exp_id: int = 1, trial_id: int = 1) -> Workload:
    return Workload(
        Workload.Kind.TERMINATE, ExperimentID(exp_id), TrialID(trial_id), StepID(step_id), 0, 0,
    )
