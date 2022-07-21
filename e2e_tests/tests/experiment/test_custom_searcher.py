import logging
import uuid
from typing import List

import pytest

from determined.searcher.search_method import (
    Close,
    Create,
    ExitedReason,
    Operation,
    SearchMethod,
    SearchState,
    Shutdown,
    ValidateAfter,
)
from determined.searcher.search_runner import SearchRunner
from tests import config as conf


@pytest.mark.e2e_cpu
def test_run_custom_searcher_experiment() -> None:
    # example searcher script
    config = conf.load_config(conf.fixtures_path("no_op/single.yaml"))
    config["searcher"] = {
        "name": "custom",
        "metric": "validation_error",
        "smaller_is_better": True,
        "max_length": {"batches": 3000},
    }
    config["description"] = "custom searcher"
    search_method = SingleSearchMethod(config)
    search_runner = SearchRunner(search_method)
    search_runner.run(config, context_dir=conf.fixtures_path("no_op"))


class SingleSearchMethod(SearchMethod):
    def __init__(self, experiment_config: dict) -> None:
        super().__init__(SearchState(None))
        # since this is a single trial the hyperparameter space comprises a single point
        self.hyperparameters = experiment_config["hyperparameters"]

    def on_trial_created(self, request_id: uuid.UUID) -> List[Operation]:
        return []

    def on_validation_completed(self, request_id: uuid.UUID, metric: float) -> List[Operation]:
        return []

    def on_trial_closed(self, request_id: uuid.UUID) -> List[Operation]:
        return []

    def progress(self) -> float:
        return 0.99  # TODO change signature

    def on_trial_exited_early(
        self, request_id: uuid.UUID, exit_reason: ExitedReason
    ) -> List[Operation]:
        logging.warning(f"Trial {request_id} exited early: {exit_reason}")
        return [Shutdown()]

    def initial_operations(self) -> List[Operation]:
        logging.info("initial_operations")

        create = Create(
            request_id=uuid.uuid4(),
            hparams=self.hyperparameters,
            checkpoint=None,
        )
        validate_after = ValidateAfter(request_id=create.request_id, length=3000)
        close = Close(request_id=create.request_id)
        logging.debug(f"Create({create.request_id}, {create.hparams})")
        return [create, validate_after, close]
