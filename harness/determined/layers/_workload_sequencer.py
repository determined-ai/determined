import sys
from typing import Any, Generator, Optional, Tuple

import determined as det
from determined import workload
from determined.common import check

# XXX: clean up these paths
from determined import _searcher
from determined import _training
from determined import _checkpointing
from determined import _preemption

WorkloadStreamElem = Tuple[workload.Workload, workload.ResponseFunc]

WorkloadGenerator = Generator[WorkloadStreamElem, None, bool]


def yield_and_await_response(
    wkld: workload.Workload,
) -> Generator[WorkloadStreamElem, None, workload.Metrics]:
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
    out: Optional[workload.Metrics] = None

    def respond(r: workload.Response) -> None:
        assert not isinstance(r, workload.Skipped)
        nonlocal out
        out = r

    yield wkld, respond

    assert out is not None

    return out


class ShouldExit(Exception):
    """
    ShouldExit breaks out of the top-level workload sequencer loop from inside function calls.
    """

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
            last_ckpt=0,
            batches_completed=0,
            step_id=0,
            last_val=0,
        ):
            self.last_ckpt = last_ckpt
            self.batches_completed = batches_completed
            self.step_id = step_id
            self.last_val = last_val

    def __init__(
        self,
        env: det.EnvContext,
        session,
        dist,
    ) -> None:
        self.env = env
        self.session = session
        self._dist = dist
        # XXX use a real run_id
        run_id = 0
        self.training = _training.Training(session, env.det_trial_id, run_id)
        self.checkpointing = _checkpointing.Checkpointing(session, env.det_trial_id)

        self.val_from_previous_run = self.training.get_last_validation()

        self.want_initial_val = self.env.experiment_config.get("perform_initial_validation", False)

        self.ckpt_policy = self.env.experiment_config.get("checkpoint_policy", "best")

        self.state = self.SavableState()

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
        print(f"min_val_period_batches: {self.min_val_period_batches}")
        print(f"min_ckpt_period_batches: {self.min_ckpt_period_batches}")

    def get_state(self) -> Any:
        return vars(self.state)

    def load_state(self, state: Any) -> None:
        self.state = self.SavableState(**state)
        # Detect the case where the final validation we made was against this exact checkpoint.
        # (If the validation was before the checkpoint, the checkpoint has the right state.  If the
        # validation was after the checkpoint, it isn't valid anymore).
        if self.state.batches_completed == self.val_from_previous_run:
            self.state.last_val = self.state.batches_completed

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
            check.is_instance(self.global_batch_size, 0, "global_batch_size must be positive")
            return max(records // self.global_batch_size, 1)
        if epochs is not None:
            check.is_instance(self.records_per_epoch, int, "length must be an integer")
            check.gt(self.global_batch_size, 0, "global_batch_size must be positive")
            return max((epochs * self.records_per_epoch) // self.global_batch_size, 1)

    def check_for_preemption(self):
        assert self.preemption is not None
        if self.preemption.should_preempt():
            raise ShouldExit()

    def train(self, num_batches: int) -> WorkloadGenerator:
        # report a train step is starting
        self.training.set_status("training")

        wkld = workload.Workload(
            kind=workload.Workload.Kind.RUN_STEP,
            e_id=self.env.det_experiment_id,
            t_id=self.env.det_trial_id,
            s_id=self.state.step_id + 1,
            num_batches=num_batches,
            total_batches_processed=self.state.batches_completed,
        )

        response = yield from yield_and_await_response(wkld)

        # train step is complete, process the result

        metrics = response.get("metrics", {}).get("avg_metrics", {})
        self.state.batches_completed += num_batches
        self.state.step_id += 1
        self.training.report_training_metrics(
            batches_trained=self.state.batches_completed,
            metrics=metrics,
        )

        exited_reason = response.get("exited_reason")
        should_exit = exited_reason is not None

        if exited_reason == "INVALID_HP":
            self.training.early_exit(_training.EarlyExitReason.INVALID_HP)

        if should_exit:
            raise ShouldExit()

        self.check_for_preemption()

    def validate(self, op) -> WorkloadGenerator:
        # report a validation step is starting
        self.training.set_status("validating")

        wkld = workload.Workload(
            kind=workload.Workload.Kind.COMPUTE_VALIDATION_METRICS,
            e_id=self.env.det_experiment_id,
            t_id=self.env.det_trial_id,
            s_id=self.state.step_id,
            num_batches=0,
            total_batches_processed=self.state.batches_completed,
        )

        response = yield from yield_and_await_response(wkld)

        # validation step is complete, process the result

        exited_reason = response.get("exited_reason")
        if exited_reason == "INVALID_HP":
            self.training.early_exit(_training.EarlyExitReason.INVALID_HP)
            raise ShouldExit()

        # report to the searcher API first, so we don't end up in a situation where we die between
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
        searcher_metric_name = self.env.experiment_config["searcher"]["metric"]
        searcher_metric = response["metrics"]["validation_metrics"][searcher_metric_name]
        if op is not None and self.batches_until_op_complete(op) < 1:
            op.complete(searcher_metric)

        self.state.last_val = self.state.batches_completed
        self.training.report_validation_metrics(
            batches_trained=self.state.batches_completed,
            metrics=response["metrics"],
        )

        if exited_reason is not None:
            raise ShouldExit()

        if not self.checkpoint_is_current():
            if self.ckpt_policy == "all" or (
                self.ckpt_policy == "best"
                and self.training.is_best_validation_of_experiment(searcher_metric)
            ):
                yield from self.checkpoint(already_exiting=False)

        self.check_for_preemption()

    def checkpoint(
        self,
        already_exiting: bool,
    ) -> Tuple[workload.Workload, workload.ResponseFunc]:
        self.training.set_status("checkpointing")

        # update the last_ckpt now so it can be captured by get_state() after we yield
        self.state.last_ckpt = self.state.batches_completed

        wkld = workload.Workload(
            kind=workload.Workload.Kind.CHECKPOINT_MODEL,
            e_id=self.env.det_experiment_id,
            t_id=self.env.det_trial_id,
            s_id=self.state.step_id,
            num_batches=0,
            total_batches_processed=self.state.batches_completed,
        )
        response = yield from yield_and_await_response(wkld)

        self.checkpointing._report_checkpoint(response["metrics"].storage_id)

        if already_exiting:
            return

        exited_reason = response.get("exited_reason")
        if exited_reason == "INVALID_HP":
            self.training.early_exit(_training.EarlyExitReason.INVALID_HP)

        if exited_reason is not None:
            raise ShouldExit()

        self.check_for_preemption()

    def terminate(self) -> Tuple[workload.Workload, workload.ResponseFunc]:
        wkld = workload.Workload(
            kind=workload.Workload.Kind.TERMINATE,
            e_id=self.env.det_experiment_id,
            t_id=self.env.det_trial_id,
            s_id=self.state.step_id,
            num_batches=0,
            total_batches_processed=self.state.batches_completed,
        )
        yield wkld, lambda _: None

    def batches_until_val(self) -> int:
        return self.state.last_val + self.min_val_period_batches - self.state.batches_completed

    def batches_until_ckpt(self) -> int:
        return self.state.last_ckpt + self.min_ckpt_period_batches - self.state.batches_completed

    def batches_until_op_complete(self, op) -> int:
        return (
            self.as_batches(
                batches=op.length if op.unit == _searcher.Unit.BATCHES else None,
                records=op.length if op.unit == _searcher.Unit.RECORDS else None,
                epochs=op.length if op.unit == _searcher.Unit.EPOCHS else None,
            )
            - self.state.batches_completed
        )

    def checkpoint_is_current(self):
        return self.state.last_ckpt == self.state.batches_completed

    def validation_is_current(self):
        return self.state.last_val == self.state.batches_completed

    def __iter__(self) -> workload.Stream:
        # XXX: wait, the preemption API only works if all workers use it.  Otherwise we need to
        #      use the _PreemptionWatcher directly.  This seems like a complex API.
        self.preemption = _preemption._PreemptionWatcher(self.session, self.env.det_trial_id)
        self.preemption.start()
        try:
            searcher = _searcher.AdvancedSearcher(self.session, self.env.det_trial_id)

            # Step-zero Validations.
            if (
                self.want_initial_val
                and self.val_from_previous_run is None
                and self.state.batches_completed == 0
            ):
                yield from self.validate(None)

            print("entering loop")
            for op in searcher.ops():
                print(f"got op: {op.length} {op.unit}")
                print(f"self.state.batches_completed: {self.state.batches_completed}")
                print(f"self.batches_until_op_complete(op): {self.batches_until_op_complete(op)}")
                print(f"self.batches_until_ckpt(): {self.batches_until_ckpt()}")
                print(f"self.batches_until_val(): {self.batches_until_val()}")

                while self.batches_until_op_complete(op) > 0:
                    # pause training to checkpoint?
                    if self.batches_until_ckpt() < 1:
                        yield from self.checkpoint(already_exiting=False)

                    # pause training to validate?
                    if self.batches_until_val() < 1:
                        print("loop-validating")
                        yield from self.validate(op)

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
                    )

                # Done training for this searcher operation!

                if not self.checkpoint_is_current():
                    yield from self.checkpoint(already_exiting=False)

                if not self.validation_is_current():
                    yield from self.validate(op)

                assert op._completed, "logic error; op was never completed"

        except ShouldExit:
            # XXX: make sure we report all of the early_exit() reasons we care about.
            pass

        finally:
            self.preemption.close()

            # Checkpoint unsaved work.
            if not self.checkpoint_is_current():
                yield from self.checkpoint(already_exiting=True)

            # Always yield a terminate message last.
            yield from self.terminate()


def make_compatibility_workloads(session, env, dist) -> workload.Stream:
    """
    make_compatibility_workloads will create a stream of workloads to allow a pre-push-architecture
    TrialController train in a push-architecture world, by imitating the exact workloads that would
    have been generated by the pre-push master.
    """

    if dist.get_rank() == 0:
        # Workloads are generated only on the chief worker.
        for wkld, response_fn in WorkloadSequencer(env, session, dist):
            # Distribute to peers.
            _ = dist._zmq_broadcast(wkld)
            # XXX: raising an exception here does not cause the trial to exit!
            # Process workload.
            try:
                yield wkld, response_fn
            finally:
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
            try:
                # Process workload.
                yield wkld, lambda _: None
            finally:
                # Tell chief we finished.
                _ = dist._zmq_gather(None)
