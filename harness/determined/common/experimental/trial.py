import enum
from typing import Any, Optional

from determined.common import api
from determined.common.api import bindings
from determined.common.experimental import checkpoint


class TrialReference:
    """
    A TrialReference object is usually obtained from
    ``determined.experimental.client.get_trial()``.

    Trial reference class used for querying relevant
    :class:`~determined.experimental.Checkpoint` instances.
    """

    def __init__(self, trial_id: int, session: api.Session):
        self.id = trial_id
        self._session = session

    def kill(self) -> None:
        bindings.post_KillTrial(self._session, id=self.id)

    def top_checkpoint(
        self,
        sort_by: Optional[str] = None,
        smaller_is_better: Optional[bool] = None,
    ) -> checkpoint.Checkpoint:
        """
        Return the :class:`~determined.experimental.Checkpoint` instance with the best
        validation metric as defined by the ``sort_by`` and ``smaller_is_better``
        arguments.

        Arguments:
            sort_by (string, optional): The name of the validation metric to
                order checkpoints by. If this parameter is unset the metric defined
                in the related experiment configuration searcher field will be
                used.

            smaller_is_better (bool, optional): Whether to sort the
                metric above in ascending or descending order. If ``sort_by`` is unset,
                this parameter is ignored. By default, the value of ``smaller_is_better``
                from the experiment's configuration is used.
        """
        return self.select_checkpoint(
            best=True, sort_by=sort_by, smaller_is_better=smaller_is_better
        )

    def select_checkpoint(
        self,
        latest: bool = False,
        best: bool = False,
        uuid: Optional[str] = None,
        sort_by: Optional[str] = None,
        smaller_is_better: Optional[bool] = None,
    ) -> checkpoint.Checkpoint:
        """
        Return the :class:`~determined.experimental.Checkpoint` instance with the best
        validation metric as defined by the ``sort_by`` and ``smaller_is_better``
        arguments.

        Exactly one of the ``best``, ``latest``, or ``uuid`` parameters must be set.

        Arguments:
            latest (bool, optional): Return the most recent checkpoint.

            best (bool, optional): Return the checkpoint with the best validation
                metric as defined by the ``sort_by`` and ``smaller_is_better``
                arguments. If ``sort_by`` and ``smaller_is_better`` are not
                specified, the values from the associated experiment
                configuration will be used.

            uuid (string, optional): Return the checkpoint for the specified UUID.

            sort_by (string, optional): The name of the validation metric to
                order checkpoints by. If this parameter is unset the metric defined
                in the related experiment configuration searcher field will be
                used.

            smaller_is_better (bool, optional): Whether to sort the
                metric above in ascending or descending order. If ``sort_by`` is unset,
                this parameter is ignored. By default, the value of ``smaller_is_better``
                from the experiment's configuration is used.
        """
        if sum([int(latest), int(best), int(uuid is not None)]) != 1:
            raise AssertionError("Exactly one of latest, best, or uuid must be set")

        if (sort_by is None) != (smaller_is_better is None):
            raise AssertionError("sort_by and smaller_is_better must be set together")

        if sort_by is not None and not best:
            raise AssertionError(
                "`sort_by` and `smaller_is_better` parameters can only be used with `best`"
            )

        if uuid:
            resp = bindings.get_GetCheckpoint(self._session, checkpointUuid=uuid)
            return checkpoint.Checkpoint._from_bindings(resp.checkpoint, self._session)

        def get_one(offset: int) -> bindings.v1GetTrialCheckpointsResponse:
            return bindings.get_GetTrialCheckpoints(
                self._session,
                id=self.id,
                orderBy=bindings.v1OrderBy.ORDER_BY_DESC,
                sortBy=bindings.v1GetTrialCheckpointsRequestSortBy.SORT_BY_BATCH_NUMBER,
                offset=offset,
            )

        resps = api.read_paginated(get_one)

        checkpoints = [
            checkpoint.Checkpoint._from_bindings(c, self._session)
            for r in resps
            for c in r.checkpoints
        ]

        if not checkpoints:
            raise AssertionError("No checkpoint found for trial {}".format(self.id))

        if latest:
            return checkpoints[0]

        if not sort_by:
            training = checkpoints[0].training
            assert training
            config = training.experiment_config
            sb = config.get("searcher", {}).get("metric")
            if not isinstance(sb, str):
                raise ValueError(
                    "no searcher.metric found in experiment config; please provide a sort_by metric"
                )
            sort_by = sb
            smaller_is_better = config.get("searcher", {}).get("smaller_is_better", True)

        def has_metric(c: checkpoint.Checkpoint) -> bool:
            if c.training is None:
                return False
            return sort_by in c.training.validation_metrics.get("avgMetrics", {})

        checkpoints_with_metric = [c for c in checkpoints if has_metric(c)]

        if not checkpoints_with_metric:
            raise AssertionError(f"No checkpoint for trial {self.id} has metric {sort_by}")

        best_checkpoint_func = min if smaller_is_better else max

        def key(ckpt: checkpoint.Checkpoint) -> Any:
            training = ckpt.training
            assert training
            return training.validation_metrics["avgMetrics"][sort_by]

        return best_checkpoint_func(checkpoints_with_metric, key=key)

    def __repr__(self) -> str:
        return "Trial(id={})".format(self.id)


# This is to shorten line lengths of the TrialSortBy definition.
_tsb = bindings.v1GetExperimentTrialsRequestSortBy


class TrialSortBy(enum.Enum):
    """
    Specifies the field to sort a list of trials on.
    """

    UNSPECIFIED = _tsb.SORT_BY_UNSPECIFIED.value
    ID = _tsb.SORT_BY_ID.value
    START_TIME = _tsb.SORT_BY_START_TIME.value
    END_TIME = _tsb.SORT_BY_END_TIME.value
    STATE = _tsb.SORT_BY_STATE.value
    BEST_VALIDATION_METRIC = _tsb.SORT_BY_BEST_VALIDATION_METRIC.value
    LATEST_VALIDATION_METRIC = _tsb.SORT_BY_LATEST_VALIDATION_METRIC.value
    BATCHES_PROCESSED = _tsb.SORT_BY_BATCHES_PROCESSED.value
    DURATION = _tsb.SORT_BY_DURATION.value
    RESTARTS = _tsb.SORT_BY_RESTARTS.value

    def _to_bindings(self) -> bindings.v1GetExperimentTrialsRequestSortBy:
        return _tsb(self.value)


class TrialOrderBy(enum.Enum):
    """
    Specifies whether a sorted list of trials should be in ascending or
    descending order.
    """

    ASCENDING = bindings.v1OrderBy.ORDER_BY_ASC.value
    ASC = bindings.v1OrderBy.ORDER_BY_ASC.value
    DESCENDING = bindings.v1OrderBy.ORDER_BY_DESC.value
    DESC = bindings.v1OrderBy.ORDER_BY_DESC.value

    def _to_bindings(self) -> bindings.v1OrderBy:
        return bindings.v1OrderBy(self.value)
