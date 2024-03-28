import logging
import os
import pathlib
import pickle
from typing import Any, Dict, Iterable, List, Optional, Tuple, Union

import determined as det
from determined import searcher
from determined.experimental import client

logger = logging.getLogger("determined.searcher")


class RemoteSearchRunner(searcher.SearchRunner):
    """
    ``RemoteSearchRunner`` performs a search for optimal hyperparameter values,
    applying the provided ``SearchMethod`` (you will subclass ``SearchMethod`` and provide
    an instance of the derived class).
    ``RemoteSearchRunner`` executes on-cluster: it runs a meta-experiment
    using ``Core API``.
    """

    def __init__(self, search_method: searcher.SearchMethod, context: det.core.Context) -> None:
        super().__init__(search_method)
        self.context = context
        info = det.get_cluster_info()
        assert info is not None, "RemoteSearchRunner only runs on-cluster"
        self.info = info

        self.latest_checkpoint = self.info.latest_checkpoint

    def run(
        self,
        exp_config: Union[Dict[str, Any], str],
        model_dir: Optional[str] = None,
        includes: Optional[Iterable[Union[str, pathlib.Path]]] = None,
    ) -> int:
        """
        Run custom search as a Core API experiment (on-cluster).

        Args:
            exp_config (dictionary, string): experiment config filename (.yaml) or a dict.
            model_dir (string): directory containing model definition.
            includes (Iterable[Union[str, pathlib.Path]], optional): Additional files
                or directories to include in the model definition.  (default: ``None``)
        """
        logger.info("RemoteSearchRunner.run")

        operations: Optional[List[searcher.Operation]] = None

        if model_dir is None:
            model_dir = os.getcwd()

        if self.latest_checkpoint is not None:
            experiment_id, operations = self.load_state(self.latest_checkpoint)
            logger.info(f"Resuming HP searcher for experiment {experiment_id}")
        else:
            logger.info("No latest checkpoint. Starting new experiment.")
            exp = client.create_experiment(exp_config, model_dir, includes)
            self.state.experiment_id = exp.id
            self.state.last_event_id = 0
            self.save_state(exp.id, [])
            experiment_id = exp.id
            # Note: Simulating the same print functionality as our CLI when making an experiment.
            # This line is needed for the e2e tests
            logger.info(f"Created experiment {exp.id}")

        # make sure client is initialized
        # TODO: remove typing suppression when mypy #14473 is resolved
        client._require_singleton(lambda: None)()  # type: ignore
        assert client._determined is not None
        session = client._determined._session
        self.run_experiment(experiment_id, session, operations)

        return experiment_id

    def load_state(self, storage_id: str) -> Tuple[int, List[searcher.Operation]]:
        with self.context.checkpoint.restore_path(storage_id) as path:
            self.state, experiment_id = self.search_method.load(path)
            with path.joinpath("ops").open("rb") as ops_file:
                operations = pickle.load(ops_file)
            return experiment_id, operations

    def save_state(self, experiment_id: int, operations: List[searcher.Operation]) -> None:
        steps_completed = self.state.last_event_id
        metadata = {"steps_completed": steps_completed}
        with self.context.checkpoint.store_path(metadata) as (path, storage_id):
            self.search_method.save(self.state, path, experiment_id=experiment_id)
            with path.joinpath("ops").open("wb") as ops_file:
                pickle.dump(operations, ops_file)

    def _show_experiment_paused_msg(self) -> None:
        logger.warning(
            f"Experiment {self.state.experiment_id} "
            "has been paused. If you leave searcher experiment running, "
            "your search method will automatically resume when the experiment "
            "becomes active again."
        )
