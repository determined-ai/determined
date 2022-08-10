import json
import logging
import os
import time
import uuid
from pathlib import Path
from typing import Any, Dict, Optional, Sequence

from determined.common.api import bindings
from determined.common.api.bindings import v1TrialExitedEarlyExitedReason, v1SearcherEvent
from determined.experimental import client
from determined.searcher.search_method import ExitedReason, Progress, SearchMethod, SearcherState

EXPERIMENT_ID_FILE = "experiment_id"
STATE_FILE = "state"


class SearchRunner:
    def __init__(self, search_method: SearchMethod) -> None:
        self.search_method = search_method

    def run(
        self,
        exp_config: Dict[str, Any],
        context_dir: Optional[str] = None,
        searcher_dir: Optional[str] = None,
    ) -> int:
        logging.info("SearchRunner.run")

        if context_dir is None:
            context_dir = os.getcwd()
        searcher_dir_path = Path(searcher_dir) if searcher_dir is not None else Path.cwd()
        if not searcher_dir_path.is_dir():
            raise FileExistsError(f"searcher_dir={searcher_dir} already exists and is not a directory")
        if not searcher_dir_path.exists():
            searcher_dir_path.mkdir(parents=True)
        experiment_id_file = searcher_dir_path.joinpath(EXPERIMENT_ID_FILE)
        if experiment_id_file.exists():
            with experiment_id_file.open("r") as f:
                experiment_id = int(f.read())
            logging.info(f"Resuming HP searcher for experiment {experiment_id}")
            # TODO load searcher state and search method state
            searcher_state_file = searcher_dir_path.joinpath(STATE_FILE)
            if searcher_state_file.exists():
                with searcher_state_file.open("r") as f:
                    state_dict = json.load(f)
                    self.search_method.searcher_state.from_dict(state_dict)
                    if self.search_method.searcher_state.experiment_completed:
                        logging.warning(f"experiment {experiment_id} has completed")
                        return experiment_id
            last_event_id = self.search_method.searcher_state.last_event_id
            if last_event_id is not None:
                # TODO make sure checkpoint is saved before state!!
                self.search_method.load_checkpoint(last_event_id)
        else:
            exp = client.create_experiment(exp_config, context_dir)
            with experiment_id_file.open("w") as f:
                f.write(str(exp.id))
            logging.info(f"Starting HP searcher for experiment {exp.id}")
            experiment_id = exp.id
            last_event_id = None

        assert client._determined is not None
        session = client._determined._session

        experiment_is_active = True

        try:
            while experiment_is_active:
                time.sleep(5)
                events = bindings.get_GetSearcherEvents(session, experimentId=experiment_id)
                if events.searcherEvents is None:
                    continue
                logging.info(json.dumps([SearchRunner._searcher_event_as_dict(e) for e in events.searcherEvents]))
                for e in events.searcherEvents:
                    if e.initialOperations:
                        logging.info("initial operations")
                        operations = self.search_method.initial_operations()
                    elif e.trialCreated:
                        logging.info(f"trialCreated({e.trialCreated.requestId})")
                        request_id = uuid.UUID(e.trialCreated.requestId)
                        self.search_method.searcher_state.trials_created.add(request_id)
                        self.search_method.searcher_state.trial_progress[request_id] = 0.0
                        operations = self.search_method.on_trial_created(request_id)
                    elif e.trialClosed:
                        logging.info(f"trialClosed({e.trialClosed.requestId})")
                        request_id = uuid.UUID(e.trialClosed.requestId)
                        self.search_method.searcher_state.trials_closed.add(request_id)
                        operations = self.search_method.on_trial_closed(request_id)
                    elif e.trialExitedEarly:
                        # duplicate exit accounting already performed by master
                        logging.info(
                            f"trialExitedEarly({e.trialExitedEarly.requestId},"
                            f" {e.trialExitedEarly.exitedReason})"
                        )
                        if e.trialExitedEarly.exitedReason is None:
                            raise RuntimeError(
                                "trialExitedEarly event is invalid without exitedReason"
                            )
                        request_id = uuid.UUID(e.trialExitedEarly.requestId)
                        if e.trialExitedEarly.exitedReason in (
                            v1TrialExitedEarlyExitedReason.EXITED_REASON_INVALID_HP,
                            v1TrialExitedEarlyExitedReason.EXITED_REASON_INIT_INVALID_HP,
                        ):
                            self.search_method.searcher_state.trial_progress.pop(request_id, None)
                        elif (
                            e.trialExitedEarly.exitedReason
                            == v1TrialExitedEarlyExitedReason.EXITED_REASON_UNSPECIFIED
                        ):
                            self.search_method.searcher_state.failures.add(request_id)
                        operations = self.search_method.on_trial_exited_early(
                            request_id,
                            exited_reason=ExitedReason._from_bindings(
                                e.trialExitedEarly.exitedReason
                            ),
                        )
                    elif e.validationCompleted:
                        # duplicate completion accounting already performed by master
                        logging.info(
                            f"validationCompleted({e.validationCompleted.requestId},"
                            f" {e.validationCompleted.metric})"
                        )
                        request_id = uuid.UUID(e.validationCompleted.requestId)
                        if e.validationCompleted.metric is None:
                            raise RuntimeError(
                                "validationCompleted event is invalid without a metric"
                            )
                        operations = self.search_method.on_validation_completed(
                            request_id,
                            e.validationCompleted.metric,
                        )
                    elif e.experimentInactive:
                        logging.info(
                            f"experiment {experiment_id} is inactive"
                            f" state={e.experimentInactive.experimentState}"
                        )
                        experiment_is_active = False
                        break
                    elif e.trialProgress:
                        logging.debug(
                            f"trialProgress({e.trialProgress.requestId}, "
                            f"{e.trialProgress.partialUnits})"
                        )
                        request_id = uuid.UUID(e.trialProgress.requestId)
                        self.search_method.searcher_state.trial_progress[request_id] = float(
                            e.trialProgress.partialUnits
                        )
                        progress = self.search_method.progress()
                        operations = [Progress(progress)]
                    else:
                        raise RuntimeError(f"Unsupported event {e}")
                    bindings.post_PostSearcherOperations(
                        session,
                        body=bindings.v1PostSearcherOperationsRequest(
                            experimentId=experiment_id,
                            searcherOperations=[op._to_searcher_operation() for op in operations],
                            triggeredByEvent=e,
                        ),
                        experimentId=experiment_id,
                    )

                    # save state
                    self.search_method.save_checkpoint(e.id)
                    self.search_method.searcher_state.last_event_id = e.id
                    d = self.search_method.searcher_state.to_dict()
                    searcher_state_file = searcher_dir_path.joinpath(STATE_FILE)
                    with searcher_state_file.open("w") as f:
                        json.dump(d, f)
        except KeyboardInterrupt:
            print("Runner interrupted")

        return experiment_id

    @staticmethod
    def _searcher_event_as_dict(event: v1SearcherEvent) -> dict:
        d = dict()
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