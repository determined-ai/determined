import dataclasses
import json
import logging
import pickle
import random
import sys
import uuid
from pathlib import Path
from typing import Dict, List, Optional, Set

from urllib3.connectionpool import HTTPConnectionPool, MaxRetryError

from determined import searcher


class RandomSearchMethod(searcher.SearchMethod):
    def __init__(
        self,
        max_trials: int,
        max_concurrent_trials: int,
        max_length: int,
        test_type: str = "core_api",
        exception_points: Optional[List[str]] = None,
    ) -> None:
        self.max_trials = max_trials
        self.max_concurrent_trials = max_concurrent_trials
        self.max_length = max_length

        self.test_type = test_type
        self.exception_points = exception_points

        self.created_trials = 0
        self.pending_trials = 0
        self.closed_trials = 0

    def on_trial_created(
        self, _: searcher.SearcherState, request_id: uuid.UUID
    ) -> List[searcher.Operation]:
        self.raise_exception("on_trial_created")
        if self.created_trials == 5:
            self.raise_exception("on_trial_created_5")
        self._log_stats()
        return []

    def on_validation_completed(
        self, _: searcher.SearcherState, request_id: uuid.UUID, metric: float, train_length: int
    ) -> List[searcher.Operation]:
        self.raise_exception("on_validation_completed")
        return []

    def on_trial_closed(
        self, _: searcher.SearcherState, request_id: uuid.UUID
    ) -> List[searcher.Operation]:
        self.pending_trials -= 1
        self.closed_trials += 1
        ops: List[searcher.Operation] = []
        if self.created_trials < self.max_trials:
            request_id = uuid.uuid4()
            ops.append(
                searcher.Create(
                    request_id=request_id, hparams=self.sample_params(), checkpoint=None
                )
            )
            ops.append(searcher.ValidateAfter(request_id=request_id, length=self.max_length))
            ops.append(searcher.Close(request_id=request_id))
            self.created_trials += 1
            self.pending_trials += 1
        elif self.pending_trials == 0:
            self.raise_exception("on_trial_closed_shutdown")
            ops.append(searcher.Shutdown())

        self._log_stats()
        self.raise_exception("on_trial_closed_end")
        return ops

    def progress(self, searcher_state: searcher.SearcherState) -> float:
        if 0 < self.max_concurrent_trials < self.pending_trials:
            logging.error("pending trials is greater than max_concurrent_trial")
        units_completed = sum(
            (
                (
                    self.max_length
                    if r in searcher_state.trials_closed
                    else searcher_state.trial_progress[r]
                )
                for r in searcher_state.trial_progress
            )
        )
        units_expected = self.max_length * self.max_trials
        progress = units_completed / units_expected
        logging.debug(
            f"progress = {progress} = {units_completed} / {units_expected},"
            f" {searcher_state.trial_progress}"
        )

        if progress >= 0.5:
            self.raise_exception("progress_middle")

        return progress

    def on_trial_exited_early(
        self, _: searcher.SearcherState, request_id: uuid.UUID, exited_reason: searcher.ExitedReason
    ) -> List[searcher.Operation]:
        self.pending_trials -= 1

        ops: List[searcher.Operation] = []
        if exited_reason == searcher.ExitedReason.INVALID_HP:
            request_id = uuid.uuid4()
            ops.append(
                searcher.Create(
                    request_id=request_id, hparams=self.sample_params(), checkpoint=None
                )
            )
            ops.append(searcher.ValidateAfter(request_id=request_id, length=self.max_length))
            ops.append(searcher.Close(request_id=request_id))
            self.pending_trials += 1
            return ops

        self.closed_trials += 1
        self._log_stats()
        return ops

    def initial_operations(self, _: searcher.SearcherState) -> List[searcher.Operation]:
        self.raise_exception("initial_operations_start")
        initial_trials = self.max_trials
        max_concurrent_trials = self.max_concurrent_trials
        if max_concurrent_trials > 0:
            initial_trials = min(initial_trials, max_concurrent_trials)

        ops: List[searcher.Operation] = []

        for __ in range(initial_trials):
            create = searcher.Create(
                request_id=uuid.uuid4(),
                hparams=self.sample_params(),
                checkpoint=None,
            )
            ops.append(create)
            ops.append(searcher.ValidateAfter(request_id=create.request_id, length=self.max_length))
            ops.append(searcher.Close(request_id=create.request_id))

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
                "exception_points": self.exception_points,
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

            if self.test_type == "core_api":
                # ony restore exception points for core_api searcher tests;
                # local searcher is providing new exception point on resumption,
                # and it shouldn't be overridden
                self.exception_points = state["exception_points"]

    def raise_exception(self, exception_id: str) -> None:
        if (
            self.exception_points is not None
            and len(self.exception_points) > 0
            and exception_id == self.exception_points[0]
        ):
            logging.info(f"Raising exception in {exception_id}")
            ex = MaxRetryError(
                HTTPConnectionPool(host="dummyhost", port=8080),
                "http://dummyurl",
            )
            raise ex


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


class ASHASearchMethod(searcher.SearchMethod):
    def __init__(
        self,
        max_length: int,
        max_trials: int,
        num_rungs: int,
        divisor: int,
        test_type: str = "core_api",
        max_concurrent_trials: int = 0,
        exception_points: Optional[List[str]] = None,
    ) -> None:
        self.asha_search_state = ASHASearchMethodState(
            max_length, max_trials, num_rungs, divisor, max_concurrent_trials
        )
        self.test_type = test_type
        self.exception_points = exception_points

    def on_trial_closed(
        self, _: searcher.SearcherState, request_id: uuid.UUID
    ) -> List[searcher.Operation]:
        self.asha_search_state.completed_trials += 1
        self.asha_search_state.closed_trials.add(request_id)

        if (
            self.asha_search_state.pending_trials == 0
            and self.asha_search_state.completed_trials == self.asha_search_state.max_trials
        ):
            self.raise_exception("shutdown")
            return [searcher.Shutdown()]

        return []

    def on_trial_created(
        self, _: searcher.SearcherState, request_id: uuid.UUID
    ) -> List[searcher.Operation]:
        self.asha_search_state.rungs[0].outstanding_trials += 1
        self.asha_search_state.trial_rungs[request_id] = 0
        self.raise_exception("on_trial_created")
        return []

    def on_validation_completed(
        self, _: searcher.SearcherState, request_id: uuid.UUID, metric: float, train_length: int
    ) -> List[searcher.Operation]:
        self.asha_search_state.pending_trials -= 1
        if self.asha_search_state.is_smaller_better is False:
            metric *= -1
        ops = self.promote_async(request_id, metric)
        self.raise_exception("on_validation_completed")
        return ops

    def on_trial_exited_early(
        self, _: searcher.SearcherState, request_id: uuid.UUID, exited_reason: searcher.ExitedReason
    ) -> List[searcher.Operation]:
        self.asha_search_state.pending_trials -= 1
        if exited_reason == searcher.ExitedReason.INVALID_HP:
            ops: List[searcher.Operation] = []

            self.asha_search_state.early_exit_trials.add(request_id)
            ops.append(searcher.Close(request_id))
            self.asha_search_state.closed_trials.add(request_id)
            self.asha_search_state.invalid_trials += 1

            highest_rung_index = self.asha_search_state.trial_rungs[request_id]
            rung = self.asha_search_state.rungs[highest_rung_index]
            rung.outstanding_trials -= 1

            for rung_idx in range(0, highest_rung_index + 1):
                rung = self.asha_search_state.rungs[rung_idx]
                rung.metrics = list(filter(lambda x: x.request_id != request_id, rung.metrics))

            create = searcher.Create(
                request_id=uuid.uuid4(),
                hparams=self.sample_params(),
                checkpoint=None,
            )
            ops.append(create)
            ops.append(
                searcher.ValidateAfter(
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

    def initial_operations(self, _: searcher.SearcherState) -> List[searcher.Operation]:
        self.raise_exception("initial_operations_start")
        ops: List[searcher.Operation] = []

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

        for __ in range(0, max_concurrent_trials):
            create = searcher.Create(
                request_id=uuid.uuid4(),
                hparams=self.sample_params(),
                checkpoint=None,
            )
            ops.append(create)
            ops.append(
                searcher.ValidateAfter(
                    request_id=create.request_id,
                    length=self.asha_search_state.rungs[0].units_needed,
                )
            )

            self.asha_search_state.trial_rungs[create.request_id] = 0
            self.asha_search_state.pending_trials += 1

        return ops

    def promote_async(self, request_id: uuid.UUID, metric: float) -> List[searcher.Operation]:
        rung_idx = self.asha_search_state.trial_rungs[request_id]
        rung = self.asha_search_state.rungs[rung_idx]
        rung.outstanding_trials -= 1
        added_train_workload = False

        ops: List[searcher.Operation] = []

        if rung_idx == self.asha_search_state.num_rungs - 1:
            rung.metrics.append(TrialMetric(request_id=request_id, metric=metric))

            if request_id not in self.asha_search_state.early_exit_trials:
                self.raise_exception("promote_async_close_trials")
                ops.append(searcher.Close(request_id=request_id))
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
                    ops.append(searcher.ValidateAfter(promoted_request_id, units_needed))
                    added_train_workload = True
                    self.asha_search_state.pending_trials += 1
                else:
                    return self.promote_async(promoted_request_id, sys.float_info.max)

        all_trials = len(self.asha_search_state.trial_rungs) - self.asha_search_state.invalid_trials
        if not added_train_workload and all_trials < self.asha_search_state.max_trials:
            logging.info("Creating new trial instead of promoting")
            self.asha_search_state.pending_trials += 1

            create = searcher.Create(
                request_id=uuid.uuid4(),
                hparams=self.sample_params(),
                checkpoint=None,
            )
            ops.append(create)
            ops.append(
                searcher.ValidateAfter(
                    request_id=create.request_id,
                    length=self.asha_search_state.rungs[0].units_needed,
                )
            )
            self.asha_search_state.trial_rungs[create.request_id] = 0

        if len(self.asha_search_state.rungs[0].metrics) == self.asha_search_state.max_trials:
            ops.extend(self._get_close_rungs_ops())

        return ops

    def _get_close_rungs_ops(self) -> List[searcher.Operation]:
        self.raise_exception("_get_close_rungs_ops")
        ops: List[searcher.Operation] = []

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
                        ops.append(searcher.Close(trial_metric.request_id))
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

    def progress(self, _: searcher.SearcherState) -> float:
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

        exception_path = path.joinpath("exceptions")
        with exception_path.open("wb") as f:
            pickle.dump(self.exception_points, f)

    def load_method_state(self, path: Path) -> None:
        self.raise_exception("load_method_state")
        checkpoint_path = path.joinpath("method_state")
        with checkpoint_path.open("rb") as f:
            self.asha_search_state = pickle.load(f)

        if self.test_type == "core_api":
            # ony restore exception points for core_api searcher tests;
            # local searcher is providing new exception point on resumption,
            # and it shouldn't be overridden
            exception_path = path.joinpath("exceptions")
            with exception_path.open("rb") as f:
                self.exception_points = pickle.load(f)

    def raise_exception(self, exception_id: str) -> None:
        if (
            self.exception_points is not None
            and len(self.exception_points) > 0
            and exception_id == self.exception_points[0]
        ):
            logging.info(f"Raising exception in {exception_id}")
            ex = MaxRetryError(HTTPConnectionPool(host="dummyhost", port=8080), "http://dummyurl")
            raise ex
