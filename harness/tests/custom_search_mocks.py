import abc
import logging
from pathlib import Path
from typing import Any, Dict, Iterable, List, Optional, Sequence, Union
from unittest.mock import Mock

from determined import searcher
from determined.common.api import bindings
from determined.experimental import client


class MockMaster(metaclass=abc.ABCMeta):
    @abc.abstractmethod
    def handle_post_operations(
        self, event: bindings.v1SearcherEvent, operations: List[searcher.Operation]
    ) -> None:
        pass

    @abc.abstractmethod
    def handle_get_events(self) -> Optional[Sequence[bindings.v1SearcherEvent]]:
        return []

    @abc.abstractmethod
    def add_event(self, event_obj: bindings.v1SearcherEvent) -> None:
        pass


class SimulateMaster(MockMaster):
    def __init__(self, metric: Union[float, Dict[str, Any]]) -> None:
        # store event and
        self.events_queue: List[bindings.v1SearcherEvent] = []
        self.events_count = 0
        self.metric = metric
        self.overall_progress = 0.0

    def handle_post_operations(
        self, event: bindings.v1SearcherEvent, operations: List[searcher.Operation]
    ) -> None:
        self._remove_upto(event)
        self._process_operations(operations)

    def _remove_upto(self, event: bindings.v1SearcherEvent) -> None:
        for i, e in enumerate(self.events_queue):
            if e.id == event.id:
                self.events_queue = self.events_queue[i + 1 :]
                return

        raise RuntimeError(f"event not found in events queue: {event}")

    def _process_operations(self, operations: List[searcher.Operation]) -> None:
        for op in operations:
            # validate_after returns two events.
            self._append_events_for_op(op)

    def add_event(self, event_obj: bindings.v1SearcherEvent) -> None:
        self.events_queue.append(event_obj)

    def handle_get_events(self) -> Optional[Sequence[bindings.v1SearcherEvent]]:
        return self.events_queue

    def _append_events_for_op(self, op: searcher.Operation) -> None:
        if type(op) == searcher.ValidateAfter:
            validation_completed = bindings.v1ValidationCompleted(
                requestId=str(op.request_id),
                metric=self.metric,
                validateAfterLength=str(op.length),
            )
            self.events_count += 1
            event = bindings.v1SearcherEvent(
                id=self.events_count, validationCompleted=validation_completed
            )
            self.events_queue.append(event)

            trial_progress = bindings.v1TrialProgress(
                requestId=str(op.request_id), partialUnits=float(op.length)
            )
            self.events_count += 1
            event = bindings.v1SearcherEvent(id=self.events_count, trialProgress=trial_progress)
            self.events_queue.append(event)

        elif type(op) == searcher.Create:
            trial_created = bindings.v1TrialCreated(requestId=str(op.request_id))
            self.events_count += 1
            event = bindings.v1SearcherEvent(id=self.events_count, trialCreated=trial_created)
            self.events_queue.append(event)

        elif type(op) == searcher.Progress:  # no events
            self.overall_progress

        elif type(op) == searcher.Close:
            trial_closed = bindings.v1TrialClosed(requestId=str(op.request_id))
            self.events_count += 1
            event = bindings.v1SearcherEvent(id=self.events_count, trialClosed=trial_closed)
            self.events_queue.append(event)

        elif type(op) == searcher.Shutdown:
            exp_state = bindings.experimentv1State.COMPLETED
            exp_inactive = bindings.v1ExperimentInactive(experimentState=exp_state)
            self.events_count += 1
            event = bindings.v1SearcherEvent(id=self.events_count, experimentInactive=exp_inactive)
            self.events_queue.append(event)
        else:
            pass


class MockMasterSearchRunner(searcher.LocalSearchRunner):
    def __init__(
        self,
        search_method: searcher.SearchMethod,
        mock_master_object: MockMaster,
        searcher_dir: Optional[Path] = None,
    ):
        super(MockMasterSearchRunner, self).__init__(search_method, searcher_dir)
        self.mock_master_obj = mock_master_object
        initial_ops = bindings.v1InitialOperations()
        event_obj = bindings.v1SearcherEvent(id=1, initialOperations=initial_ops)
        self.mock_master_obj.add_event(event_obj)

    def post_operations(
        self,
        session: client.Session,
        experiment_id: int,
        event: bindings.v1SearcherEvent,
        operations: List[searcher.Operation],
    ) -> None:
        logging.info("MockMasterSearchRunner.post_operations")
        self.mock_master_obj.handle_post_operations(event, operations)

    def get_events(
        self,
        session: client.Session,
        experiment_id: int,
    ) -> Optional[Sequence[bindings.v1SearcherEvent]]:
        logging.info("MockMasterSearchRunner.get_events")
        return self.mock_master_obj.handle_get_events()

    def run(
        self,
        exp_config: Union[Dict[str, Any], str],
        context_dir: Optional[str] = None,
        includes: Optional[Iterable[Union[str, Path]]] = None,
    ) -> int:
        logging.info("MockMasterSearchRunner.run")
        experiment_id_file = self.searcher_dir.joinpath("experiment_id")
        exp_id = 4  # dummy exp
        with experiment_id_file.open("w") as f:
            f.write(str(exp_id))
        state_path = self._get_state_path(exp_id)
        state_path.mkdir(parents=True)
        logging.info(f"Starting HP searcher for mock experiment {exp_id}")
        self.state.experiment_id = exp_id
        self.state.last_event_id = 0
        super(MockMasterSearchRunner, self).save_state(exp_id, [])
        experiment_id = exp_id
        operations: Optional[List[searcher.Operation]] = None
        session: client.Session = Mock()
        super(MockMasterSearchRunner, self).run_experiment(
            experiment_id, session, operations, sleep_time=0.0
        )
        return exp_id

    def _get_state_path(self, experiment_id: int) -> Path:
        return self.searcher_dir.joinpath(f"exp_{experiment_id}")
