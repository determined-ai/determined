import enum
import logging
import math
from typing import Iterator, Optional

import determined as det
from determined.common.experimental.session import Session

logger = logging.getLogger("determined.generic")


class Unit(enum.Enum):
    EPOCHS = "UNIT_EPOCHS"
    RECORDS = "UNIT_RECORDS"
    BATCHES = "UNIT_BATCHES"


class SearcherOp:
    def __init__(
        self,
        session: Session,
        trial_id: int,
        unit: Unit,
        length: int,
    ) -> None:
        self._session = session
        self._trial_id = trial_id
        self._unit = unit
        self._length = length
        self._completed = False

    @property
    def unit(self) -> Unit:
        return self._unit

    @property
    def length(self) -> int:
        return self._length

    @property
    def records(self) -> int:
        if self._unit != Unit.RECORDS:
            raise RuntimeError(
                "you can only read op.records if you configured your searcher in terms of records"
            )
        return self._length

    @property
    def batches(self) -> int:
        if self._unit != Unit.BATCHES:
            raise RuntimeError(
                "you can only read op.batches if you configured your searcher in terms of batches"
            )
        return self._length

    @property
    def epochs(self) -> int:
        if self._unit != Unit.EPOCHS:
            raise RuntimeError(
                "you can only read op.epochs if you configured your searcher in terms of epochs"
            )
        return self._length

    def report_progress(self, length: float) -> None:
        if self._completed and length != self._length:
            raise RuntimeError("you must not call op.report_progress() after op.complete()")
        logger.debug(f"op.report_progress({length})")
        self._session.post(
            f"/api/v1/trials/{self._trial_id}/progress",
            data=det.util.json_encode(length),
        )

    def complete(self, searcher_metric: float) -> None:
        if self._completed:
            raise RuntimeError("you may only call op.complete() once")
        if math.isnan(searcher_metric):
            raise RuntimeError("searcher_metric may not be NaN")
        self._completed = True
        body = {
            "op": {
                "length": {
                    "length": self._length,
                    "unit": self._unit.value,
                }
            },
            "searcherMetric": searcher_metric,
        }
        logger.debug(f"op.complete({searcher_metric})")
        self._session.post(
            f"/api/v1/trials/{self._trial_id}/searcher/completed_operation",
            data=det.util.json_encode(body),
        )


class SearcherEpoch:
    def __init__(self, epoch_idx: int, op: SearcherOp) -> None:
        self.epoch_idx = epoch_idx
        self._op = op
        self._completed = False

    def complete(self, searcher_metric: float) -> None:
        if self._completed:
            raise RuntimeError("you may only call op.complete() once")
        self._completed = True
        epochs_completed = self.epoch_idx + 1
        if epochs_completed < self._op.epochs:
            self._op.report_progress(epochs_completed)
        else:
            self._op.complete(searcher_metric)


class Searcher:
    """
    There are two ways to use the Searcher API: the basic epoch-based approached, and the more
    advanced operation-based approach.

    The basic epoch-based approach requires that you configure your searcher in terms of epochs.
    You iterate through ``SearcherEpoch`` objects one at a time, reporting your searcher metric
    after each epoch is complete:

    .. code:: python

       # You'll have to load initial_epoch from a checkpoint if you
       # want to support pausing/continuing training:
       epochs_trained = 0

       # Iterate through epoch objects during training:
       for epoch in generic_context.searcher.epochs(epochs_trained):
           val_metrics = my_train_and_validate(epoch_idx = epoch.epoch_idx)

           # pass your searcher metric to epoch.complete() after each epoch:
           epoch.complete(val_metrics["my_searcher_metric"])

    The more advanced operation-based approach supports searchers configured in epochs, batches, or
    records.  It requires you to introduce a second layer of for-loop into the structure of your
    training loop, but gives you more control over the frequency of training interruptions.  You can
    report training progress as frequently as you like, or if validation of your model is expensive,
    you can validate as infrequently as the searcher algorithm will allow, for optimal performance.

    In the operation-based approach, you iterate through ``SearcherOp`` objects.  When you report
    progress, it will be reflected in the WebUI, and when you have finished the training necessary
    for the SearcherOp, you pass it to ``op.complete``:

    .. code:: python

       # You'll have to load your starting point from a checkpoint
       # if you want to support pausing/continuing training:
       length_trained = 0

       for op in generic_context.searcher.ops():

           # Train differently based on op.unit.  Note that op.unit
           # reflects how you configured your searcher, so you are
           # free to only support units you care about using.
           if op.unit == det.generic.Unit.EPOCHS:
               # Train for however long the op requires you to.
               # Note that op.length is an absolute length, not an
               # incremental length; i.e. you need to subtract any
               # length of training you've already completed:
               for i in range(length_trained, op.length):
                   my_train_epoch()
                   # Report progress in the same units as the op:
                   op.report_progress(i+1)

           elif op.unit == det.generic.Unit.BATCHES:
               # Same thing but in batches:
               for i in range(length_trained, op.length):
                   my_train_batch()
                   # Reporting progress every batch would be expensive:
                   if i % 1000:
                       op.report_progress(i+1)

           elif op.unit == det.generic.Unit.RECORDS:
               # Same thing but in records.
               ...

           else:
               # Future-proofing is always encouraged.
               raise ValueError("unrecognized op type:", type(op))

           length_trained = op.length

           # After training the required amount, pass your searcher
           # metric to op.complete():

           # Pass your searcher metric to op.complete():
           val_metrics = my_validate()
           op.complete(va_metrics["my_searcher_metric"])
    """

    def __init__(self, session: Session, trial_id: int, run_id: int, allocation_id: str) -> None:
        self._session = session
        self._trial_id = trial_id
        self._run_id = run_id
        self._allocation_id = allocation_id

        self._iterated = None  # type: Optional[str]

    def _get_searcher_op(self) -> Optional[SearcherOp]:
        logger.debug("_get_searcher_op()")
        r = self._session.get(f"/api/v1/trials/{self._trial_id}/searcher/operation")
        body = r.json()
        if body["completed"]:
            return None

        length = body["op"]["validateAfter"]["length"]
        return SearcherOp(
            self._session, self._trial_id, unit=Unit(length["unit"]), length=length["length"]
        )

    def epochs(self, initial_epoch: int = 0, auto_ack: bool = True) -> Iterator[SearcherEpoch]:
        """
        Iterate through all the epochs requested by this searcher.

        The caller must call epoch.complete() after completing each epoch of training, with the
        validation metric computed after the training.

        Searcher.epochs() is simple, but has the following limitations:
          - It will raise an exception if the searcher is configured in units other than epochs.
          - It requires you to validate every epoch, in order to call epoch.complete().
          - It does not support reporting progress on except on epoch boundaries.
        """
        if self._iterated is not None:
            raise RuntimeError(
                f"illegal second iteration through the searcher: searcher.{self._iterated}() has "
                "already been called previously"
            )

        self._iterated = "epochs"

        epochs_completed = initial_epoch
        while True:
            op = self._get_searcher_op()
            if op is None:
                if auto_ack:
                    self.acknowledge_out_of_ops()
                break
            if op.unit != Unit.EPOCHS:
                raise RuntimeError(
                    f"illegal call to searcher.epochs() with a searcher configured in {op.unit}"
                )
            for epoch_idx in range(epochs_completed, op.epochs):
                epoch = SearcherEpoch(epoch_idx, op)
                yield epoch
                if not epoch._completed:
                    raise RuntimeError("you must call epoch.complete() on each epoch")
            epochs_completed = op.epochs

    def ops(self, auto_ack: bool = True) -> Iterator[SearcherOp]:
        """
        Iterate through all the ops this searcher has to offer.

        The caller must call op.complete() after completing the training requested by each
        operation, with the validation metric calculated after the training.

        Searcher.ops() supports the full capabilities of hyperparameter search in the Determined
        platform, but is more complex than using Searcher.epochs().
        """
        if self._iterated is not None:
            raise RuntimeError(
                f"illegal second iteration through the searcher: searcher.{self._iterated}() has "
                "already been called previously"
            )

        self._iterated = "ops"

        while True:
            op = self._get_searcher_op()
            if op is None:
                if auto_ack:
                    self.acknowledge_out_of_ops()
                break
            yield op
            if not op._completed:
                raise RuntimeError("you must call op.complete() on each operation")

    def acknowledge_out_of_ops(self) -> None:
        """
        acknowledge_out_of_ops() tells the Determined master that you are shutting down because
        you have recognized the searcher has no more operations for you to complete at this time.

        This is important for the Determined master to know that it is safe to restart this process
        should new operations be assigned to this trial.

        acknowledge_out_of_ops() is normally called automatically just before searcher.ops() or
        searcher.epochs() raises a StopIteration exception, unless ops()/epochs() is called with
        auto_ack=False.
        """
        logger.debug(f"acknowledge_out_of_ops(allocation_id:{self._allocation_id})")
        self._session.post(f"/api/v1/allocations/{self._allocation_id}/signals/ack_preemption")


class DummySearcherOp(SearcherOp):
    def __init__(self, unit: Unit, length: int) -> None:
        self._unit = unit
        self._length = length
        self._completed = False

    def report_progress(self, length: float) -> None:
        if self._completed and length != self._length:
            raise RuntimeError("you must not call op.report_progress() after op.complete()")
        logger.info("progress report: {length}/{self._length}")

    def complete(self, searcher_metric: float) -> None:
        if self._completed:
            raise RuntimeError("you may only call op.complete() once")
        if math.isnan(searcher_metric):
            raise RuntimeError("searcher_metric may not be NaN")
        self._completed = True
        logger.info(f"SearcherOp Complete: searcher_metric={det.util.json_encode(searcher_metric)}")


class DummySearcher(Searcher):
    """Yield a singe search op.  We need a way for this to be configurable."""

    def __init__(self, unit: Unit = Unit.EPOCHS, length: int = 1) -> None:
        self._unit = unit
        self._length = length

    def ops(self, auto_ack: bool = True) -> Iterator[SearcherOp]:
        op = DummySearcherOp(self._unit, self._length)
        yield op
        if not op._completed:
            raise RuntimeError("you must call op.complete() on each operation")

    def epochs(self, initial_epoch: int = 0, auto_ack: bool = True) -> Iterator[SearcherEpoch]:
        target = initial_epoch + self._length
        op = DummySearcherOp(Unit.EPOCHS, target)
        for epoch_idx in range(initial_epoch, op.epochs):
            epoch = SearcherEpoch(epoch_idx, op)
            yield epoch
            if not epoch._completed:
                raise RuntimeError("you must call epoch.complete() on each epoch")

    def acknowledge_out_of_ops(self) -> None:
        pass
