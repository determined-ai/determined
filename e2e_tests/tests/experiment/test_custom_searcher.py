import logging
import random
import uuid
from typing import Dict, List

import pytest

from determined.common.api import bindings
from determined.experimental import client
from determined.searcher.search_method import (
    Close,
    Create,
    ExitedReason,
    Operation,
    SearchMethod,
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
    search_method = SingleSearchMethod(config, 3000)
    search_runner = SearchRunner(search_method)
    experiment_id = search_runner.run(config, context_dir=conf.fixtures_path("no_op"))

    assert client._determined is not None
    session = client._determined._session
    response = bindings.get_GetExperiment(session, experimentId=experiment_id)
    assert response.experiment.numTrials == 1


class SingleSearchMethod(SearchMethod):
    def __init__(self, experiment_config: dict, max_length: int) -> None:
        super().__init__()
        # since this is a single trial the hyperparameter space comprises a single point
        self.hyperparameters = experiment_config["hyperparameters"]
        self.max_length = max_length
        self.trial_closed = False

    def on_trial_created(self, request_id: uuid.UUID) -> List[Operation]:
        return []

    def on_validation_completed(self, request_id: uuid.UUID, metric: float) -> List[Operation]:
        return []

    def on_trial_closed(self, request_id: uuid.UUID) -> List[Operation]:
        self.trial_closed = True
        return [Shutdown()]

    def progress(self) -> float:
        if self.trial_closed:
            return 1.0
        (the_trial,) = self.searcher_state.trials_created
        return self.searcher_state.trial_progress[the_trial] / self.max_length

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
        validate_after = ValidateAfter(request_id=create.request_id, length=self.max_length)
        close = Close(request_id=create.request_id)
        logging.debug(f"Create({create.request_id}, {create.hparams})")
        return [create, validate_after, close]


@pytest.mark.e2e_cpu_2a
def test_run_random_searcher_exp() -> None:
    config = conf.load_config(conf.fixtures_path("no_op/single.yaml"))
    config["searcher"] = {
        "name": "custom",
        "metric": "validation_error",
        "smaller_is_better": True,
        "max_length": {"batches": 3000},
    }
    config["description"] = "custom searcher"

    max_trials = 5
    max_concurrent_trials = 2
    max_length = 3000

    search_method = RandomSearcherMethod(max_trials, max_concurrent_trials, max_length)
    search_runner = SearchRunner(search_method)
    experiment_id = search_runner.run(config, context_dir=conf.fixtures_path("no_op"))

    assert client._determined is not None
    session = client._determined._session
    response = bindings.get_GetExperiment(session, experimentId=experiment_id)
    assert response.experiment.numTrials == 5
    assert search_method.created_trials == 5
    assert search_method.pending_trials == 0
    assert search_method.closed_trials == 5


class RandomSearcherMethod(SearchMethod):
    def __init__(self, max_trials: int, max_concurrent_trials: int, max_length: int) -> None:
        super().__init__()
        self.max_trials = max_trials
        self.max_concurrent_trials = max_concurrent_trials
        self.max_length = max_length

        self.created_trials = 0
        self.pending_trials = 0
        self.closed_trials = 0

    def on_trial_created(self, request_id: uuid.UUID) -> List[Operation]:
        self._log_stats()
        return []

    def on_validation_completed(self, request_id: uuid.UUID, metric: float) -> List[Operation]:
        return []

    def on_trial_closed(self, request_id: uuid.UUID) -> List[Operation]:
        self.pending_trials -= 1
        self.closed_trials += 1
        ops: List[Operation] = []
        if self.created_trials < self.max_trials:
            request_id = uuid.uuid4()
            ops.append(Create(request_id=request_id, hparams=self.sample_params(), checkpoint=None))
            ops.append(ValidateAfter(request_id=request_id, length=self.max_length))
            ops.append(Close(request_id=request_id))
            self.created_trials += 1
            self.pending_trials += 1
        elif self.pending_trials == 0:
            ops.append(Shutdown())

        self._log_stats()
        return ops

    def progress(self) -> float:
        if 0 < self.max_concurrent_trials < self.pending_trials:
            logging.error("pending trials is greater than max_concurrent_trial")
        units_completed = sum(
            (
                self.max_length
                if r in self.searcher_state.trials_closed
                else self.searcher_state.trial_progress[r]
                for r in self.searcher_state.trial_progress
            )
        )
        units_expected = self.max_length * self.max_trials
        progress = units_completed / units_expected

        logging.info(f"progress = {progress}")

        return progress

    def on_trial_exited_early(
        self, request_id: uuid.UUID, exit_reason: ExitedReason
    ) -> List[Operation]:
        self.pending_trials -= 1

        ops: List[Operation] = []
        if exit_reason == ExitedReason.INVALID_HP or exit_reason == ExitedReason.INIT_INVALID_HP:
            request_id = uuid.uuid4()
            ops.append(Create(request_id=request_id, hparams=self.sample_params(), checkpoint=None))
            ops.append(ValidateAfter(request_id=request_id, length=self.max_length))
            ops.append(Close(request_id=request_id))
            self.pending_trials += 1
            return ops

        self.closed_trials += 1
        self._log_stats()
        return ops

    def initial_operations(self) -> List[Operation]:
        initial_trials = self.max_trials
        max_concurrent_trials = self.max_concurrent_trials
        if max_concurrent_trials > 0:
            initial_trials = min(initial_trials, max_concurrent_trials)

        ops: List[Operation] = []

        for _ in range(initial_trials):
            create = Create(
                request_id=uuid.uuid4(),
                hparams=self.sample_params(),
                checkpoint=None,
            )
            ops.append(create)
            ops.append(ValidateAfter(request_id=create.request_id, length=self.max_length))
            ops.append(Close(request_id=create.request_id))

            self.created_trials += 1
            self.pending_trials += 1

        self._log_stats()
        return ops

    def _log_stats(self) -> None:
        logging.info(f"created trials={self.created_trials}")
        logging.info(f"pending trials={self.pending_trials}")
        logging.info(f"closed trials={self.closed_trials}")

    def sample_params(self) -> Dict[str, int]:
        hparams = {"global_batch_size": random.randint(10, 100)}
        logging.info(f"hparams={hparams}")
        return hparams
