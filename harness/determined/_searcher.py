import enum
import json
import selectors
import socket
import threading
from typing import Any, Optional

from determined.common import api
from determined.common.api import certs


# XXX: change the text?  Use 'unit' instead of 'units'?
class Unit(enum.Enum):
    EPOCHS = "UNITS_EPOCHS"
    RECORDS = "UNITS_RECORDS"
    BATCHES = "UNITS_BATCHES"


class SearcherOp:
    def __init__(self, session, trial_id, unit: Unit, length: int) -> None:
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
            raise AssertionError(
                "you can only read op.records if you configured your searcher in terms of records"
            )
        return self._length

    @property
    def batches(self) -> int:
        if self._unit != Unit.BATCHES:
            raise AssertionError(
                "you can only read op.batches if you configured your searcher in terms of batches"
            )
        return self._length

    @property
    def epochs(self) -> int:
        if self._unit != Unit.EPOCHS:
            raise AssertionError(
                "you can only read op.epochs if you configured your searcher in terms of epochs"
            )
        return self._length

    def report_progress(self, length: float) -> None:
        if self._completed and length != self._length:
            raise AssertionError("you must not call op.report_progress() after op.complete()")
        self._session.post(f"/api/v1/trials/{self._trial_id}/progress", body=length)

    def complete(self, searcher_metric: float) -> None:

        if self._completed:
            raise AssertionError("you may only call op.complete() once")
        self._completed = True
        body = {
            "op": {
                "length": {
                    "length": self._length,
                    "units": self._unit.value,
                }
            },
            "searcherMetric": searcher_metric,
        }
        self._session.post(f"/api/v1/trials/{self._trial_id}/searcher/completed_operation", body=body)


class AdvancedSearcher:
    """A namespaced API to make it clear which searcher-related things go to which API"""

    def __init__(self, session, trial_id):
        self._session = session
        self._trial_id = trial_id

    def _get_searcher_op(self):
        r = self._session.get(f"/api/v1/trials/{self._trial_id}/searcher/operation")
        # XXX: handle non-validateAfter workloads
        body = r.json()
        if body["complete"]:
            return None

        length = body["op"]["validateAfter"]["length"]
        return SearcherOp(self._session, self._trial_id, unit=Unit(length["units"]), length=length["length"])

    def ops(self):
        """
        Iterate through all the ops this searcher has to offer.

        The caller must call op.complete() on each operation.
        """

        last_op_length = 0
        while True:
            op = self._get_searcher_op()
            if op is None:
                break
            # XXX: remove this when we have non-validateAfter workloads
            if op.length == last_op_length:
                break
            last_op_length = op.length
            yield op
            if not op._completed:
                raise AssertionError("you must call op.complete() on each operation")
