import logging
import os
import pickle
import time
from typing import Any, Dict, List, Optional, Tuple, Union


import determined as det
from determined.common.api.bindings import (
    determinedexperimentv1State,
    get_GetExperiment,
    post_PauseExperiment,
)
from determined.experimental import client
from determined.searcher.search_method import Operation, SearchMethod
from determined.searcher.search_runner import SearchRunner


class CoreSearchRunner(SearchRunner):
    def __init__(self, search_method: SearchMethod, context: det.core.Context) -> None:
        super().__init__(search_method)
        self.context = context
        self.info = det.get_cluster_info()

        assert self.info is not None, "CoreSearchRunner only runs on-cluster"
        self.latest_checkpoint = self.info.latest_checkpoint

    def run(
        self,
        exp_config: Union[Dict[str, Any], str],
        context_dir: Optional[str] = None,
    ) -> int:
        logging.info("CoreSearchRunner.run")
        client._require_singleton(lambda: None)()

        operations: Optional[List[Operation]] = None

        if context_dir is None:
            context_dir = os.getcwd()

        if self.latest_checkpoint is not None:
            experiment_id, operations = self.load_state(self.latest_checkpoint)
            logging.info(f"Resuming HP searcher for experiment {experiment_id}")
        else:
            logging.info("No latest checkpoint. Starting new experiment.")
            exp = client.create_experiment(exp_config, context_dir)
            self.search_method.searcher_state.experiment_id = exp.id
            self.search_method.searcher_state.last_event_id = 0
            self.save_state(exp.id, [])
            experiment_id = exp.id

        client._require_singleton(lambda: None)()
        assert client._determined is not None
        session = client._determined._session
        self.run_experiment(experiment_id, session, operations)

        return experiment_id

    def load_state(self, storage_id: str) -> Tuple[int, List[Operation]]:
        with self.context.checkpoint.restore_path(storage_id) as path:
            experiment_id = self.search_method.load(path)
            with path.joinpath("ops").open("rb") as ops_file:
                operations = pickle.load(ops_file)
            return experiment_id, operations

    def save_state(self, experiment_id: int, operations: List[Operation]) -> None:
        steps_completed = self.search_method.searcher_state.last_event_id
        metadata = {"steps_completed": steps_completed}
        with self.context.checkpoint.store_path(metadata) as (path, storage_id):
            self.search_method.save(path, experiment_id=experiment_id)
            with path.joinpath("ops").open("wb") as ops_file:
                pickle.dump(operations, ops_file)

    def maybe_preempt(self, session: client.Session, multitrial_experiment_id: int) -> bool:
        if self.context.preempt.should_preempt():
            # if searcher is preempted, then pause multitrial experiment
            self._pause_and_wait(session, multitrial_experiment_id)
            return True
        return False

    def _pause_and_wait(self, session: client.Session, experiment_id: int) -> None:
        logging.info(f"Pausing multi-trial experiment {experiment_id}")
        exp = get_GetExperiment(session, experimentId=experiment_id).experiment
        if exp.state == determinedexperimentv1State.STATE_PAUSED:
            return
        elif _is_experiment_active(exp.state):
            post_PauseExperiment(session, id=experiment_id)

            while True:
                time.sleep(5)
                state = get_GetExperiment(session, experimentId=experiment_id).experiment.state
                if state == determinedexperimentv1State.STATE_PAUSED:
                    return
                elif not _is_experiment_active(state):
                    break

        logging.warning(f"Cannot pause Experiment {experiment_id} with current state {exp.state}.")

    def pause_searcher(self, session: client.Session) -> bool:
        logging.info("Pausing searcher experiment")

        assert self.info is not None

        exp_id = self.info.trial.experiment_id
        post_PauseExperiment(session, id=exp_id)
        while self.context.preempt.should_preempt() is False:
            time.sleep(5)
            continue

        return True


def _is_experiment_active(state: determinedexperimentv1State) -> bool:
    if (
        state == determinedexperimentv1State.STATE_ACTIVE
        or state == determinedexperimentv1State.STATE_QUEUED
        or state == determinedexperimentv1State.STATE_RUNNING
        or state == determinedexperimentv1State.STATE_STARTING
        or state == determinedexperimentv1State.STATE_PULLING
    ):
        return True
    return False
