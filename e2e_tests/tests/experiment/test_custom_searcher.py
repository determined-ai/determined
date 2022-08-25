import dataclasses
import json
import logging
import pickle
import random
import sys
import tempfile
import uuid
from pathlib import Path
from typing import Dict, List, Optional, Set

import pytest
from urllib3.connectionpool import HTTPConnectionPool, MaxRetryError

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
from determined.searcher.search_runner import LocalSearchRunner
from tests import config as conf


@pytest.mark.e2e_cpu
def test_run_custom_searcher_experiment(tmp_path: Path) -> None:
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
    search_method = SingleSearchMethod(config, 500)
    search_runner = LocalSearchRunner(search_method, tmp_path)
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
    max_length = 500

    with tempfile.TemporaryDirectory() as searcher_dir:
        search_method = RandomSearchMethod(max_trials, max_concurrent_trials, max_length)
        search_runner = LocalSearchRunner(search_method, Path(searcher_dir))
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


@pytest.mark.e2e_cpu_2a
@pytest.mark.parametrize(
    "exceptions",
    [
        ["initial_operations_start", "progress_middle", "on_trial_closed_shutdown"],
        ["on_validation_completed", "on_trial_closed_end", "on_trial_created_5"],
        ["on_trial_created", "save_method_state", "after_save"],
        [
            "on_trial_created",
            "save_method_state",
            "load_method_state",
            "after_save",
            "after_save",
            "on_validation_completed",
            "after_save",
            "save_method_state",
        ],
    ],
)
def test_resume_random_searcher_exp(exceptions: List[str]) -> None:
    config = conf.load_config(conf.fixtures_path("no_op/single.yaml"))
    config["searcher"] = {
        "name": "custom",
        "metric": "validation_error",
        "smaller_is_better": True,
        "unit": "batches",
    }
    config["description"] = ";".join(exceptions) if exceptions else "custom searcher"

    max_trials = 5
    max_concurrent_trials = 2
    max_length = 500
    failures_expected = len(exceptions)
    logging.info(f"expected_failures={failures_expected}")

    # do not use pytest tmp_path to experience LocalSearchRunner in the wild
    with tempfile.TemporaryDirectory() as searcher_dir:
        logging.info(f"searcher_dir type = {type(searcher_dir)}")
        failures = 0
        while failures < failures_expected:
            try:
                exception_point = exceptions.pop(0)
                # re-create RandomSearchMethod and LocalSearchRunner after every fail
                # to simulate python process crash
                search_method = RandomSearchMethod(
                    max_trials, max_concurrent_trials, max_length, exception_point
                )
                search_runner_mock = FallibleSearchRunner(
                    exception_point, search_method, Path(searcher_dir)
                )
                search_runner_mock.run(config, context_dir=conf.fixtures_path("no_op"))
                pytest.fail("Expected an exception")
            except MaxRetryError:
                failures += 1

        assert failures == failures_expected

        search_method = RandomSearchMethod(max_trials, max_concurrent_trials, max_length)
        search_runner = LocalSearchRunner(search_method, Path(searcher_dir))
        experiment_id = search_runner.run(config, context_dir=conf.fixtures_path("no_op"))

    assert search_method.searcher_state.last_event_id == 41
    assert search_method.searcher_state.experiment_completed is True
    assert client._determined is not None
    session = client._determined._session
    response = bindings.get_GetExperiment(session, experimentId=experiment_id)
    assert response.experiment.numTrials == 5
    assert search_method.created_trials == 5
    assert search_method.pending_trials == 0
    assert search_method.closed_trials == 5
    assert len(search_method.searcher_state.trials_created) == search_method.created_trials
    assert len(search_method.searcher_state.trials_closed) == search_method.closed_trials

    assert search_method.progress() == pytest.approx(1.0)


class RandomSearchMethod(SearchMethod):
    def __init__(
        self,
        max_trials: int,
        max_concurrent_trials: int,
        max_length: int,
        exception_point: Optional[str] = None,
    ) -> None:
        super().__init__()
        self.max_trials = max_trials
        self.max_concurrent_trials = max_concurrent_trials
        self.max_length = max_length

        self.exception_point = exception_point

        # TODO remove created_trials and closed_trials before merging the feature branch
        self.created_trials = 0
        self.pending_trials = 0
        self.closed_trials = 0

    def on_trial_created(self, request_id: uuid.UUID) -> List[Operation]:
        self.raise_exception("on_trial_created")
        if self.created_trials == 5:
            self.raise_exception("on_trial_created_5")
        self._log_stats()
        return []

    def on_validation_completed(self, request_id: uuid.UUID, metric: float) -> List[Operation]:
        self.raise_exception("on_validation_completed")
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
            self.raise_exception("on_trial_closed_shutdown")
            ops.append(Shutdown())

        self._log_stats()
        self.raise_exception("on_trial_closed_end")
        return ops

    def progress(self) -> float:
        if 0 < self.max_concurrent_trials < self.pending_trials:
            logging.error("pending trials is greater than max_concurrent_trial")
        units_completed = sum(
            (
                (
                    self.max_length
                    if r in self.searcher_state.trials_closed
                    else self.searcher_state.trial_progress[r]
                )
                for r in self.searcher_state.trial_progress
            )
        )
        units_expected = self.max_length * self.max_trials
        progress = units_completed / units_expected
        logging.debug(
            f"progress = {progress} = {units_completed} / {units_expected},"
            f" {self.searcher_state.trial_progress}"
        )

        if progress >= 0.5:
            self.raise_exception("progress_middle")

        return progress

    def on_trial_exited_early(
        self, request_id: uuid.UUID, exit_reason: ExitedReason
    ) -> List[Operation]:
        self.pending_trials -= 1

        ops: List[Operation] = []
        if exit_reason == ExitedReason.INVALID_HP:
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
        self.raise_exception("initial_operations_start")
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

    def save_method_state(self, path: Path) -> None:
        self.raise_exception("save_method_state")
        checkpoint_path = path.joinpath("method_state")
        with checkpoint_path.open("w") as f:
            state = {
                "max_trials": self.max_trials,
                "max_concurrent_trials": self.max_concurrent_trials,
                "max_length": self.max_length,
                "created_trials": self.created_trials,
                "pending_trials": self.pending_trials,
                "closed_trials": self.closed_trials,
            }
            json.dump(state, f)

    def load_method_state(self, path: Path) -> None:
        self.raise_exception("load_method_state")
        checkpoint_path = path.joinpath("method_state")
        with checkpoint_path.open("r") as f:
            state = json.load(f)
            self.max_trials = state["max_trials"]
            self.max_concurrent_trials = state["max_concurrent_trials"]
            self.max_length = state["max_length"]
            self.created_trials = state["created_trials"]
            self.pending_trials = state["pending_trials"]
            self.closed_trials = state["closed_trials"]

    def raise_exception(self, exception_id: str) -> None:
        if exception_id == self.exception_point:
            logging.info(f"Raising exception in {exception_id}")
            ex = MaxRetryError(
                HTTPConnectionPool(host="dummyhost", port=8080),
                "http://dummyurl",
            )
            raise ex


@pytest.mark.e2e_cpu
def test_run_asha_batches_exp(tmp_path: Path) -> None:
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
    search_runner = LocalSearchRunner(search_method, tmp_path)
    experiment_id = search_runner.run(config, context_dir=conf.fixtures_path("no_op"))

    assert client._determined is not None
    session = client._determined._session
    response = bindings.get_GetExperiment(session, experimentId=experiment_id)

    assert response.experiment.numTrials == 16
    assert search_method.asha_search_state.pending_trials == 0
    assert search_method.asha_search_state.completed_trials == 16
    assert len(search_method.searcher_state.trials_closed) == len(
        search_method.asha_search_state.closed_trials
    )

    response_trials = bindings.get_GetExperimentTrials(session, experimentId=experiment_id).trials

    # 16 trials in rung 1 (#batches = 187)
    assert sum([t.totalBatchesProcessed >= 187 for t in response_trials]) == 16
    # at least 4 trials in rung 2 (#batches = 750)
    assert sum([t.totalBatchesProcessed >= 750 for t in response_trials]) >= 4
    # at least 1 trial in rung 3 (#batches = 3000)
    assert sum([t.totalBatchesProcessed == 3000 for t in response_trials]) >= 1

    for trial in response_trials:
        assert trial.state == bindings.determinedexperimentv1State.STATE_COMPLETED


@pytest.mark.e2e_cpu
@pytest.mark.parametrize(
    "exceptions",
    [
        [
            "initial_operations_start",  # fail before sending initial operations
            "after_save",  # fail on save - should not send initial operations again
            "save_method_state",
            "save_method_state",
            "after_save",
            "on_trial_created_10_trials_in_rung_0",
            "_get_close_rungs_ops",
        ],
        [  # searcher state and search method state are restored to last saved state
            "on_validation_completed",
            "on_validation_completed",
            "save_method_state",
            "save_method_state",
            "after_save",
            "after_save",
            "load_method_state",
            "on_validation_completed",
            "shutdown",
        ],
    ],
)
def test_resume_asha_batches_exp(exceptions: List[str]) -> None:
    config = conf.load_config(conf.fixtures_path("no_op/adaptive.yaml"))
    config["searcher"] = {
        "name": "custom",
        "metric": "validation_error",
        "smaller_is_better": True,
        "unit": "batches",
    }
    config["name"] = "asha"
    config["description"] = ";".join(exceptions) if exceptions else "custom searcher"

    max_length = 3000
    max_trials = 16
    num_rungs = 3
    divisor = 4
    failures_expected = len(exceptions)

    with tempfile.TemporaryDirectory() as searcher_dir:
        logging.info(f"searcher_dir type = {type(searcher_dir)}")
        failures = 0
        while failures < failures_expected:
            try:
                exception_point = exceptions.pop(0)
                search_method = ASHASearchMethod(
                    max_length, max_trials, num_rungs, divisor, exception_point=exception_point
                )
                search_runner_mock = FallibleSearchRunner(
                    exception_point, search_method, Path(searcher_dir)
                )
                search_runner_mock.run(config, context_dir=conf.fixtures_path("no_op"))
                pytest.fail("Expected an exception")
            except MaxRetryError:
                failures += 1

        assert failures == failures_expected

        search_method = ASHASearchMethod(max_length, max_trials, num_rungs, divisor)
        search_runner = LocalSearchRunner(search_method, Path(searcher_dir))
        experiment_id = search_runner.run(config, context_dir=conf.fixtures_path("no_op"))

    assert search_method.searcher_state.experiment_completed is True
    assert client._determined is not None
    session = client._determined._session
    response = bindings.get_GetExperiment(session, experimentId=experiment_id)

    assert response.experiment.numTrials == 16
    # asha search method state
    assert search_method.asha_search_state.pending_trials == 0
    assert search_method.asha_search_state.completed_trials == 16
    # searcher state
    assert len(search_method.searcher_state.trials_created) == 16
    assert len(search_method.searcher_state.trials_closed) == 16

    assert len(search_method.searcher_state.trials_closed) == len(
        search_method.asha_search_state.closed_trials
    )

    response_trials = bindings.get_GetExperimentTrials(session, experimentId=experiment_id).trials

    # 16 trials in rung 1 (#batches = 187)
    assert sum([t.totalBatchesProcessed >= 187 for t in response_trials]) == 16
    # at least 4 trials in rung 2 (#batches = 750)
    assert sum([t.totalBatchesProcessed >= 750 for t in response_trials]) >= 4
    # at least 1 trial in rung 3 (#batches = 3000)
    assert sum([t.totalBatchesProcessed == 3000 for t in response_trials]) >= 1

    for trial in response_trials:
        assert trial.state == bindings.determinedexperimentv1State.STATE_COMPLETED

    assert search_method.progress() == pytest.approx(1.0)


@dataclasses.dataclass
class TrialMetric:
    request_id: uuid.UUID
    metric: float
    promoted: bool = False


@dataclasses.dataclass
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


class ASHASearchMethodState:
    def __init__(
        self,
        max_length: int,
        max_trials: int,
        num_rungs: int,
        divisor: int,
        max_concurrent_trials: int = 0,
    ) -> None:
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


class ASHASearchMethod(SearchMethod):
    def __init__(
        self,
        max_length: int,
        max_trials: int,
        num_rungs: int,
        divisor: int,
        max_concurrent_trials: int = 0,
        exception_point: Optional[str] = None,
    ) -> None:
        super().__init__()
        self.asha_search_state = ASHASearchMethodState(
            max_length, max_trials, num_rungs, divisor, max_concurrent_trials
        )
        self.exception_point = exception_point

    def on_trial_closed(self, request_id: uuid.UUID) -> List[Operation]:
        self.asha_search_state.completed_trials += 1
        self.asha_search_state.closed_trials.add(request_id)

        if (
            self.asha_search_state.pending_trials == 0
            and self.asha_search_state.completed_trials == self.asha_search_state.max_trials
        ):
            self.raise_exception("shutdown")
            return [Shutdown()]

        return []

    def on_trial_created(self, request_id: uuid.UUID) -> List[Operation]:
        self.asha_search_state.rungs[0].outstanding_trials += 1
        self.asha_search_state.trial_rungs[request_id] = 0
        if len(self.asha_search_state.rungs[0].metrics) == 10:
            self.raise_exception("on_trial_created_10_trials_in_rung_0")
        return []

    def on_validation_completed(self, request_id: uuid.UUID, metric: float) -> List[Operation]:
        self.asha_search_state.pending_trials -= 1
        if self.asha_search_state.is_smaller_better is False:
            metric *= -1
        ops = self.promote_async(request_id, metric)
        self.raise_exception("on_validation_completed")
        return ops

    def on_trial_exited_early(
        self, request_id: uuid.UUID, exited_reason: ExitedReason
    ) -> List[Operation]:
        self.pending_trials -= 1
        if exited_reason == ExitedReason.INVALID_HP:
            ops: List[Operation] = []

            self.asha_search_state.early_exit_trials.add(request_id)
            ops.append(Close(request_id))
            self.asha_search_state.closed_trials.add(request_id)
            self.asha_search_state.invalid_trials += 1

            highest_rung_index = self.asha_search_state.trial_rungs[request_id]
            rung = self.asha_search_state.rungs[highest_rung_index]
            rung.outstanding_trials -= 1

            for rung_idx in range(0, highest_rung_index + 1):
                rung = self.asha_search_state.rungs[rung_idx]
                rung.metrics = list(filter(lambda x: x.request_id != request_id, rung.metrics))

            create = Create(
                request_id=uuid.uuid4(),
                hparams=self.sample_params(),
                checkpoint=None,
            )
            ops.append(create)
            ops.append(
                ValidateAfter(
                    request_id=create.request_id,
                    length=self.asha_search_state.rungs[0].units_needed,
                )
            )

            self.asha_search_state.trial_rungs[create.request_id] = 0
            self.asha_search_state.pending_trials += 1

            return ops

        self.asha_search_state.early_exit_trials.add(request_id)
        self.asha_search_state.closed_trials.add(request_id)
        return self.promote_async(request_id, sys.float_info.max)

    def initial_operations(self) -> List[Operation]:
        self.raise_exception("initial_operations_start")
        ops: List[Operation] = []

        if self.asha_search_state.max_concurrent_trials > 0:
            max_concurrent_trials = min(
                self.asha_search_state.max_concurrent_trials, self.asha_search_state.max_trials
            )
        else:
            max_concurrent_trials = max(
                1,
                min(
                    int(pow(self.asha_search_state.divisor, self.asha_search_state.num_rungs - 1)),
                    self.asha_search_state.max_trials,
                ),
            )

        for _ in range(0, max_concurrent_trials):
            create = Create(
                request_id=uuid.uuid4(),
                hparams=self.sample_params(),
                checkpoint=None,
            )
            ops.append(create)
            ops.append(
                ValidateAfter(
                    request_id=create.request_id,
                    length=self.asha_search_state.rungs[0].units_needed,
                )
            )

            self.asha_search_state.trial_rungs[create.request_id] = 0
            self.asha_search_state.pending_trials += 1

        return ops

    def promote_async(self, request_id: uuid.UUID, metric: float) -> List[Operation]:
        rung_idx = self.asha_search_state.trial_rungs[request_id]
        rung = self.asha_search_state.rungs[rung_idx]
        rung.outstanding_trials -= 1
        added_train_workload = False

        ops: List[Operation] = []

        if rung_idx == self.asha_search_state.num_rungs - 1:
            rung.metrics.append(TrialMetric(request_id=request_id, metric=metric))

            if request_id not in self.asha_search_state.early_exit_trials:
                self.raise_exception("promote_async_close_trials")
                ops.append(Close(request_id=request_id))
                logging.info(f"Closing trial {request_id}")
                self.asha_search_state.closed_trials.add(request_id)
        else:
            next_rung = self.asha_search_state.rungs[rung_idx + 1]
            self.raise_exception("promote_async")
            logging.info(f"Promoting in rung {rung_idx}")
            for promoted_request_id in rung.promotions_async(
                request_id, metric, self.asha_search_state.divisor
            ):
                self.asha_search_state.trial_rungs[promoted_request_id] = rung_idx + 1
                next_rung.outstanding_trials += 1
                if promoted_request_id not in self.asha_search_state.early_exit_trials:
                    logging.info(f"Promoted {promoted_request_id}")
                    units_needed = max(next_rung.units_needed - rung.units_needed, 1)
                    ops.append(ValidateAfter(promoted_request_id, units_needed))
                    added_train_workload = True
                    self.asha_search_state.pending_trials += 1
                else:
                    return self.promote_async(promoted_request_id, sys.float_info.max)

        all_trials = len(self.asha_search_state.trial_rungs) - self.asha_search_state.invalid_trials
        if not added_train_workload and all_trials < self.asha_search_state.max_trials:
            logging.info("Creating new trial instead of promoting")
            self.asha_search_state.pending_trials += 1

            create = Create(
                request_id=uuid.uuid4(),
                hparams=self.sample_params(),
                checkpoint=None,
            )
            ops.append(create)
            ops.append(
                ValidateAfter(
                    request_id=create.request_id,
                    length=self.asha_search_state.rungs[0].units_needed,
                )
            )
            self.asha_search_state.trial_rungs[create.request_id] = 0

        if len(self.asha_search_state.rungs[0].metrics) == self.asha_search_state.max_trials:
            ops.extend(self._get_close_rungs_ops())

        return ops

    def _get_close_rungs_ops(self) -> List[Operation]:
        self.raise_exception("_get_close_rungs_ops")
        ops: List[Operation] = []

        for rung in self.asha_search_state.rungs:
            if rung.outstanding_trials > 0:
                break
            for trial_metric in rung.metrics:
                if (
                    not trial_metric.promoted
                    and trial_metric.request_id not in self.asha_search_state.closed_trials
                ):
                    if trial_metric.request_id not in self.asha_search_state.early_exit_trials:
                        logging.info(f"Closing trial {trial_metric.request_id}")
                        ops.append(Close(trial_metric.request_id))
                        self.asha_search_state.closed_trials.add(trial_metric.request_id)
        return ops

    def sample_params(self) -> Dict[str, object]:
        hparams = {
            "global_batch_size": 10,
            "metrics_base": 0.05 * (len(self.asha_search_state.trial_rungs) + 1),
            "metrics_progression": "constant",
        }
        logging.info(f"hparams={hparams}")
        return hparams

    def progress(self) -> float:
        if 0 < self.asha_search_state.max_concurrent_trials < self.asha_search_state.pending_trials:
            raise RuntimeError("Pending trial is greater than max concurrent trials")
        all_trials = len(self.asha_search_state.rungs[0].metrics)

        progress = all_trials / (1.2 * self.asha_search_state.max_trials)
        if all_trials == self.asha_search_state.max_trials:
            num_valid_trials = (
                self.asha_search_state.completed_trials - self.asha_search_state.invalid_trials
            )
            progress_no_overhead = num_valid_trials / self.asha_search_state.max_trials
            progress = max(progress_no_overhead, progress)

        return progress

    def save_method_state(self, path: Path) -> None:
        self.raise_exception("save_method_state")
        checkpoint_path = path.joinpath("method_state")
        with checkpoint_path.open("wb") as f:
            pickle.dump(self.asha_search_state, f)

    def load_method_state(self, path: Path) -> None:
        self.raise_exception("load_method_state")
        checkpoint_path = path.joinpath("method_state")
        with checkpoint_path.open("rb") as f:
            self.asha_search_state = pickle.load(f)

    def raise_exception(self, exception_id: str) -> None:
        if exception_id == self.exception_point:
            logging.info(f"Raising exception in {exception_id}")
            ex = MaxRetryError(HTTPConnectionPool(host="dummyhost", port=8080), "http://dummyurl")
            raise ex


class FallibleSearchRunner(LocalSearchRunner):
    def __init__(
        self,
        exception_point: str,
        search_method: SearchMethod,
        searcher_dir: Optional[Path] = None,
    ):
        super(FallibleSearchRunner, self).__init__(search_method, searcher_dir)
        self.fail_on_save = False
        if exception_point == "after_save":
            self.fail_on_save = True

    def save_state(self, experiment_id: int, operations: List[Operation]) -> None:
        super(FallibleSearchRunner, self).save_state(experiment_id, operations)
        if self.fail_on_save:
            logging.info(
                "Raising exception in after saving the state and before posting operations"
            )
            ex = MaxRetryError(HTTPConnectionPool(host="dummyhost", port=8080), "http://dummyurl")
            raise ex
