import logging
from typing import Any, Dict, Optional

import determined as det
from determined.experimental import client
from determined.searcher.search_method import SearchMethod
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
    ) -> None:
        logging.info("CoreSearchRunner.run")

        if self.latest_checkpoint is not None:
            experiment_id = self.load_state(self.latest_checkpoint)
        else:
            exp = client.create_experiment(exp_config, context_dir)
            experiment_id = exp.id
            self.search_method.searcher_state.last_event_id = None

        self.run_experiment(experiment_id)

    def load_state(self, storage_id: str) -> int:
        with self.context.checkpoint.restore_path(storage_id) as path:
            return self.search_method.load(path)

    def save_state(self, experiment_id: int) -> None:
        with self.context.checkpoint.store_path() as (path, storage_id):
            self.search_method.save(path, experiment_id=experiment_id)
