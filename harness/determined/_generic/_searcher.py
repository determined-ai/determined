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


class AdvancedSearcher:
    """A namespaced API to make it clear which searcher-related things go to which API"""

    def __init__(self, session: Session, trial_id: int, run_id: int, allocation_id: str) -> None:
        self._session = session
        self._trial_id = trial_id
        self._run_id = run_id
        self._allocation_id = allocation_id

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
