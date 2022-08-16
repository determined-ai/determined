import dataclasses
import logging
import random
import sys
import uuid
from dataclasses import dataclass
from typing import Dict, List, Set

import numpy as np
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
        "unit": "batches",
    }
    config["name"] = "single"
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
        "unit": "batches",
    }
    config["name"] = "random"
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
    assert len(search_method.searcher_state.trials_created) == search_method.created_trials
    assert len(search_method.searcher_state.trials_closed) == search_method.closed_trials


class RandomSearcherMethod(SearchMethod):
    def __init__(self, max_trials: int, max_concurrent_trials: int, max_length: int) -> None:
        super().__init__()
        self.max_trials = max_trials
        self.max_concurrent_trials = max_concurrent_trials
        self.max_length = max_length

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


@pytest.mark.e2e_cpu
def test_run_asha_batches_exp() -> None:
    config = conf.load_config(conf.fixtures_path("no_op/adaptive.yaml"))
    config["searcher"] = {
        "name": "custom",
        "metric": "validation_error",
        "smaller_is_better": True,
        "unit": "batches",
    }
    config["name"] = "asha"
    config["description"] = "custom searcher"

    max_length = 3000
    max_trials = 16
    num_rungs = 3
    divisor = 4

    search_method = ASHASearchMethod(max_length, max_trials, num_rungs, divisor)
    search_runner = SearchRunner(search_method)
    experiment_id = search_runner.run(config, context_dir=conf.fixtures_path("no_op"))

    assert client._determined is not None
    session = client._determined._session
    response = bindings.get_GetExperiment(session, experimentId=experiment_id)

    assert response.experiment.numTrials == 16
    assert search_method.pending_trials == 0
    assert search_method.completed_trials == 16
    assert len(search_method.searcher_state.trials_closed) == len(search_method.closed_trials)

    response_trials = bindings.get_GetExperimentTrials(session, experimentId=experiment_id).trials

    # 16 trials in rung 1 (#batches = 187)
    assert sum([t.totalBatchesProcessed >= 187 for t in response_trials]) == 16
    # at least 4 trials in rung 2 (#batches = 750)
    assert sum([t.totalBatchesProcessed >= 750 for t in response_trials]) >= 4
    # at least 1 trial in rung 3 (#batches = 3000)
    assert sum([t.totalBatchesProcessed == 3000 for t in response_trials]) >= 1

    for trial in response_trials:
        assert trial.state == bindings.determinedexperimentv1State.STATE_COMPLETED


@dataclass
class TrialMetric:
    request_id: uuid.UUID
    metric: float
    promoted: bool = False


@dataclass
class Rung:
    units_needed: int
    idx: int
    metrics: List[TrialMetric] = dataclasses.field(default_factory=list)
    outstanding_trials: int = 0

    def promotions_async(
        self, request_id: uuid.UUID, metric: float, divisor: int
    ) -> List[uuid.UUID]:
        logging.info(f"Rung {self.idx}")
        logging.info(f"outstanding_trials {self.outstanding_trials}")

        old_num_promote = len(self.metrics) // divisor
        num_promote = (len(self.metrics) + 1) // divisor

        index = self._search_metric_index(metric)
        promote_now = index < num_promote
        trial_metric = TrialMetric(request_id=request_id, metric=metric, promoted=promote_now)
        self.metrics.insert(index, trial_metric)

        if promote_now:
            return [request_id]
        if num_promote != old_num_promote and not self.metrics[old_num_promote].promoted:
            self.metrics[old_num_promote].promoted = True
            return [self.metrics[old_num_promote].request_id]

        logging.info("No promotion")
        return []

    def _search_metric_index(self, metric: float) -> int:
        i: int = 0
        j: int = len(self.metrics)
        while i < j:
            mid = (i + j) >> 1
            if self.metrics[mid].metric <= metric:
                i = mid + 1
            else:
                j = mid
        return i


class ASHASearchMethod(SearchMethod):
    def __init__(
        self,
        max_length: int,
        max_trials: int,
        num_rungs: int,
        divisor: int,
        max_concurrent_trials: int = 0,
    ) -> None:
        super().__init__()

        # Asha params
        self.max_length = max_length
        self.max_trials = max_trials
        self.num_rungs = num_rungs
        self.divisor = divisor
        self.max_concurrent_trials = max_concurrent_trials

        self.is_smaller_better = True

        # structs
        self.rungs: List[Rung] = []
        self.trial_rungs: Dict[uuid.UUID, int] = {}

        # accounting
        self.pending_trials: int = 0
        self.completed_trials: int = 0
        self.invalid_trials: int = 0
        self.early_exit_trials: Set[uuid.UUID] = set()
        self.closed_trials: Set[uuid.UUID] = set()

        self._init_rungs()

    def _init_rungs(self) -> None:
        units_needed = 0
        for idx in range(self.num_rungs):
            downsampling_rate = pow(self.divisor, float(self.num_rungs - idx - 1))
            units_needed += max(int(self.max_length / downsampling_rate), 1)
            self.rungs.append(Rung(units_needed, idx))

    def on_trial_closed(self, request_id: uuid.UUID) -> List[Operation]:
        self.completed_trials += 1
        self.closed_trials.add(request_id)

        if self.pending_trials == 0 and self.completed_trials == self.max_trials:
            return [Shutdown()]

        return []

    def on_trial_created(self, request_id: uuid.UUID) -> List[Operation]:
        self.rungs[0].outstanding_trials += 1
        self.trial_rungs[request_id] = 0
        return []

    def on_validation_completed(self, request_id: uuid.UUID, metric: float) -> List[Operation]:
        self.pending_trials -= 1
        if self.is_smaller_better is False:
            metric *= -1
        return self.promote_async(request_id, metric)

    def on_trial_exited_early(
        self, request_id: uuid.UUID, exited_reason: ExitedReason
    ) -> List[Operation]:
        self.pending_trials -= 1
        if (
            exited_reason == ExitedReason.INVALID_HP
            or exited_reason == ExitedReason.INIT_INVALID_HP
        ):
            ops: List[Operation] = []

            self.early_exit_trials.add(request_id)
            ops.append(Close(request_id))
            self.closed_trials.add(request_id)
            self.invalid_trials += 1

            highest_rung_index = self.trial_rungs[request_id]
            rung = self.rungs[highest_rung_index]
            rung.outstanding_trials -= 1

            for rung_idx in range(0, highest_rung_index + 1):
                rung = self.rungs[rung_idx]
                rung.metrics = list(filter(lambda x: x.request_id != request_id, rung.metrics))

            create = Create(
                request_id=uuid.uuid4(),
                hparams=self.sample_params(),
                checkpoint=None,
            )
            ops.append(create)
            ops.append(
                ValidateAfter(request_id=create.request_id, length=self.rungs[0].units_needed)
            )

            self.trial_rungs[create.request_id] = 0
            self.pending_trials += 1

            return ops

        self.early_exit_trials.add(request_id)
        self.closed_trials.add(request_id)
        return self.promote_async(request_id, 100.0)

    def initial_operations(self) -> List[Operation]:
        ops: List[Operation] = []

        if self.max_concurrent_trials > 0:
            max_concurrent_trials = min(self.max_concurrent_trials, self.max_trials)
        else:
            max_concurrent_trials = max(
                1, min(int(pow(self.divisor, self.num_rungs - 1)), self.max_trials)
            )

        for _ in range(0, max_concurrent_trials):
            create = Create(
                request_id=uuid.uuid4(),
                hparams=self.sample_params(),
                checkpoint=None,
            )
            ops.append(create)
            ops.append(
                ValidateAfter(request_id=create.request_id, length=self.rungs[0].units_needed)
            )

            self.trial_rungs[create.request_id] = 0
            self.pending_trials += 1

        return ops

    def promote_async(self, request_id: uuid.UUID, metric: float) -> List[Operation]:
        rung_idx = self.trial_rungs[request_id]
        rung = self.rungs[rung_idx]
        rung.outstanding_trials -= 1
        added_train_workload = False

        ops: List[Operation] = []

        if rung_idx == self.num_rungs - 1:
            rung.metrics.append(TrialMetric(request_id=request_id, metric=metric))

            if request_id not in self.early_exit_trials:
                ops.append(Close(request_id=request_id))
                logging.info(f"Closing trial {request_id}")
                self.closed_trials.add(request_id)
        else:
            next_rung = self.rungs[rung_idx + 1]
            logging.info(f"Promoting in rung {rung_idx}")
            for promoted_request_id in rung.promotions_async(request_id, metric, self.divisor):
                self.trial_rungs[promoted_request_id] = rung_idx + 1
                next_rung.outstanding_trials += 1
                if promoted_request_id not in self.early_exit_trials:
                    logging.info(f"Promoted {promoted_request_id}")
                    units_needed = max(next_rung.units_needed - rung.units_needed, 1)
                    ops.append(ValidateAfter(promoted_request_id, units_needed))
                    added_train_workload = True
                    self.pending_trials += 1
                else:
                    return self.promote_async(promoted_request_id, sys.float_info.max)

        all_trials = len(self.trial_rungs) - self.invalid_trials
        if not added_train_workload and all_trials < self.max_trials:
            logging.info("Creating new trial instead of promoting")
            self.pending_trials += 1

            create = Create(
                request_id=uuid.uuid4(),
                hparams=self.sample_params(),
                checkpoint=None,
            )
            ops.append(create)
            ops.append(
                ValidateAfter(request_id=create.request_id, length=self.rungs[0].units_needed)
            )
            self.trial_rungs[create.request_id] = 0

        if len(self.rungs[0].metrics) == self.max_trials:
            ops.extend(self._get_close_rungs_ops())

        return ops

    def _get_close_rungs_ops(self) -> List[Operation]:
        ops: List[Operation] = []

        for rung in self.rungs:
            if rung.outstanding_trials > 0:
                break
            for trial_metric in rung.metrics:
                if not trial_metric.promoted and trial_metric.request_id not in self.closed_trials:
                    if trial_metric.request_id not in self.early_exit_trials:
                        logging.info(f"Closing trial {trial_metric.request_id}")
                        ops.append(Close(trial_metric.request_id))
                        self.closed_trials.add(trial_metric.request_id)
        return ops

    def sample_params(self) -> Dict[str, object]:
        hparams = {
            "global_batch_size": 10,
            "metrics_base": 0.05 * (len(self.trial_rungs)+1),
            "metrics_progression": "constant",
        }
        logging.info(f"hparams={hparams}")
        return hparams

    def progress(self) -> float:
        if 0 < self.max_concurrent_trials < self.pending_trials:
            raise RuntimeError("Pending trial is greater than max concurrent trials")

        all_trials = len(self.rungs[0].metrics)
        progress = all_trials / (1.2 * self.max_trials)
        if all_trials == self.max_trials:
            num_valid_trials = self.completed_trials - self.invalid_trials
            progress_no_overhead = num_valid_trials / self.max_trials
            progress = max(progress_no_overhead, progress)

        return progress
