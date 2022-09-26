import logging
import os
import pickle
import time
from typing import Any, Dict, List, Optional, Tuple, Union

import determined as det
from determined import searcher
from determined.common.api import bindings
from determined.experimental import client

logger = logging.getLogger("determined.searcher")


class CoreSearchRunner(searcher.SearchRunner):
    """
    ``CoreSearchRunner`` performs a search for optimal hyperparameter values on-cluster,
    applying the provided ``SearchMethod`` (you will subclass ``SearchMethod`` and provide
    an instance of the derived class).
    ``CoreSearchRunner`` is intended to execute on-cluster: it runs a meta-experiment
    using ``Core API``.
    """

    def __init__(self, search_method: searcher.SearchMethod, context: det.core.Context) -> None:
        super().__init__(search_method)
        self.context = context
        info = det.get_cluster_info()
        assert info is not None, "CoreSearchRunner only runs on-cluster"
        self.info = info

        self.latest_checkpoint = self.info.latest_checkpoint

    def run(
        self,
        exp_config: Union[Dict[str, Any], str],
        context_dir: Optional[str] = None,
    ) -> int:
        logger.info("CoreSearchRunner.run")

        operations: Optional[List[searcher.Operation]] = None

        if context_dir is None:
            context_dir = os.getcwd()

        if self.latest_checkpoint is not None:
            experiment_id, operations = self.load_state(self.latest_checkpoint)
            logger.info(f"Resuming HP searcher for experiment {experiment_id}")
        else:
            logger.info("No latest checkpoint. Starting new experiment.")
            exp = client.create_experiment(exp_config, context_dir)
            self.search_method.searcher_state.experiment_id = exp.id
            self.search_method.searcher_state.last_event_id = 0
            self.save_state(exp.id, [])
            experiment_id = exp.id

        # make sure client is initialized
        client._require_singleton(lambda: None)()
        assert client._determined is not None
        session = client._determined._session
        self.run_experiment(experiment_id, session, operations)

        return experiment_id

    def load_state(self, storage_id: str) -> Tuple[int, List[searcher.Operation]]:
        with self.context.checkpoint.restore_path(storage_id) as path:
            experiment_id = self.search_method.load(path)
            with path.joinpath("ops").open("rb") as ops_file:
                operations = pickle.load(ops_file)
            return experiment_id, operations

    def save_state(self, experiment_id: int, operations: List[searcher.Operation]) -> None:
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
        logger.info(f"Pausing multi-trial experiment {experiment_id}")
        exp = bindings.get_GetExperiment(session, experimentId=experiment_id).experiment
        if exp.state == bindings.determinedexperimentv1State.STATE_PAUSED:
            return
        elif _is_experiment_active(exp.state):
            bindings.post_PauseExperiment(session, id=experiment_id)

            while True:
                time.sleep(5)
                state = bindings.get_GetExperiment(
                    session, experimentId=experiment_id
                ).experiment.state
                if state == bindings.determinedexperimentv1State.STATE_PAUSED:
                    return
                elif not _is_experiment_active(state):
                    break

        logger.warning(f"Cannot pause Experiment {experiment_id} with current state {exp.state}.")

    def pause_searcher(self, session: client.Session) -> bool:
        logger.info("Pausing searcher experiment")

        exp_id = self.info.trial.experiment_id
        bindings.post_PauseExperiment(session, id=exp_id)
        while self.context.preempt.should_preempt() is False:
            time.sleep(5)
            continue

        return True


def _is_experiment_active(state: bindings.determinedexperimentv1State) -> bool:
    return state in (
        bindings.determinedexperimentv1State.STATE_ACTIVE,
        bindings.determinedexperimentv1State.STATE_QUEUED,
        bindings.determinedexperimentv1State.STATE_RUNNING,
        bindings.determinedexperimentv1State.STATE_STARTING,
        bindings.determinedexperimentv1State.STATE_PULLING,
    )
