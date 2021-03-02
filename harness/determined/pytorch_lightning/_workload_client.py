import contextlib
import logging
import pathlib
from typing import Any, Callable, Dict, Iterator, List, Tuple, cast

import determined as det
from determined import errors, util, workload
from determined_common import check

CheckpointFunc = Callable[[pathlib.Path], None]
ValidateFunc = Callable[[], None]


class WorkloadClient:
    """
    WorkloadClient helps any framework or library to interact with the Determined
    cluster within the callback under training on the fly.

    It has the following functionalities:

    - It fetches workloads and constructs workload responses.
    - It executes validation and checkpoint workloads.
    - It reduces training and validation metrics across processes.
    - It detects and handles early stopping and invalid HP exception.

    The frameworks or libraries are responsible for:

    - Calling `enter_training` as a context manager.
    - Calling `finish_train_batch`, `finish_validate_batch`, `finish_validate`,
      `finish_checkpoint`, and `finish_validate` in their callbacks.
    """

    def __init__(
        self,
        context: det.TrialContext,
        checkpoint_func: CheckpointFunc,
        validate_func: ValidateFunc,
    ):
        self.context = context

        self._workloads = context.workloads
        self._workload_iter = iter(self._workloads)
        self._fetch_next_workload()

        self.checkpoint_func = checkpoint_func
        self.validate_func = validate_func

        self.step_metrics = []  # type: List[Dict]
        self.val_metrics = []  # type: List[Dict]

        self.reduce_helper = det.MetricsReduceHelper(self.context)

    @contextlib.contextmanager
    def enter_training(self) -> Iterator:
        try:
            self._run_workloads()

            yield
        except det.errors.WorkerFinishedGracefully:
            return
        except det.InvalidHP as e:
            logging.info(f"Invalid hyperparameter exception in LightningTrialContext.fit: {e}")
            self._cur_response_func(
                util.wrap_metrics(
                    {},
                    self.context.get_stop_requested(),
                    invalid_hp=True,
                )
            )
            return

        try:
            self._handle_early_stopping()
        except det.errors.WorkerFinishedGracefully:
            return

    def _run_workloads(self) -> None:
        while True:
            w, args = self._cur_workload, self._cur_args
            if w.kind == workload.Workload.Kind.RUN_STEP:
                return
            elif w.kind == workload.Workload.Kind.CHECKPOINT_MODEL:
                check.eq(len(args), 1)
                check.is_instance(args[0], pathlib.Path)
                path = cast(pathlib.Path, args[0])
                self.checkpoint_func(path)
            elif w.kind == workload.Workload.Kind.COMPUTE_VALIDATION_METRICS:
                self.validate_func()
            elif w.kind == workload.Workload.Kind.TERMINATE:
                self._finish_terminate()
            else:
                raise AssertionError("Unexpected workload: {}".format(w.kind))

    def _handle_early_stopping(self) -> None:
        """
        Handle early stopping by responding with stop_requested and consuming the rest
        of the workloads until getting a checkpoint workload or a terminate workload.
        """
        self.context.set_stop_requested(True)
        w, args = self._cur_workload, self._cur_args
        while True:
            if w.kind == workload.Workload.Kind.RUN_STEP:
                self._finish_train_step()
            elif w.kind == workload.Workload.Kind.CHECKPOINT_MODEL:
                check.eq(len(args), 1)
                check.is_instance(args[0], pathlib.Path)
                path = cast(pathlib.Path, args[0])
                self.checkpoint_func(path)
                raise errors.WorkerFinishedGracefully("Exiting normally.")
            elif w.kind == workload.Workload.Kind.COMPUTE_VALIDATION_METRICS:
                self.finish_validate()
            elif w.kind == workload.Workload.Kind.TERMINATE:
                self._finish_terminate()
            else:
                raise AssertionError("Unexpected workload: {}".format(w.kind))

    def _fetch_next_workload(
        self,
    ) -> Tuple[workload.Workload, workload.Args, workload.ResponseFunc]:
        self._cur_workload, self._cur_args, self._cur_response_func = next(self._workload_iter)
        check.is_in(
            self._cur_workload.kind.name,
            workload.Workload.Kind.__members__.keys(),
            f"Unexpected workload: {self._cur_workload.kind}",
        )
        return self._cur_workload, self._cur_args, self._cur_response_func

    def _make_metric_response(
        self, metrics: List[Dict[str, Any]], train: bool = True
    ) -> workload.Response:
        """Reduces metrics across processes and construct the metric workload response."""

        # Reduce metrics across processes
        per_slot_metrics = self.reduce_helper.allgather_metrics(metrics)
        per_batch_all_slot_metrics = zip(*per_slot_metrics)
        reduced_metrics = []
        for all_slot_metrics in per_batch_all_slot_metrics:
            slot_reduced_metrics = util.make_metrics(
                None,
                list(all_slot_metrics),
                True,
            )["avg_metrics"]
            reduced_metrics.append(slot_reduced_metrics)

        # Construct the response for metrics.
        if self.context.distributed.is_chief():
            check.is_instance(reduced_metrics, List)
            response = cast(
                workload.Response,
                util.make_metrics(len(metrics), reduced_metrics, train),
            )
        else:
            response = workload.Skipped()

        return util.wrap_metrics(
            response,
            self.context.get_stop_requested(),
            invalid_hp=False,
        )

    def finish_train_batch(self, metrics: Dict[str, Any]) -> None:
        check.true(
            self._cur_workload.kind == workload.Workload.Kind.RUN_STEP,
            "Must call finish_train_batch in a RUN_STEP workload.",
        )

        # Check if we enter here on the right batch index.
        check.true(
            self.context._cur_total_batches <= self._cur_workload.total_batches()
            and self.context._cur_total_batches >= self._cur_workload.total_batches_processed,
            f"Current total batches should be within the workload: {self._cur_workload.__repr__()}",
        )

        # Add training metrics of the current batch for the future reporting.
        check.is_instance(
            metrics,
            dict,
            "reduced metrics must be a dictionary "
            f"mapping string names to Tensor metrics, got {type(metrics)}",
        )
        self.step_metrics.append(metrics)

        # Do nothing if not hit the end of current RUN_STEP workload yet.
        if self.context._cur_total_batches == self._cur_workload.total_batches():
            self._finish_train_step()

    def _finish_train_step(self) -> None:
        check.true(
            self._cur_workload.kind == workload.Workload.Kind.RUN_STEP,
            "Must call _finish_train_step in a RUN_STEP workload.",
        )

        self._cur_response_func(self._make_metric_response(self.step_metrics, True))
        self.step_metrics = []

        self._fetch_next_workload()
        self._run_workloads()

    def finish_validate_batch(self, metrics: Dict[str, Any]) -> None:
        # This might be called under a RUN_STEP workload if the validation is not initialized
        # by itself.
        if self._cur_workload.kind != workload.Workload.Kind.COMPUTE_VALIDATION_METRICS:
            return

        # Check if we enter here on the right batch index.
        check.true(
            self.context._cur_total_batches == self._cur_workload.total_batches(),
            f"Current total batches {self.context._cur_total_batches} should be "
            f"within the workload: {self._cur_workload.__repr__()}",
        )

        # Add training metrics of the current batch for the future reporting.
        check.is_instance(
            metrics,
            dict,
            "reduced metrics must be a dictionary "
            f"mapping string names to Tensor metrics, got {type(metrics)}",
        )
        self.val_metrics.append(metrics)

    def finish_validate(self) -> None:
        # This might be called under a RUN_STEP workload if the validation is not initialized
        # by itself.
        if self._cur_workload.kind != workload.Workload.Kind.COMPUTE_VALIDATION_METRICS:
            return

        self._cur_response_func(self._make_metric_response(self.val_metrics, False))
        self.val_metrics = []

        self._fetch_next_workload()

        # There is no need to call _run_workloads() here because the call of this function
        # must come from one iteration in the loop of _run_workloads.

    def finish_checkpoint(self, ckpt_resp: workload.Response) -> None:
        # This might be called under a RUN_STEP workload if the checkpoint is not initialized
        # by itself.
        if self._cur_workload.kind != workload.Workload.Kind.CHECKPOINT_MODEL:
            return

        if not self.context.distributed.is_chief():
            ckpt_resp = workload.Skipped()

        self._cur_response_func(ckpt_resp)
        self._fetch_next_workload()

        # There is no need to call _run_workloads() here because the call of this function
        # must come from one iteration in the loop of _run_workloads.

    def _finish_terminate(self) -> None:
        self._cur_response_func({} if self.context.distributed.is_chief() else workload.Skipped())
        raise errors.WorkerFinishedGracefully("Exiting normally.")
