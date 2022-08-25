import json
import logging
import os
import pickle
import time
import uuid
from pathlib import Path
from typing import Any, Dict, List, Optional, Sequence, Tuple

from determined.common.api import bindings
from determined.common.api.bindings import v1SearcherEvent, v1TrialExitedEarlyExitedReason
from determined.experimental import client
from determined.searcher.search_method import ExitedReason, Operation, Progress, SearchMethod

EXPERIMENT_ID_FILE = "experiment_id"
STATE_FILE = "state"


class _ExperimentInactiveException(Exception):
    pass


class SearchRunner:
    def __init__(
        self,
        search_method: SearchMethod,
    ) -> None:
        self.search_method = search_method

    def _get_operations(self, event: bindings.v1SearcherEvent) -> List[Operation]:
        if event.initialOperations:
            logging.info("initial operations")
            operations = self.search_method.initial_operations()
        elif event.trialCreated:
            logging.info(f"trialCreated({event.trialCreated.requestId})")
            request_id = uuid.UUID(event.trialCreated.requestId)
            self.search_method.searcher_state.trials_created.add(request_id)
            self.search_method.searcher_state.trial_progress[request_id] = 0.0
            operations = self.search_method.on_trial_created(request_id)
        elif event.trialClosed:
            logging.info(f"trialClosed({event.trialClosed.requestId})")
            request_id = uuid.UUID(event.trialClosed.requestId)
            self.search_method.searcher_state.trials_closed.add(request_id)
            operations = self.search_method.on_trial_closed(request_id)
        elif event.trialExitedEarly:
            # duplicate exit accounting already performed by master
            logging.info(
                f"trialExitedEarly({event.trialExitedEarly.requestId},"
                f" {event.trialExitedEarly.exitedReason})"
            )
            if event.trialExitedEarly.exitedReason is None:
                raise RuntimeError("trialExitedEarly event is invalid without exitedReason")
            request_id = uuid.UUID(event.trialExitedEarly.requestId)
            if (
                event.trialExitedEarly.exitedReason
                == v1TrialExitedEarlyExitedReason.EXITED_REASON_INVALID_HP
            ):
                self.search_method.searcher_state.trial_progress.pop(request_id, None)
            elif (
                event.trialExitedEarly.exitedReason
                == v1TrialExitedEarlyExitedReason.EXITED_REASON_UNSPECIFIED
            ):
                self.search_method.searcher_state.failures.add(request_id)
            operations = self.search_method.on_trial_exited_early(
                request_id,
                exited_reason=ExitedReason._from_bindings(event.trialExitedEarly.exitedReason),
            )
        elif event.validationCompleted:
            # duplicate completion accounting already performed by master
            logging.info(
                f"validationCompleted({event.validationCompleted.requestId},"
                f" {event.validationCompleted.metric})"
            )
            request_id = uuid.UUID(event.validationCompleted.requestId)
            if event.validationCompleted.metric is None:
                raise RuntimeError("validationCompleted event is invalid without a metric")
            operations = self.search_method.on_validation_completed(
                request_id,
                event.validationCompleted.metric,
            )
        elif event.experimentInactive:
            logging.info(
                f"experiment {self.search_method.searcher_state.experiment_id} is "
                f"inactive; state={event.experimentInactive.experimentState}"
            )
            raise _ExperimentInactiveException()
        elif event.trialProgress:
            logging.debug(
                f"trialProgress({event.trialProgress.requestId}, "
                f"{event.trialProgress.partialUnits})"
            )
            request_id = uuid.UUID(event.trialProgress.requestId)
            self.search_method.searcher_state.trial_progress[request_id] = float(
                event.trialProgress.partialUnits
            )
            progress = self.search_method.progress()
            operations = [Progress(progress)]
        else:
            raise RuntimeError(f"Unsupported event {event}")
        return operations

    def run_experiment(
        self, experiment_id: int, prior_operations: Optional[List[Operation]]
    ) -> None:
        assert client._determined is not None
        session = client._determined._session

        experiment_is_active = True

        try:
            while experiment_is_active:
                time.sleep(5)
                events = self.get_events(session, experiment_id)
                if events is None:
                    continue
                logging.info(json.dumps([SearchRunner._searcher_event_as_dict(e) for e in events]))
                # the first event is an event we have already processed and told master about it
                # however, we may not have saved the state after that event if we crashed
                # after POSTing operations but before saving state
                last_event_id = self.search_method.searcher_state.last_event_id
                first_event = True
                for event in events:
                    if (
                        first_event
                        and last_event_id is not None
                        and last_event_id > event.id >= 0
                        and prior_operations is not None
                    ):
                        operations = prior_operations
                    else:
                        try:
                            operations = self._get_operations(event)
                        except _ExperimentInactiveException:
                            experiment_is_active = False
                            break

                        # save state
                        self.search_method.searcher_state.last_event_id = event.id
                        self.save_state(experiment_id, operations)
                    first_event = False

                    self.post_operations(session, experiment_id, event, operations)
        except KeyboardInterrupt:
            print("Runner interrupted")

    def post_operations(
        self,
        session: client.Session,
        experiment_id: int,
        event: bindings.v1SearcherEvent,
        operations: List[Operation],
    ) -> None:
        body = bindings.v1PostSearcherOperationsRequest(
            experimentId=self.search_method.searcher_state.experiment_id,
            searcherOperations=[op._to_searcher_operation() for op in operations],
            triggeredByEvent=event,
        )
        bindings.post_PostSearcherOperations(
            session,
            body=body,
            experimentId=experiment_id,
        )

    def get_events(
        self,
        session: client.Session,
        experiment_id: int,
    ) -> Optional[Sequence[bindings.v1SearcherEvent]]:
        events = bindings.get_GetSearcherEvents(session, experimentId=experiment_id)
        return events.searcherEvents

    def save_state(self, experiment_id: int, operations: List[Operation]) -> None:
        pass

    @staticmethod
    def _searcher_event_as_dict(event: v1SearcherEvent) -> dict:
        d = {}
        if event.trialExitedEarly:
            d["trialExitedEarly"] = event.trialExitedEarly.to_json()
        if event.validationCompleted:
            d["validationCompleted"] = event.validationCompleted.to_json()
        if event.trialProgress:
            d["trialProgress"] = event.trialProgress.to_json()
        if event.trialClosed:
            d["trialClosed"] = event.trialClosed.to_json()
        if event.trialCreated:
            d["trialCreated"] = event.trialCreated.to_json()
        if event.initialOperations:
            d["initialOperations"] = event.initialOperations.to_json()
        if event.experimentInactive:
            d["experimentInactive"] = event.experimentInactive.to_json()
        d["id"] = event.id
        return d


class LocalSearchRunner(SearchRunner):
    def __init__(
        self,
        search_method: SearchMethod,
        searcher_dir: Optional[Path] = None,
    ):
        super().__init__(search_method)
        self.state_path = None

        self.searcher_dir = searcher_dir or Path.cwd()
        if not self.searcher_dir.exists():
            self.searcher_dir.mkdir(parents=True)
        elif not self.searcher_dir.is_dir():
            raise FileExistsError(
                f"searcher_dir={self.searcher_dir} already exists and is not a directory"
            )

    def run(
        self,
        exp_config: Dict[str, Any],
        context_dir: Optional[str] = None,
    ) -> int:
        """
        Run custom search without an experiment id
        """
        logging.info("LocalSearchRunner.run")

        if context_dir is None:
            context_dir = os.getcwd()
        experiment_id_file = self.searcher_dir.joinpath(EXPERIMENT_ID_FILE)
        operations: Optional[List[Operation]] = None
        if experiment_id_file.exists():
            with experiment_id_file.open("r") as f:
                experiment_id = int(f.read())
            logging.info(f"Resuming HP searcher for experiment {experiment_id}")
            # load searcher state and search method state
            _, operations = self.load_state(experiment_id)
        else:
            exp = client.create_experiment(exp_config, context_dir)
            with experiment_id_file.open("w") as f:
                f.write(str(exp.id))
            state_path = self._get_state_path(exp.id)
            state_path.mkdir(parents=True)
            logging.info(f"Starting HP searcher for experiment {exp.id}")
            self.search_method.searcher_state.experiment_id = exp.id
            self.search_method.searcher_state.last_event_id = None
            experiment_id = exp.id

        self.run_experiment(experiment_id, operations)
        return experiment_id

    def load_state(self, experiment_id: int) -> Tuple[int, List[Operation]]:
        experiment_searcher_dir = self._get_state_path(experiment_id)
        with experiment_searcher_dir.joinpath("event_id").open("r") as event_id_file:
            last_event_id = int(event_id_file.read())
        state_path = experiment_searcher_dir.joinpath(f"event_{last_event_id}")
        loaded_experiment_id = self.search_method.load(state_path)
        assert experiment_id == loaded_experiment_id, (
            f"Experiment id mismatch. Expected {experiment_id}." f" Found {loaded_experiment_id}"
        )
        with state_path.joinpath("ops").open("rb") as f:
            operations = pickle.load(f)
        return loaded_experiment_id, operations

    def save_state(self, experiment_id: int, operations: List[Operation]) -> None:
        experiment_searcher_dir = self._get_state_path(experiment_id)
        state_path = experiment_searcher_dir.joinpath(
            f"event_{self.search_method.searcher_state.last_event_id}"
        )
        state_path.mkdir(parents=True)
        self.search_method.save(
            state_path,
            experiment_id=experiment_id,
        )
        with state_path.joinpath("ops").open("wb") as ops_file:
            pickle.dump(operations, ops_file)

        # commit
        event_id_path = experiment_searcher_dir.joinpath("event_id")
        event_id_new_path = experiment_searcher_dir.joinpath("event_id_new")
        with event_id_new_path.open("w") as f:
            f.write(str(self.search_method.searcher_state.last_event_id))
        os.replace(event_id_new_path, event_id_path)

    def _get_state_path(self, experiment_id: int) -> Path:
        return self.searcher_dir.joinpath(f"exp_{experiment_id}")
