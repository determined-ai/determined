import json
import logging
import os
import time
import uuid
from pathlib import Path
from typing import Any, Dict, Optional

from determined.common.api import bindings
from determined.common.api.bindings import v1SearcherEvent, v1TrialExitedEarlyExitedReason
from determined.experimental import client
from determined.searcher.search_method import ExitedReason, Progress, SearchMethod

EXPERIMENT_ID_FILE = "experiment_id"
STATE_FILE = "state"


class SearchRunner:
    def __init__(
        self,
        search_method: SearchMethod,
    ) -> None:
        self.search_method = search_method

    def run_experiment(self, experiment_id: int) -> None:
        assert client._determined is not None
        session = client._determined._session

        experiment_is_active = True

        try:
            while experiment_is_active:
                time.sleep(5)
                events = bindings.get_GetSearcherEvents(session, experimentId=experiment_id)
                if events.searcherEvents is None:
                    continue
                logging.info(
                    json.dumps(
                        [SearchRunner._searcher_event_as_dict(e) for e in events.searcherEvents]
                    )
                )
                # the first event is an event we have already processed and told master about it
                # however, we may not have saved the state after that event if we crashed
                # after POSTing operations but before saving state
                last_event_id = self.search_method.searcher_state.last_event_id
                first_event = True
                for event in events.searcherEvents:
                    assert event.id is not None
                    skip_posting = (
                        first_event and last_event_id is not None and last_event_id < event.id
                    )
                    skip_event = (
                        first_event and last_event_id is not None and last_event_id == event.id
                    )
                    first_event = False

                    if skip_event:
                        continue

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
                            raise RuntimeError(
                                "trialExitedEarly event is invalid without exitedReason"
                            )
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
                            exited_reason=ExitedReason._from_bindings(
                                event.trialExitedEarly.exitedReason
                            ),
                        )
                    elif event.validationCompleted:
                        # duplicate completion accounting already performed by master
                        logging.info(
                            f"validationCompleted({event.validationCompleted.requestId},"
                            f" {event.validationCompleted.metric})"
                        )
                        request_id = uuid.UUID(event.validationCompleted.requestId)
                        if event.validationCompleted.metric is None:
                            raise RuntimeError(
                                "validationCompleted event is invalid without a metric"
                            )
                        operations = self.search_method.on_validation_completed(
                            request_id,
                            event.validationCompleted.metric,
                        )
                    elif event.experimentInactive:
                        logging.info(
                            f"experiment {self.search_method.searcher_state.experiment_id} is "
                            f"inactive; state={event.experimentInactive.experimentState}"
                        )
                        experiment_is_active = False
                        break
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

                    if skip_posting:
                        logging.warning(f"event {event.id} has already been acknowledged by master")
                    else:
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

                    # save state
                    assert event.id is not None  # TODO change proto to make id mandatory
                    self.search_method.searcher_state.last_event_id = event.id
                    self.save_state(experiment_id)
        except KeyboardInterrupt:
            print("Runner interrupted")

    def save_state(self, experiment_id: int) -> None:
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
        if experiment_id_file.exists():
            with experiment_id_file.open("r") as f:
                experiment_id = int(f.read())
            logging.info(f"Resuming HP searcher for experiment {experiment_id}")
            # TODO load searcher state and search method state
            loaded_experiment_id = self.search_method.load(self._get_state_path(experiment_id))
            assert experiment_id == loaded_experiment_id, (
                f"Experiment id mismatch. Expected {experiment_id}."
                f" Found {loaded_experiment_id}"
            )
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

        self.run_experiment(experiment_id)
        return experiment_id

    def save_state(self, experiment_id: int) -> None:
        self.search_method.save(
            self._get_state_path(experiment_id),
            experiment_id=experiment_id,
        )

    def _get_state_path(self, experiment_id: int) -> Path:
        return self.searcher_dir.joinpath(f"exp_{experiment_id}")
