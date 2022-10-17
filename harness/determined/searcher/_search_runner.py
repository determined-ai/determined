import json
import logging
import os
import pickle
import time
import uuid
from pathlib import Path
from typing import Any, Dict, List, Optional, Sequence, Tuple, Union

from determined import searcher
from determined.common.api import bindings
from determined.experimental import client

EXPERIMENT_ID_FILE = "experiment_id.txt"
logger = logging.getLogger("determined.searcher")


class _ExperimentInactiveException(Exception):
    def __init__(self, exp_state: bindings.determinedexperimentv1State):
        self.exp_state = exp_state


class SearchRunner:
    def __init__(
        self,
        search_method: searcher.SearchMethod,
    ) -> None:
        self.search_method = search_method
        self.state = searcher.SearcherState()

    def _get_operations(self, event: bindings.v1SearcherEvent) -> List[searcher.Operation]:
        if event.initialOperations:
            logger.info("initial operations")
            operations = self.search_method.initial_operations(self.state)
        elif event.trialCreated:
            logger.info(f"trialCreated({event.trialCreated.requestId})")
            request_id = uuid.UUID(event.trialCreated.requestId)
            self.state.trials_created.add(request_id)
            self.state.trial_progress[request_id] = 0.0
            operations = self.search_method.on_trial_created(self.state, request_id)
        elif event.trialClosed:
            logger.info(f"trialClosed({event.trialClosed.requestId})")
            request_id = uuid.UUID(event.trialClosed.requestId)
            self.state.trials_closed.add(request_id)
            operations = self.search_method.on_trial_closed(self.state, request_id)

            # add progress operation
            progress = self.search_method.progress(self.state)
            operations.append(searcher.Progress(progress))
        elif event.trialExitedEarly:
            # duplicate exit accounting already performed by master
            logger.info(
                f"trialExitedEarly({event.trialExitedEarly.requestId},"
                f" {event.trialExitedEarly.exitedReason})"
            )
            if event.trialExitedEarly.exitedReason is None:
                raise RuntimeError("trialExitedEarly event is invalid without exitedReason")
            request_id = uuid.UUID(event.trialExitedEarly.requestId)
            if (
                event.trialExitedEarly.exitedReason
                == bindings.v1TrialExitedEarlyExitedReason.EXITED_REASON_INVALID_HP
            ):
                self.state.trial_progress.pop(request_id, None)
            elif (
                event.trialExitedEarly.exitedReason
                == bindings.v1TrialExitedEarlyExitedReason.EXITED_REASON_UNSPECIFIED
            ):
                self.state.failures.add(request_id)
            operations = self.search_method.on_trial_exited_early(
                self.state,
                request_id,
                exited_reason=searcher.ExitedReason._from_bindings(
                    event.trialExitedEarly.exitedReason
                ),
            )
            # add progress operation
            progress = self.search_method.progress(self.state)
            operations.append(searcher.Progress(progress))
        elif event.validationCompleted:
            # duplicate completion accounting already performed by master
            logger.info(
                f"validationCompleted({event.validationCompleted.requestId},"
                f" {event.validationCompleted.metric})"
            )
            request_id = uuid.UUID(event.validationCompleted.requestId)
            if event.validationCompleted.metric is None:
                raise RuntimeError("validationCompleted event is invalid without a metric")

            operations = self.search_method.on_validation_completed(
                self.state,
                request_id,
                event.validationCompleted.metric,
                int(event.validationCompleted.validateAfterLength),
            )
            # add progress operation
            progress = self.search_method.progress(self.state)
            operations.append(searcher.Progress(progress))
        elif event.experimentInactive:
            logger.info(
                f"experiment {self.state.experiment_id} is "
                f"inactive; state={event.experimentInactive.experimentState}"
            )

            raise _ExperimentInactiveException(event.experimentInactive.experimentState)
        elif event.trialProgress:
            logger.debug(
                f"trialProgress({event.trialProgress.requestId}, "
                f"{event.trialProgress.partialUnits})"
            )
            request_id = uuid.UUID(event.trialProgress.requestId)
            self.state.trial_progress[request_id] = float(event.trialProgress.partialUnits)
            progress = self.search_method.progress(self.state)
            operations = [searcher.Progress(progress)]
        else:
            raise RuntimeError(f"Unsupported event {event}")
        return operations

    def run_experiment(
        self,
        experiment_id: int,
        session: client.Session,
        prior_operations: Optional[List[searcher.Operation]],
    ) -> None:
        experiment_is_active = True

        try:
            while experiment_is_active:
                time.sleep(
                    1
                )  # we don't want to call long polling API more often than every second.
                events = self.get_events(session, experiment_id)
                if not events:
                    continue
                logger.info(json.dumps([SearchRunner._searcher_event_as_dict(e) for e in events]))
                # the first event is an event we have already processed and told master about it
                # however, we may not have saved the state after that event if we crashed
                # after POSTing operations but before saving state
                last_event_id = self.state.last_event_id
                first_event = True
                for event in events:
                    if (
                        first_event
                        and last_event_id != 0
                        and last_event_id >= event.id >= 0
                        and prior_operations is not None
                    ):
                        logger.info(f"Resubmitting operations for event.id={event.id}")
                        operations = prior_operations
                    else:
                        if event.experimentInactive:
                            logger.info(
                                f"experiment {self.state.experiment_id} is "
                                f"inactive; state={event.experimentInactive.experimentState}"
                            )
                            if (
                                event.experimentInactive.experimentState
                                == bindings.determinedexperimentv1State.STATE_COMPLETED
                            ):
                                self.state.experiment_completed = True

                            if (
                                event.experimentInactive.experimentState
                                == bindings.determinedexperimentv1State.STATE_PAUSED
                            ):
                                self._show_experiment_paused_msg()
                            else:
                                experiment_is_active = False
                            break

                        operations = self._get_operations(event)

                        # save state
                        self.state.last_event_id = event.id
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
        operations: List[searcher.Operation],
    ) -> None:
        body = bindings.v1PostSearcherOperationsRequest(
            experimentId=self.state.experiment_id,
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
        # API is implemented with long polling.
        events = bindings.get_GetSearcherEvents(session, experimentId=experiment_id)
        return events.searcherEvents

    def save_state(self, experiment_id: int, operations: List[searcher.Operation]) -> None:
        pass

    def _show_experiment_paused_msg(self) -> None:
        pass

    @staticmethod
    def _searcher_event_as_dict(event: bindings.v1SearcherEvent) -> dict:
        return {k: v for k, v in event.to_json().items() if v is not None}


class LocalSearchRunner(SearchRunner):
    """
    ``LocalSearchRunner`` performs a search for optimal hyperparameter values,
    applying the provided ``SearchMethod``. It is executed locally and interacts
    with a Determined cluster where it starts a multi-trial experiment. It then
    reacts to event notifications coming from the running experiments by forwarding
    them to event handler methods in your ``SearchMethod`` implementation and sending
    the returned operations back to the experiment.
    """

    def __init__(
        self,
        search_method: searcher.SearchMethod,
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
        exp_config: Union[Dict[str, Any], str],
        model_dir: Optional[str] = None,
    ) -> int:
        """
        Run custom search.

        Args:
            exp_config (dictionary, string): experiment config filename (.yaml) or a dict.
            model_dir (string): directory containing model definition.
        """
        logger.info("LocalSearchRunner.run")

        if model_dir is None:
            model_dir = os.getcwd()
        experiment_id_file = self.searcher_dir.joinpath(EXPERIMENT_ID_FILE)
        operations: Optional[List[searcher.Operation]] = None
        if experiment_id_file.exists():
            with experiment_id_file.open("r") as f:
                experiment_id = int(f.read())
            logger.info(f"Resuming HP searcher for experiment {experiment_id}")
            # load searcher state and search method state
            _, operations = self.load_state(experiment_id)
        else:
            exp = client.create_experiment(exp_config, model_dir)
            with experiment_id_file.open("w") as f:
                f.write(str(exp.id))
            state_path = self._get_state_path(exp.id)
            state_path.mkdir(parents=True)
            logger.info(f"Starting HP searcher for experiment {exp.id}")
            self.state.experiment_id = exp.id
            self.state.last_event_id = 0
            self.save_state(exp.id, [])
            experiment_id = exp.id

        # make sure client is initialized
        client._require_singleton(lambda: None)()
        assert client._determined is not None
        session = client._determined._session
        self.run_experiment(experiment_id, session, operations)
        return experiment_id

    def load_state(self, experiment_id: int) -> Tuple[int, List[searcher.Operation]]:
        experiment_searcher_dir = self._get_state_path(experiment_id)
        with experiment_searcher_dir.joinpath("event_id").open("r") as event_id_file:
            last_event_id = int(event_id_file.read())
        state_path = experiment_searcher_dir.joinpath(f"event_{last_event_id}")
        self.state, loaded_experiment_id = self.search_method.load(state_path)
        assert experiment_id == loaded_experiment_id, (
            f"Experiment id mismatch. Expected {experiment_id}." f" Found {loaded_experiment_id}"
        )
        with state_path.joinpath("ops").open("rb") as f:
            operations = pickle.load(f)
        return loaded_experiment_id, operations

    def save_state(self, experiment_id: int, operations: List[searcher.Operation]) -> None:
        experiment_searcher_dir = self._get_state_path(experiment_id)
        state_path = experiment_searcher_dir.joinpath(f"event_{self.state.last_event_id}")

        if not state_path.exists():
            state_path.mkdir(parents=True)

        self.search_method.save(
            self.state,
            state_path,
            experiment_id=experiment_id,
        )
        with state_path.joinpath("ops").open("wb") as ops_file:
            pickle.dump(operations, ops_file)

        # commit
        event_id_path = experiment_searcher_dir.joinpath("event_id")
        event_id_new_path = experiment_searcher_dir.joinpath("event_id_new")
        with event_id_new_path.open("w") as f:
            f.write(str(self.state.last_event_id))
        os.replace(event_id_new_path, event_id_path)

    def _get_state_path(self, experiment_id: int) -> Path:
        return self.searcher_dir.joinpath(f"exp_{experiment_id}")

    def _show_experiment_paused_msg(self) -> None:
        logger.warning(
            f"Experiment {self.state.experiment_id} "
            f"has been paused. If you leave searcher process running, your search method"
            f" will automatically resume when the experiment becomes active again. "
            f"Otherwise, you can terminate this process and restart it "
            f"manually to continue the search."
        )
