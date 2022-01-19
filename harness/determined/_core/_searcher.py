import enum
import logging
import math
from typing import Iterator, Optional

import determined as det
from determined.common.experimental.session import Session

logger = logging.getLogger("determined.core")


class Unit(enum.Enum):
    EPOCHS = "EPOCHS"
    RECORDS = "RECORDS"
    BATCHES = "BATCHES"


def _parse_searcher_units(experiment_config: dict) -> Optional[Unit]:
    searcher = experiment_config.get("searcher", {})
    # All searchers have max_length, except pbt which has a length_per_round.
    length_example = searcher.get("max_length") or searcher.get("length_per_round")
    if isinstance(length_example, dict) and len(length_example) == 1:
        key = next(iter(length_example.keys()))
        return {"records": Unit.RECORDS, "epochs": Unit.EPOCHS, "batches": Unit.BATCHES}.get(key)
    # Either a `max_length: 50` situation or a broken config.
    return None


class SearcherOp:
    def __init__(
        self,
        session: Session,
        trial_id: int,
        length: int,
    ) -> None:
        self._session = session
        self._trial_id = trial_id
        self._length = length
        self._completed = False

    @property
    def length(self) -> int:
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
        body = {"op": {"length": self._length}, "searcherMetric": searcher_metric}
        logger.debug(f"op.complete({searcher_metric})")
        self._session.post(
            f"/api/v1/trials/{self._trial_id}/searcher/completed_operation",
            data=det.util.json_encode(body),
        )


class Searcher:
    """
    Searcher gives direct access to operations emitted by the search algorithm in the master.  Each
    SearcherOp emitted has a (unitless) length that you should train for, then you complete the op
    by reporting the validation metric you are searching over.

    It is the user's responsibility to execute the required training.  Since the user configured the
    length of the searcher in the experiment configuration, the user should know if the unitless
    length represents epochs, batches, records, etc.

    It is also the user's responsibility to evaluate the model after training and report the correct
    metric; if you intend to search over a metric called val_accuracy, you should report
    val_accuracy.

    Lastly, it is recommended (not required) to report progress periodically, so that the webui can
    accurately reflect current progress.  Progress is another unitless length.

    Example:

    .. code:: python

       # We'll pretend we configured we configured the searcher
       # in terms of batches, so we will interpet the the op.length as a
       # count of batches.
       # Note that you'll have to load your starting point from a
       # checkpoint if you want to support pausing/continuing training.
       batches_trained = 0

       for op in generic_context.searcher.ops():
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
           # metric to op.complete():
           val_metrics = my_validate()
           op.complete(val_metrics["my_searcher_metric"])

    Note that reporting metrics is completely independent of the Searcher API, via
    ``core_context.training.report_training_metrics()`` or
    ``core_context.training.report_validation_metrics()``.
    """

    def __init__(
        self,
        session: Session,
        trial_id: int,
        run_id: int,
        allocation_id: str,
        units: Optional[Unit] = None,
    ) -> None:
        self._session = session
        self._trial_id = trial_id
        self._run_id = run_id
        self._allocation_id = allocation_id
        self._units = units

    def _get_searcher_op(self) -> Optional[SearcherOp]:
        logger.debug("_get_searcher_op()")
        r = self._session.get(f"/api/v1/trials/{self._trial_id}/searcher/operation")
        body = r.json()
        if body["completed"]:
            return None

        # grpc-gateway encodes uint64 as a string, since it is bigger than a JavaScript `number`.
        length = int(body["op"]["validateAfter"]["length"])
        return SearcherOp(self._session, self._trial_id, length=length)

    def ops(self, auto_ack: bool = True) -> Iterator[SearcherOp]:
        """
        Iterate through all the ops this searcher has to offer.

        The caller must call op.complete() on each operation.
        """

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

        acknowledge_out_of_ops() is normally called automatically just before ops() raises a
        StopIteration, unless ops() is called with auto_ack=False.
        """
        logger.debug(f"acknowledge_out_of_ops(allocation_id:{self._allocation_id})")
        self._session.post(f"/api/v1/allocations/{self._allocation_id}/signals/ack_preemption")

    def get_configured_units(self) -> Optional[Unit]:
        """
        get_configured_units() reports what units were used in the searcher field of the experiment
        config.  If no units were configured, None is returned.

        An experiment configured like this would cause ``get_configured_units()`` to return EPOCHS:

        .. code:: yaml

           searcher:
             name: single
             max_length:
               epochs: 50

        An experiment configured like this would cause ``get_configured_units()`` to return None:

        .. code:: yaml

           searcher:
             name: single
             max_length: 50
        """
        return self._units


class DummySearcherOp(SearcherOp):
    def __init__(self, length: int) -> None:
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

    def __init__(self, length: int = 1) -> None:
        self._length = length

    def ops(self, auto_ack: bool = True) -> Iterator[SearcherOp]:
        op = DummySearcherOp(self._length)
        yield op
        if not op._completed:
            raise RuntimeError("you must call op.complete() on each operation")

    def acknowledge_out_of_ops(self) -> None:
        pass

    def get_configured_units(self) -> Optional[Unit]:
        return Unit.EPOCHS
