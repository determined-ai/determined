import logging
import math
import sys
from typing import Any, Generator, Optional, Tuple

import determined as det
from determined import _generic, tensorboard, workload
from determined.common import check, constants
from determined.experimental import client

WorkloadStreamElem = Tuple[workload.Workload, workload.ResponseFunc]

WorkloadGenerator = Generator[WorkloadStreamElem, None, None]


def yield_and_await_response(
    wkld: workload.Workload,
) -> Generator[WorkloadStreamElem, None, workload.Response]:
    """
    rb: I didn't know that generators could return meaningful values when I designed the layers
    abstraction of the harness.  If I had, I would have used it all over, most likely in place of
    the response_func.

    yield_and_await_response is a convenience function that yields a value and a response func, then
    returns whatever got passed in the response func.

    It's not worth refactoring all of the layers of the harness to use this pattern because the
    whole harness is getting refactored with push architecture, and the layers will be a thing of
    the past.
    """
    out: Optional[workload.Response] = None

    def respond(r: workload.Response) -> None:
        nonlocal out
        out = r

    yield wkld, respond

    assert out is not None

    return out


class ShouldExit(Exception):
    """
    ShouldExit breaks out of the top-level workload sequencer loop from inside function calls.
    """

    def __init__(self, skip_exit_checkpoint: bool = False):
        self.skip_exit_checkpoint = skip_exit_checkpoint

    pass


class WorkloadSequencer(workload.Source):
    """
    WorkloadSequencer is the python rewrite of the old golang
    TrialWorkloadSequencer.

    Like the go version, it fuses the dual stream of searcher operations +
    descheduling decisions into a single stream of Workload events.

    When the sequencer was in the master, the resulting stream of Workloads was
    the basis for all master/harness communications, but now that the sequencer
    lives in the harness, all master/harness communications are over the new
    push APIs.

    This Workoad stream (and the whole WorkloadSequencer) is only even needed
    for reverse-compatibility with the old TrialControllers that we don't care
    to update (TFKerasTrial and EstimatorTrial).
    """

    class SavableState:
        def __init__(
            self,
            trial_id: int,
            last_ckpt: int = 0,
            latest_batch: int = 0,
            step_id: int = 0,
            last_val: int = 0,
        ) -> None:
            # Store TrialID to distinguish between e.g. pause/restart and continue training.
            self.trial_id = trial_id
            self.last_ckpt = last_ckpt
            self.latest_batch = latest_batch
            self.step_id = step_id
            self.last_val = last_val

    def __init__(
        self,
        env: det.EnvContext,
        session: client.Session,
        dist: det.DistributedContext,
    ) -> None:
        self.env = env
        self.session = session
        self._dist = dist
        self._run_id = env.trial_run_id
        self._trial_id = int(env.det_trial_id)
        self._allocation_id = env.allocation_id
        self._exp_id = int(env.det_experiment_id)

        tbd_mgr = tensorboard.build(
            env.det_cluster_id,
            env.det_experiment_id,
            env.det_trial_id,
            env.experiment_config["checkpoint_storage"],
            container_path=None if not env.on_cluster else constants.SHARED_FS_CONTAINER_PATH,
        )
        tbd_writer = tensorboard.get_metric_writer()
        self.training = _generic.Training(
            session, self._trial_id, self._run_id, self._exp_id, tbd_mgr, tbd_writer
        )

        api_path = f"/api/v1/trials/{self._trial_id}/checkpoint_metadata"
        static_metadata = {"trial_id": self._trial_id, "trial_run_id": self._run_id}
        self.checkpointing = _generic.Checkpointing(session, api_path, static_metadata, tbd_mgr)

        self.val_from_previous_run = self.training.get_last_validation()

        self.want_initial_val = self.env.experiment_config.get("perform_initial_validation", False)

        self.ckpt_policy = self.env.experiment_config.get("checkpoint_policy", "best")

        self.state = self.SavableState(trial_id=self._trial_id)

        # precalculated periods, in batches
        self.records_per_epoch = env.experiment_config.get_records_per_epoch()
        self.global_batch_size = env.global_batch_size
        self.min_val_period_batches = self.as_batches(
            **env.experiment_config.get_min_validation_period()
        )
        self.min_ckpt_period_batches = self.as_batches(
            **env.experiment_config.get_min_checkpoint_period()
        )
        if self.min_val_period_batches < 1:
            self.min_val_period_batches = sys.maxsize
        if self.min_ckpt_period_batches < 1:
            self.min_ckpt_period_batches = sys.maxsize

    def get_state(self) -> Any:
        return vars(self.state)

    def load_state(self, state: Any) -> None:
        # Load our state from the checkpoint if we are continuing training after a pause or restart.
        # If the trial_id doesn't match our current trial id, we're continuing training a previous
        # trial and the state in the checkpoint should be discarded.
        if state.get("trial_id") != self._trial_id:
            return

        self.state = self.SavableState(**state)

        # Detect the case where the final validation we made was against this exact checkpoint.  In
        # that case, the master will know about the validation, but it would not appear in the
        # checkpoint state.  If the validation was before the last checkpoint, the checkpoint state
        # is already correct, while any validations after the last checkpoint aren't valid anymore
        # and can be safely ignored.
        if self.state.latest_batch == self.val_from_previous_run:
            self.state.last_val = self.state.latest_batch

    def as_batches(
        self,
        batches: Optional[int] = None,
        records: Optional[int] = None,
        epochs: Optional[int] = None,
    ) -> int:
        if sum((batches is not None, records is not None, epochs is not None)) != 1:
            raise ValueError(f"invalid length: batches={batches} records={records} epochs={epochs}")
        if batches is not None:
            return batches
        if records is not None:
            check.gt(self.global_batch_size, 0, "global_batch_size must be positive")
            return max(records // self.global_batch_size, 1)
        if epochs is not None:
            check.is_instance(self.records_per_epoch, int, "length must be an integer")
            assert self.records_per_epoch is not None
            check.gt(self.global_batch_size, 0, "global_batch_size must be positive")
            return max((epochs * self.records_per_epoch) // self.global_batch_size, 1)
        # Make mypy happy.
        raise ValueError("invalid length")

    def check_for_preemption(self) -> None:
        assert self.preemption is not None
        if self.preemption.should_preempt(chief_only=True):
            raise ShouldExit()

    def train(self, num_batches: int, op: _generic.SearcherOp) -> WorkloadGenerator:
        # Report a train step is starting.
        self.training.set_status("training")

        wkld = workload.Workload(
            kind=workload.Workload.Kind.RUN_STEP,
            e_id=self._exp_id,
            t_id=self._trial_id,
            s_id=self.state.step_id + 1,
            num_batches=num_batches,
            total_batches_processed=self.state.latest_batch,
        )

        response = yield from yield_and_await_response(wkld)

        # Train step is complete, process the result.

        if isinstance(response, workload.InvalidHP):
            # Exit before reporting metrics (which would be empty anyway).
            self.training.report_early_exit(_generic.EarlyExitReason.INVALID_HP)
            raise ShouldExit()

        metrics = response.get("metrics", {}).get("avg_metrics", {})
        batch_metrics = response.get("metrics", {}).get("batch_metrics", [])

        self.state.latest_batch += num_batches
        self.state.step_id += 1
        self.training.report_training_metrics(
            latest_batch=self.state.latest_batch,
            metrics=metrics,
            batch_metrics=batch_metrics,
        )

        # Report progress to the searcher.  For historical reasons we only deal in batches.
        if op.unit == _generic.Unit.BATCHES:
            op.report_progress(self.state.latest_batch)
        elif op.unit == _generic.Unit.RECORDS:
            op.report_progress(self.global_batch_size * self.state.latest_batch)
        elif op.unit == _generic.Unit.EPOCHS:
            op.report_progress(self.state.latest_batch / self.as_batches(epochs=1))
        else:
            raise ValueError(f"unrecognized searcher op unit: {op.unit}")

        if response.get("stop_requested"):
            # Exit after reporting metrics.
            raise ShouldExit()

        self.check_for_preemption()

    def is_best_validation(self, now: float, before: Optional[float]) -> bool:
        if before is None:
            return True
        smaller_is_better = self.env.experiment_config["searcher"]["smaller_is_better"]
        return (now < before) if smaller_is_better else (now > before)

    def validate(self, op: Optional[_generic.SearcherOp]) -> WorkloadGenerator:
        # Report a validation step is starting.
        self.training.set_status("validating")

        wkld = workload.Workload(
            kind=workload.Workload.Kind.COMPUTE_VALIDATION_METRICS,
            e_id=self._exp_id,
            t_id=self._trial_id,
            s_id=self.state.step_id,
            num_batches=0,
            total_batches_processed=self.state.latest_batch,
        )

        response = yield from yield_and_await_response(wkld)

        # Validation step is complete, process the result.

        if isinstance(response, workload.InvalidHP):
            self.training.report_early_exit(_generic.EarlyExitReason.INVALID_HP)
            raise ShouldExit()

        metrics = response["metrics"]["validation_metrics"]

        # Check that the validation metrics computed by the model code
        # includes the metric used by the search method.
        searcher_metric_name = self.env.experiment_config["searcher"]["metric"]
        if searcher_metric_name not in metrics:
            raise RuntimeError(
                f"Search method is configured to use metric '{searcher_metric_name}' but model "
                f"definition returned validation metrics {list(metrics.keys())}. The metric "
                "used by the search method must be one of the validation "
                "metrics returned by the model definition."
            )

        # Check that the searcher metric has a scalar value so that it can be compared for
        # search purposes. Other metrics don't have to be scalars.
        searcher_metric = metrics[searcher_metric_name]
        if not tensorboard.metric_writers.util.is_numerical_scalar(searcher_metric):
            raise RuntimeError(
                f"Searcher validation metric '{searcher_metric_name}' returned "
                f"a non-scalar value: {searcher_metric}"
            )

        if math.isnan(searcher_metric):
            raise RuntimeError(
                f"Searcher validation metric '{searcher_metric_name}' returned a NaN value"
            )

        # Report to the searcher API first, so we don't end up in a situation where we die between
        # reporting to the metrics API and when we come back we refuse to repeat a validation, but
        # we also don't have any validation metrics to report the the searcher API.
        #
        # A simpler solution here would be to execute in the following order (which would be
        # suitable for most customers to implement on their own):
        #   - validation
        #   - report to metrics API
        #   - report to searcher API
        #   - checkpoint
        #
        # But we can't do that without breaking behavior.
        if op is not None and self.batches_until_op_complete(op) < 1:
            op.complete(searcher_metric)

        if self.ckpt_policy == "best" and not self.checkpoint_is_current():
            # Before reporting our own validation metric, check what the best known validation is
            # without it.
            best_validation_before = self.training.get_experiment_best_validation()

        self.state.last_val = self.state.latest_batch
        self.training.report_validation_metrics(
            latest_batch=self.state.latest_batch,
            metrics=metrics,
        )

        if response.get("stop_requested"):
            raise ShouldExit()

        if not self.checkpoint_is_current():
            if self.ckpt_policy == "all" or (
                self.ckpt_policy == "best"
                and self.is_best_validation(now=searcher_metric, before=best_validation_before)
            ):
                yield from self.checkpoint(already_exiting=False)

        self.check_for_preemption()

    def checkpoint(self, already_exiting: bool) -> WorkloadGenerator:
        self.training.set_status("checkpointing")

        # Update the last_ckpt now so it can be captured by get_state() after we yield.
        self.state.last_ckpt = self.state.latest_batch

        wkld = workload.Workload(
            kind=workload.Workload.Kind.CHECKPOINT_MODEL,
            e_id=self._exp_id,
            t_id=self._trial_id,
            s_id=self.state.step_id,
            num_batches=0,
            total_batches_processed=self.state.latest_batch,
        )
        response = yield from yield_and_await_response(wkld)

        if isinstance(response, workload.InvalidHP):
            self.training.report_early_exit(_generic.EarlyExitReason.INVALID_HP)
            if not already_exiting:
                raise ShouldExit(skip_exit_checkpoint=True)
            return

        uuid = response["uuid"]

        logging.info(f"Saved trial to checkpoint {uuid}")

        self.checkpointing._report_checkpoint(
            uuid=uuid,
            resources=response["resources"],
            metadata={
                "framework": response.get("framework", ""),
                "format": response.get("format", ""),
                "latest_batch": self.state.latest_batch,
            },
        )

        if already_exiting:
            return

        if response.get("stop_requested"):
            raise ShouldExit()

        self.check_for_preemption()

    def batches_until_val(self) -> int:
        return self.state.last_val + self.min_val_period_batches - self.state.latest_batch

    def batches_until_ckpt(self) -> int:
        return self.state.last_ckpt + self.min_ckpt_period_batches - self.state.latest_batch

    def batches_until_op_complete(self, op: _generic.SearcherOp) -> int:
        return (
            self.as_batches(
                batches=op.length if op.unit == _generic.Unit.BATCHES else None,
                records=op.length if op.unit == _generic.Unit.RECORDS else None,
                epochs=op.length if op.unit == _generic.Unit.EPOCHS else None,
            )
            - self.state.latest_batch
        )

    def checkpoint_is_current(self) -> bool:
        return self.state.last_ckpt == self.state.latest_batch

    def validation_is_current(self) -> bool:
        return self.state.last_val == self.state.latest_batch

    def __iter__(self) -> workload.Stream:
        self.preemption = _generic.Preemption(self.session, self._allocation_id, self._dist)
        self.preemption.start()
        try:
            searcher = _generic.AdvancedSearcher(
                self.session, self._trial_id, self._run_id, self._allocation_id
            )

            # Step-zero Validations.
            if (
                self.want_initial_val
                and self.val_from_previous_run is None
                and self.state.latest_batch == 0
            ):
                yield from self.validate(None)

            for op in searcher.ops():
                while self.batches_until_op_complete(op) > 0:
                    # Do some training.
                    yield from self.train(
                        max(
                            1,
                            min(
                                self.batches_until_ckpt(),
                                self.batches_until_val(),
                                self.batches_until_op_complete(op),
                                self.env.experiment_config.scheduling_unit(),
                            ),
                        ),
                        op,
                    )

                    # Note: for historical compatibility we do validate-then-checkpoint here but
                    # when doing searcher-requested validations, we do checkpoint-then-validate.
                    # Most users preferred validate-then-checkpoint, but the distributed snapshot
                    # master restart logic required checkpoint-then-validate for searcher
                    # validations.

                    # Pause training to validate?
                    if self.batches_until_val() < 1:
                        yield from self.validate(op)

                    # Pause training to checkpoint?
                    if self.batches_until_ckpt() < 1:
                        yield from self.checkpoint(already_exiting=False)

                # Done training for this searcher operation!

                if not self.checkpoint_is_current():
                    yield from self.checkpoint(already_exiting=False)

                if not self.validation_is_current():
                    yield from self.validate(op)

                assert op._completed, "logic error; op was never completed"

        except ShouldExit as e:
            # Checkpoint unsaved work and exit.
            if not e.skip_exit_checkpoint and not self.checkpoint_is_current():
                yield from self.checkpoint(already_exiting=True)

        finally:
            self.preemption.close()


def make_compatibility_workloads(
    session: client.Session,
    env: det.EnvContext,
    dist: det.DistributedContext,
) -> Tuple[workload.Stream, Optional[WorkloadSequencer]]:
    """
    make_compatibility_workloads will create a stream of workloads to allow a pre-push-architecture
    TrialController train in a push-architecture world, by imitating the exact workloads that would
    have been generated by the pre-push master.
    """

    if dist.get_rank() == 0:
        wlsq = WorkloadSequencer(env, session, dist)  # type: Optional[WorkloadSequencer]
    else:
        wlsq = None

    def workloads() -> workload.Stream:
        if wlsq:
            # Workloads are generated only on the chief worker.
            for wkld, response_fn in wlsq:
                # Distribute to peers.
                _ = dist._zmq_broadcast(wkld)
                # Process workload.
                yield wkld, response_fn
                # Wait for peers.
                _ = dist._zmq_gather(None)
            # Break the workers out of their loop.
            _ = dist._zmq_broadcast(None)
        else:
            while True:
                # Wait for chief to broadcast workload.
                wkld = dist._zmq_broadcast(None)
                if wkld is None:
                    break
                # Process workload.
                yield wkld, lambda _: None
                # Tell chief we finished.
                _ = dist._zmq_gather(None)

    return workloads(), wlsq
