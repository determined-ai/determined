import logging
import os
import pickle
from typing import Any, Dict, List, Optional, Tuple

import determined as det
from determined.experimental import client
from determined.searcher.search_method import Operation, SearchMethod
from determined.searcher.search_runner import SearchRunner


class CoreSearchRunner(SearchRunner):
    def __init__(self, search_method: SearchMethod, context: det.core.Context) -> None:
        super().__init__(search_method)
        self.context = context
        info = det.get_cluster_info()

        assert info is not None, "CoreSearchRunner only runs on-cluster"
        self.latest_checkpoint = info.latest_checkpoint

    def run(
        self,
        exp_config: Dict[str, Any],
        context_dir: Optional[str] = None,
    ) -> int:
        logging.info("CoreSearchRunner.run")

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

        self.run_experiment(experiment_id, operations)

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
