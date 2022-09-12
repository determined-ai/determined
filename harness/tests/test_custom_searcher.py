import logging
import tempfile
from pathlib import Path
from typing import Any, Dict, List, Optional, Sequence

import pytest
from numpy import float64

from determined.common.api import bindings
from determined.experimental import client
from determined.searcher.search_method import (
    Close,
    Create,
    Operation,
    Progress,
    SearchMethod,
    Shutdown,
    ValidateAfter,
)
from determined.searcher.search_runner import LocalSearchRunner
from .e2e_tests.tests.fixtures.custom_searcher.searchers import (
    ASHASearchMethod,
    RandomSearchMethod,
)

def test_run_random_searcher_exp_mock_master() -> None:
    config = conf.load_config(conf.fixtures_path("no_op/single.yaml"))
    config["searcher"] = {
        "name": "custom",
        "metric": "validation_error",
        "smaller_is_better": True,
        "unit": "batches",
    }
    config["name"] = "random"
    config["description"] = "custom searcher"

    max_trials = 5
    max_concurrent_trials = 2
    max_length = 500

    with tempfile.TemporaryDirectory() as searcher_dir:
        search_method = RandomSearchMethod(max_trials, max_concurrent_trials, max_length)
        mock_master_obj = SimulateMaster(validation_fn=SimulateMaster.constant_validation)
        search_runner = MockMasterSearchRunner(search_method, Path(searcher_dir), mock_master_obj)
        search_runner.run(config, context_dir=conf.fixtures_path("no_op"))

    assert search_method.created_trials == 5
    assert search_method.pending_trials == 0
    assert search_method.closed_trials == 5
    assert len(search_method.searcher_state.trials_created) == search_method.created_trials
    assert len(search_method.searcher_state.trials_closed) == search_method.closed_trials


def test_run_asha_batches_exp_mock_master(tmp_path: Path) -> None:
    config = conf.load_config(conf.fixtures_path("no_op/adaptive.yaml"))
    config["searcher"] = {
        "name": "custom",
        "metric": "validation_error",
        "smaller_is_better": True,
        "unit": "batches",
    }
    config["name"] = "asha"
    config["description"] = "custom searcher"

    max_length = 3000
    max_trials = 16
    num_rungs = 3
    divisor = 4

    search_method = ASHASearchMethod(max_length, max_trials, num_rungs, divisor)
    mock_master_obj = SimulateMaster(validation_fn=SimulateMaster.constant_validation)
    search_runner = MockMasterSearchRunner(search_method, tmp_path, mock_master_obj)
    search_runner.run(config, context_dir=conf.fixtures_path("no_op"))

    assert search_method.asha_search_state.pending_trials == 0
    assert search_method.asha_search_state.completed_trials == 16
    assert len(search_method.searcher_state.trials_closed) == len(
        search_method.asha_search_state.closed_trials
    )



class SimulateMaster:
    def __init__(self, validation_fn):
        self.events_queue = []  # store event and
        self.events_count = 0
        self.validation_fn = validation_fn
        self.overall_progress = 0.0
        return

    def handle_post_operations(self, event: bindings.v1SearcherEvent, operations: List[Operation]):
        self._remove_upto(event)
        self._process_operations(operations)

    def _remove_upto(self, event: bindings.v1SearcherEvent):
        for i, e in enumerate(self.events_queue):
            if e.id == event.id:
                self.events_queue = self.events_queue[i + 1 :]
                return

        pytest.raises(RuntimeError(f"event not found in events queue: {event}"))

    def _process_operations(self, operations: List[Operation]):
        for op in operations:
            self._append_events_for_op(op)  # validate_after returns two events.

    def handle_get_events(self) -> Optional[Sequence[bindings.v1SearcherEvent]]:
        return self.events_queue

    def _append_events_for_op(self, op: Operation):
        if type(op) == ValidateAfter:
            metric = (
                self.validation_fn()
            )  # is it useful to be able to use constant or random validation function?
            validation_completed = bindings.v1ValidationCompleted(
                requestId=str(op.request_id), metric=metric, validateAfterLength=str(op.length)
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

        if type(op) == Create:
            trial_created = bindings.v1TrialCreated(requestId=str(op.request_id))
            self.events_count += 1
            event = bindings.v1SearcherEvent(id=self.events_count, trialCreated=trial_created)
            self.events_queue.append(event)

        if type(op) == Progress:  # no events
            self.overall_progress

        if type(op) == Close:
            trial_closed = bindings.v1TrialClosed(requestId=str(op.request_id))
            self.events_count += 1
            event = bindings.v1SearcherEvent(id=self.events_count, trialClosed=trial_closed)
            self.events_queue.append(event)

        if type(op) == Shutdown:
            exp_state = bindings.determinedexperimentv1State.STATE_COMPLETED
            exp_inactive = bindings.v1ExperimentInactive(experimentState=exp_state)
            self.events_count += 1
            event = bindings.v1SearcherEvent(id=self.events_count, experimentInactive=exp_inactive)
            self.events_queue.append(event)

    def constant_validation() -> float64:
        return 1

    def random_validation() -> float64:
        import random

        return random.random()


class MockMasterSearchRunner(LocalSearchRunner):
    def __init__(
        self,
        search_method: SearchMethod,
        searcher_dir: Optional[Path] = None,
        mock_master_object: SimulateMaster = None,
    ):
        super(MockMasterSearchRunner, self).__init__(search_method, searcher_dir)
        if mock_master_object:
            self.mock_master_obj = mock_master_object
            initial_ops = bindings.v1InitialOperations()
            event_obj = bindings.v1SearcherEvent(id=1, initialOperations=initial_ops)
            mock_master_object.events_queue.append(event_obj)

    def post_operations(
        self,
        session: client.Session,
        experiment_id: int,
        event: bindings.v1SearcherEvent,
        operations: List[Operation],
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

    def run(self, exp_config: Dict[str, Any], context_dir: Optional[str]) -> int:
        logging.info("MockMasterSearchRunner.run")
        experiment_id_file = self.searcher_dir.joinpath("experiment_id")
        exp_id = "4"  # dummy exp
        with experiment_id_file.open("w") as f:
            f.write(str(exp_id))
        state_path = self._get_state_path(exp_id)
        state_path.mkdir(parents=True)
        logging.info(f"Starting HP searcher for mock experiment {exp_id}")
        self.search_method.searcher_state.experiment_id = exp_id
        self.search_method.searcher_state.last_event_id = 0
        super(MockMasterSearchRunner, self).save_state(exp_id, [])
        experiment_id = exp_id
        operations: Optional[List[Operation]] = None
        super(MockMasterSearchRunner, self).run_experiment(experiment_id, operations, None)

    def _get_state_path(self, experiment_id: int) -> Path:
        return self.searcher_dir.joinpath(f"exp_{experiment_id}")

