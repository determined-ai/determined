import logging
import os
import pickle
from typing import Any, Dict, List, Optional, Tuple, Union

import determined as det
from determined import searcher
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

    def _show_experiment_paused_msg(self) -> None:
        logging.warning(
            f"Experiment {self.search_method.searcher_state.experiment_id} "
            f"has been paused. If you leave searcher experiment running, "
            f"your search method will automatically resume when the experiment "
            f"becomes active again."
        )
