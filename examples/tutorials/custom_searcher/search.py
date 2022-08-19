import logging
import os
import pathlib
import random
import sys
import uuid
import yaml

from typing import Dict, List, Optional

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


class RandomSearcherMethod(SearchMethod):
    def __init__(
        self,
        max_trials: int,
        max_concurrent_trials: int,
        max_length: int,
        exception: Optional[Exception] = None,
    ) -> None:
        super().__init__()
        self.max_trials = max_trials
        self.max_concurrent_trials = max_concurrent_trials
        self.max_length = max_length
        self.exception = exception

        # TODO remove created_trials and closed_trials before merging the feature branch
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

        if progress >= 0.5 and self.exception is not None:
            exception = self.exception
            self.exception = None
            raise exception


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


def main(searcher_dir: str):
    search_method = RandomSearcherMethod(5, 2, 3000)
    search_runner = SearchRunner(search_method)

    with open("no_op/random.yaml", 'r') as stream:
        config = yaml.safe_load(stream)

    config["searcher"] = {
        "name": "custom",
        "metric": "validation_error",
        "smaller_is_better": True,
    }

    client.login(master="http://35.85.175.88:8080", user="determined", password="")
        # resuming experiment
    files = os.listdir()
    if "experiment_id" in files:
        with open(pathlib.Path(searcher_dir).joinpath("experiment_id"), "r") as f:
            experiment_id = int(f.read())
        search_runner.run(config, context_dir=".", resume_exp_id=experiment_id)
    else:

    experiment_id = search_runner.run(config, context_dir=".")
    print(f"experiment = {experiment_id}")


if __name__ == "__main__":
    main(sys.argv[1])
    if len(sys.argv) > 1:
        main(sys.argv[1])
    else
        print("usage: search.py searcher_directory")
