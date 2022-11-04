import enum
import logging
from typing import Any, Iterator, Optional

import determined as det
from determined import core
from determined.common import api

logger = logging.getLogger("determined.core")


class Unit(enum.Enum):
    EPOCHS = "EPOCHS"
    RECORDS = "RECORDS"
    BATCHES = "BATCHES"


def _parse_searcher_units(experiment_config: dict) -> Optional[Unit]:
    searcher = experiment_config.get("searcher", {})

    def convert_key(key: Any) -> Optional[Unit]:
        return {"records": Unit.RECORDS, "epochs": Unit.EPOCHS, "batches": Unit.BATCHES}.get(key)

    if "unit" in searcher:
        return convert_key(searcher["unit"])

    length_example = searcher.get("max_length")
    if isinstance(length_example, dict) and len(length_example) == 1:
        key = next(iter(length_example.keys()))
        return convert_key(key)
    # Either a `max_length: 50` situation or a broken config.
    return None


class SearcherOperation:
    """
    A ``SearcherOperation`` is a request from the hyperparameter-search logic for the training
    script to execute one train-validate-report cycle.

    Some searchers, such as single, random, or grid, pass only a single ``SearcherOperation`` to
    each trial, while others may pass many ``SearcherOperations``.

    Each ``SearcherOperation`` has a length attribute representing the cumulative training that
    should be completed before the validate-report steps of the cycle.  The length attribute is
    absolute, not incremental, meaning that if the searcher wants you to train for 10 units and
    validate, then train for 10 more units and validate, it emits one ``SearcherOperation`` with
    ``.length=10`` followed by a second ``SearcherOperation`` with ``.length=20``.  Using absolute
    lengths instead of incremental lengths makes restarting after crashes simple and robust.
    """

    def __init__(
        self,
        session: api.Session,
        trial_id: int,
        length: int,
        is_chief: bool,
    ) -> None:
        self._session = session
        self._trial_id = trial_id
        self._length = length
        self._is_chief = is_chief
        self._completed = False

    @property
    def length(self) -> int:
        """
        ``length`` represents the total amount of training which should be reached by the train step
        before the validate-report steps.
        """
        return self._length

    def report_progress(self, length: float) -> None:
        """
        ``report_progress()`` reports the training progress to the Determined master so the WebUI
        can show accurate progress to users.

        The unit of the length value passed to ``report_progress()`` must match the unit of the
        ``.length`` attribute.  The unit of the ``.length`` attribute is user-defined.  When
        treating ``.length`` as batches, ``report_progress()`` should report batches.  When treating
        .length as epochs, ``report_progress()`` must also be in epochs.
        """
        if not self._is_chief:
            raise RuntimeError("you must only call op.report_progress() from the chief worker")
        if self._completed and length != self._length:
            raise RuntimeError("you must not call op.report_progress() after op.report_completed()")
        logger.debug(f"op.report_progress({length})")
        self._session.post(
            f"/api/v1/trials/{self._trial_id}/progress",
            data=det.util.json_encode(length),
        )

    def report_completed(self, searcher_metric: float) -> None:
        """
        ``report_completed()`` is the final step of a train-validate-report cycle.

        ``report_completed()`` requires the value of the metric you are searching over.  This value
        is typically the output of the "validate" step of the train-validate-report cycle.
        """
        if not self._is_chief:
            raise RuntimeError("you must only call op.report_completed() from the chief worker")
        if self._completed:
            raise RuntimeError("you may only call op.report_completed() once")
        self._completed = True
        body = {"op": {"length": self._length}, "searcherMetric": searcher_metric}
        logger.debug(f"op.report_completed({searcher_metric})")
        self._session.post(
            f"/api/v1/trials/{self._trial_id}/searcher/completed_operation",
            data=det.util.json_encode(body),
        )


class SearcherMode(enum.Enum):
    """
    ``SearcherMode`` defines the calling behavior of the ``SearcherContext.operations()`` call.

    When mode is ``WorkersAskChief`` (the default), all workers must call
    ``SearcherContext.operations()`` in step with each other.  The chief iterates through
    searcher operations from the master and then propagates the operations to each worker,
    introducing a synchronization point between workers.

    When mode is ``ChiefOnly``, only the chief may call ``SearcherContext.operations()``.  Usually
    this implies you must manually inform the workers of what work to do next.
    """

    WorkersAskChief = "WORKERS_ASK_CHIEF"
    ChiefOnly = "CHEIF_ONLY"


class SearcherContext:
    """
    ``SearcherContext`` gives direct access to operations emitted by the search algorithm in the
    master.  Each ``SearcherOperation`` emitted has a (unitless) length that you should train for,
    then you complete the op by reporting the validation metric you are searching over.

    It is the user's responsibility to execute the required training.  Because the user configured
    the length of the searcher in the experiment configuration, the user should know if the unitless
    length represents epochs, batches, records, etc.

    It is also the user's responsibility to evaluate the model after training and report the correct
    metric; if you intend to search over a metric called val_accuracy, you should report
    val_accuracy.

    Lastly, it is recommended (not required) to report progress periodically, so that the webui can
    accurately reflect current progress.  Progress is another unitless length.

    Example:

    .. code:: python

       # Assuming you configured the searcher in terms of batches,
       # the op.length is also interpeted as a batch count.
       # Note that you'll have to load your starting point from a
       # checkpoint if you want to support pausing/continuing training.
       batches_trained = 0

       for op in generic_context.searcher.operations():
           # Train for however long the op requires you to.
           # Note that op.length is an absolute length, not an
           # incremental length:
           while batches_trained < op.length:
               my_train_batch()

               batches_trained += 1

               # Reporting progress every batch would be expensive:
               if batches_trained % 1000:
                   op.report_progress(batches_trained)

           # After training the required amount, pass your searcher
           # metric to op.report_completed():
           val_metrics = my_validate()
           op.report_completed(val_metrics["my_searcher_metric"])

    Note that reporting metrics is completely independent of the SearcherContext API, using
    ``core_context.train.report_training_metrics()`` or
    ``core_context.train.report_validation_metrics()``.
    """

    def __init__(
        self,
        session: api.Session,
        dist: core.DistributedContext,
        trial_id: int,
        run_id: int,
        allocation_id: str,
        units: Optional[Unit] = None,
    ) -> None:
        self._session = session
        self._dist = dist
        self._trial_id = trial_id
        self._run_id = run_id
        self._allocation_id = allocation_id
        self._units = units

    def _get_searcher_op(self) -> Optional[SearcherOperation]:
        logger.debug("_get_searcher_op()")
        r = self._session.get(f"/api/v1/trials/{self._trial_id}/searcher/operation")
        body = r.json()
        if body["completed"]:
            return None

        # grpc-gateway encodes uint64 as a string, since it is bigger than a JavaScript `number`.
        length = int(body["op"]["validateAfter"]["length"])
        is_chief = self._dist.rank == 0
        return SearcherOperation(self._session, self._trial_id, length=length, is_chief=is_chief)

    def operations(
        self,
        searcher_mode: SearcherMode = SearcherMode.WorkersAskChief,
        auto_ack: bool = True,
    ) -> Iterator[SearcherOperation]:
        """
        Iterate through all the operations this searcher has to offer.

        See :class:`~determined.core.SearcherMode` for details about calling requirements in
        distributed training scenarios.

        After training to the point specified by each ``SearcherOperation``, the chief, and only the
        chief, must call ``op.report_completed(``) on each operation.  This is true regardless of
        the ``searcher_mode`` setting because the Determined master needs a clear, unambiguous
        report of when an operation is completed.
        """
        searcher_mode = SearcherMode(searcher_mode)

        if self._dist.rank == 0:
            # Chief gets operations from master.
            while True:
                op = self._get_searcher_op()
                if searcher_mode == SearcherMode.WorkersAskChief:
                    # Broadcast op.length (or None) to workers.  We broadcast just the length
                    # because SearcherOperation is not serializable, and the is_chief attribute
                    # obviously must be set on a per-worker basis.
                    _ = self._dist.broadcast(op and op.length)
                if op is None:
                    if auto_ack:
                        self.acknowledge_out_of_ops()
                    break
                yield op
                if not op._completed:
                    raise RuntimeError("you must call op.report_completed() on each operation")
        else:
            if searcher_mode != SearcherMode.WorkersAskChief:
                raise RuntimeError(
                    "you cannot call searcher.operations(searcher_mode=ChiefOnly) from a non-chief "
                    "worker."
                )
            # Worker gets operations from chief.
            while True:
                op_length = self._dist.broadcast(None)
                if op_length is None:
                    break
                yield SearcherOperation(
                    self._session, self._trial_id, length=op_length, is_chief=False
                )

    def acknowledge_out_of_ops(self) -> None:
        """
        acknowledge_out_of_ops() tells the Determined master that you are shutting down because
        you have recognized the searcher has no more operations for you to complete at this time.

        This is important for the Determined master to know that it is safe to restart this process
        should new operations be assigned to this trial.

        acknowledge_out_of_ops() is normally called automatically just before operations() raises a
        StopIteration, unless operations() is called with auto_ack=False.
        """
        logger.debug(f"acknowledge_out_of_ops(allocation_id:{self._allocation_id})")
        self._session.post(f"/api/v1/allocations/{self._allocation_id}/signals/ack_preemption")

    def get_configured_units(self) -> Optional[Unit]:
        """
        get_configured_units() reports what units were used in the searcher field of the experiment
        config.  If no units were configured, None is returned.

        An experiment configured like this causes ``get_configured_units()`` to return EPOCHS:

        .. code:: yaml

           searcher:
             name: single
             max_length:
               epochs: 50

        An experiment configured like this causes ``get_configured_units()`` to return None:

        .. code:: yaml

           searcher:
             name: single
             max_length: 50
        """
        return self._units


class DummySearcherOperation(SearcherOperation):
    def __init__(self, length: int, is_chief: bool) -> None:
        self._length = length
        self._is_chief = is_chief
        self._completed = False

    def report_progress(self, length: float) -> None:
        if not self._is_chief:
            raise RuntimeError("you must only call op.report_progress() from the chief worker")
        if self._completed and length != self._length:
            raise RuntimeError("you must not call op.report_progress() after op.report_completed()")
        logger.info("progress report: {length}/{self._length}")

    def report_completed(self, searcher_metric: float) -> None:
        if not self._is_chief:
            raise RuntimeError("you must only call op.report_completed() from the chief worker")
        if self._completed:
            raise RuntimeError("you may only call op.report_completed() once")
        self._completed = True
        logger.info(
            f"SearcherOperation Complete: searcher_metric={det.util.json_encode(searcher_metric)}"
        )


class DummySearcherContext(SearcherContext):
    """Yield a singe search op.  We need a way for this to be configurable."""

    def __init__(self, dist: core.DistributedContext, length: int = 1) -> None:
        self._dist = dist
        self._length = length

    def operations(
        self,
        searcher_mode: SearcherMode = SearcherMode.WorkersAskChief,
        auto_ack: bool = True,
    ) -> Iterator[SearcherOperation]:
        searcher_mode = SearcherMode(searcher_mode)
        # Force the same synchronization behavior in the DummySearcherContext as the real one.
        if self._dist.rank == 0:
            # Chief makes a dummy op.
            op = DummySearcherOperation(self._length, self._dist.rank == 0)
            if searcher_mode == SearcherMode.WorkersAskChief:
                # Broadcast op to workers.
                _ = self._dist.broadcast(op and op.length)
            yield op
            if not op._completed:
                raise RuntimeError("you must call op.report_completed() on each operation")
            if searcher_mode == SearcherMode.WorkersAskChief:
                _ = self._dist.broadcast(None)
        else:
            if searcher_mode != SearcherMode.WorkersAskChief:
                raise RuntimeError(
                    "you cannot call searcher.operations(searcher_mode=ChiefOnly) "
                    "from a non-chief worker."
                )
            # Worker gets operations from chief.
            while True:
                op_length = self._dist.broadcast(None)
                if op_length is None:
                    break
                yield DummySearcherOperation(op_length, False)

    def acknowledge_out_of_ops(self) -> None:
        pass

    def get_configured_units(self) -> Optional[Unit]:
        return Unit.EPOCHS
