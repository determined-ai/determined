import json
import logging
import os
import time
import uuid
from typing import Any, Dict, Optional

from determined.common.api import bindings
from determined.common.api.bindings import v1TrialExitedEarlyExitedReason
from determined.experimental import client
from determined.searcher.search_method import ExitedReason, Progress, SearchMethod


class SearchRunner:
    def __init__(self, search_method: SearchMethod) -> None:
        self.search_method = search_method

    def run(
        self,
        exp_config: Dict[str, Any],
        context_dir: Optional[str] = None,
        resume_exp_id: Optional[int] = None,
    ) -> int:
        logging.info("SearchRunner.run")

        if context_dir is None:
            context_dir = os.getcwd()
        if resume_exp_id is None:
            exp = client.create_experiment(exp_config, context_dir)
        else:
            exp = client.get_experiment(resume_exp_id)
            # TODO obtain searcher state from master
            # searcher_state = client.get_searcher_state(resume_exp_id)
            searcher_state = {"lastEventId": 1}
            last_event_id = searcher_state["lastEventId"]
            self.search_method.load_checkpoint(last_event_id)

        # searcher_state = exp.get
        assert client._determined is not None
        session = client._determined._session
        experiment_id: int = exp.id
        logging.debug(f"Running experiment {experiment_id}")

        experiment_is_active = True

        try:
            while experiment_is_active:
                time.sleep(5)
                events = bindings.get_GetSearcherEvents(session, experimentId=experiment_id)
                if events.searcherEvents is None:
                    continue
                logging.warning(json.dumps([e.to_json() for e in events.searcherEvents]))
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
                            experimentId=exp._id,
                            searcherOperations=[op._to_searcher_operation() for op in operations],
                            triggeredByEvent=e,
                        ),
                        experimentId=exp._id,
                    )
        except KeyboardInterrupt:
            print("Runner interrupted")

        return experiment_id
