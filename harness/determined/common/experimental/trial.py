import dataclasses
import datetime
import enum
from typing import Any, Dict, Iterable, List, Optional, Union

from determined.common import api, util
from determined.common.api import bindings, logs
from determined.common.experimental import checkpoint


class LogLevel(enum.Enum):
    TRACE = bindings.v1LogLevel.LOG_LEVEL_TRACE.value
    DEBUG = bindings.v1LogLevel.LOG_LEVEL_DEBUG.value
    INFO = bindings.v1LogLevel.LOG_LEVEL_INFO.value
    WARNING = bindings.v1LogLevel.LOG_LEVEL_WARNING.value
    ERROR = bindings.v1LogLevel.LOG_LEVEL_ERROR.value
    CRITICAL = bindings.v1LogLevel.LOG_LEVEL_CRITICAL.value

    def _to_bindings(self) -> bindings.v1LogLevel:
        return bindings.v1LogLevel(self.value)


_csb = bindings.v1GetTrialCheckpointsRequestSortBy


class CheckpointSortBy(enum.Enum):
    """
    Specifies the field to sort a list of checkpoints on.
    """

    END_TIME = _csb.SORT_BY_END_TIME.value
    STATE = _csb.SORT_BY_STATE.value
    UUID = _csb.SORT_BY_UUID.value
    BATCH_NUMBER = _csb.SORT_BY_BATCH_NUMBER.value

    def _to_bindings(self) -> bindings.v1GetTrialCheckpointsRequestSortBy:
        return _csb(self.value)


class CheckpointOrderBy(enum.Enum):
    """
    Specifies whether a sorted list of checkpoints should be in ascending or
    descending order.
    """

    ASC = bindings.v1OrderBy.ORDER_BY_ASC.value
    DESC = bindings.v1OrderBy.ORDER_BY_DESC.value

    def _to_bindings(self) -> bindings.v1OrderBy:
        return bindings.v1OrderBy(self.value)


@dataclasses.dataclass
class TrainingMetrics:
    """
    Specifies a training metric report that the trial reported.

    Attributes:
        trial_id
        trial_run_id
        steps_completed
        end_time
        metrics
        batch_metrics
    """

    trial_id: int
    trial_run_id: int
    steps_completed: int
    end_time: datetime.datetime
    metrics: Dict[str, Any]
    batch_metrics: Optional[List[Dict[str, Any]]] = None

    @classmethod
    def _from_bindings(cls, metric_report: bindings.v1MetricsReport) -> "TrainingMetrics":
        return cls(
            trial_id=metric_report.trialId,
            trial_run_id=metric_report.trialRunId,
            steps_completed=metric_report.totalBatches,
            end_time=util.parse_protobuf_timestamp(metric_report.endTime),
            metrics=metric_report.metrics["avg_metrics"],
            batch_metrics=metric_report.metrics.get("batch_metrics", None),
        )


@dataclasses.dataclass
class ValidationMetrics:
    """
    Specifies a validation metric report that the trial reported.

    Attributes:
        trial_id
        trial_run_id
        steps_completed
        end_time
        metrics
    """

    trial_id: int
    trial_run_id: int
    steps_completed: int
    end_time: datetime.datetime
    metrics: Dict[str, Any]

    @classmethod
    def _from_bindings(cls, metric_report: bindings.v1MetricsReport) -> "ValidationMetrics":
        return cls(
            trial_id=metric_report.trialId,
            trial_run_id=metric_report.trialRunId,
            steps_completed=metric_report.totalBatches,
            end_time=util.parse_protobuf_timestamp(metric_report.endTime),
            metrics=metric_report.metrics["validation_metrics"],
        )


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

    def logs(
        self,
        follow: bool = False,
        *,
        head: Optional[int] = None,
        tail: Optional[int] = None,
        container_ids: Optional[List[str]] = None,
        rank_ids: Optional[List[int]] = None,
        stdtypes: Optional[List[str]] = None,
        min_level: Optional[LogLevel] = None,
    ) -> Iterable[str]:
        """
        Return an iterable of log lines from this trial meeting the specified criteria.

        Arguments:
            follow (bool, optional): If the iterable should block waiting for new logs to arrive.
                Mutually exclusive with ``head`` and ``tail``.  Defaults to ``False``.
            head (int, optional): When set, only fetches the first ``head`` lines.  Mutually
                exclusive with ``follow`` and ``tail``.  Defaults to ``None``.
            tail (int, optional): When set, only fetches the first ``head`` lines.  Mutually
                exclusive with ``follow`` and ``head``.  Defaults to ``None``.
            container_ids (List[str], optional): When set, only fetch logs from lines from
                specific containers.  Defaults to ``None``.
            rank_ids (List[int], optional): When set, only fetch logs from lines from
                specific ranks.  Defaults to ``None``.
            stdtypes (List[int], optional): When set, only fetch logs from lines from the given
                stdio outputs.  Defaults to ``None`` (same as ``["stdout", "stderr"]``).
            min_level: (LogLevel, optional): When set, defines the minimum log priority for lines
                that will be returned.  Defaults to ``None`` (all logs returned).
        """
        if head is not None and head < 0:
            raise ValueError(f"head must be non-negative, got {head}")
        if tail is not None and tail < 0:
            raise ValueError(f"tail must be non-negative, got {tail}")
        for log in logs.trial_logs(
            session=self._session,
            trial_id=self.id,
            head=head,
            tail=tail,
            follow=follow,
            # TODO: Rename this to "node_id" and support it in the python sdk.
            agent_ids=None,
            container_ids=container_ids,
            rank_ids=rank_ids,
            # sources would be something like "originated from master" or "originated from task".
            sources=None,
            stdtypes=stdtypes,
            min_level=None if min_level is None else min_level._to_bindings(),
            # TODO: figure out what type is a good type to accept for timestamps.  Until then, be
            # conservative with the public API and disallow it.
            timestamp_before=None,
            timestamp_after=None,
        ):
            yield log.message

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

        order_by = None
        if latest:
            sort_by = CheckpointSortBy.BATCH_NUMBER  # type: ignore
            order_by = CheckpointOrderBy.DESC

        if sort_by:
            order_by = CheckpointOrderBy.ASC if smaller_is_better else CheckpointOrderBy.DESC

        checkpoints = self.get_checkpoints(sort_by=sort_by, order_by=order_by)

        if not checkpoints:
            raise ValueError("No checkpoints found for criteria.")
        return checkpoints[0]

    def get_checkpoints(
        self,
        sort_by: Optional[Union[str, CheckpointSortBy]] = None,
        order_by: Optional[CheckpointOrderBy] = None,
    ) -> List[checkpoint.Checkpoint]:
        """
        Return a list of :class:`~determined.experimental.Checkpoint` instances for the current
        trial.

        Either ``sort_by`` and ``order_by`` are both specified or neither are.

        Arguments:
            sort_by (string, :class:`~determined.experimental.CheckpointSortBy`): Which field to
                sort by. Strings are assumed to be validation metric names.
            order_by (:class:`~determined.experimental.CheckpointOrderBy`): Whether to sort in
                ascending or descending order.
        """

        if (sort_by is None) != (order_by is None):
            raise AssertionError("sort_by and order_by must be set together")

        def get_trial_checkpoints(offset: int) -> bindings.v1GetTrialCheckpointsResponse:
            return bindings.get_GetTrialCheckpoints(
                self._session,
                id=self.id,
                orderBy=order_by._to_bindings() if order_by else None,
                sortBy=sort_by._to_bindings() if isinstance(sort_by, CheckpointSortBy) else None,
                offset=offset,
            )

        resps = api.read_paginated(get_trial_checkpoints)

        checkpoints = [
            checkpoint.Checkpoint._from_bindings(c, self._session)
            for r in resps
            for c in r.checkpoints
        ]

        # If sort_by was a defined field, we already sorted and ordered.
        if isinstance(sort_by, CheckpointSortBy) or not checkpoints:
            return checkpoints

        # If sort not specified, sort and order default to searcher configs.
        if not sort_by:
            training = checkpoints[0].training
            assert training
            config = training.experiment_config
            searcher_metric = config.get("searcher", {}).get("metric")
            if not isinstance(searcher_metric, str):
                raise ValueError(
                    "no searcher.metric found in experiment config; please provide a sort_by metric"
                )
            sort_by = searcher_metric
            smaller_is_better = config.get("searcher", {}).get("smaller_is_better", True)
            order_by = CheckpointOrderBy.ASC if smaller_is_better else CheckpointOrderBy.DESC

        assert sort_by is not None and order_by is not None, "sort_by and order_by not defined."

        reverse = order_by == CheckpointOrderBy.DESC

        def key(ckpt: checkpoint.Checkpoint) -> Any:
            training = ckpt.training
            assert training
            metric = training.validation_metrics.get("avgMetrics") or {}
            metric = metric.get(sort_by)

            # Return a bool here to sort checkpoints that may have no validation metrics.
            if reverse:
                return metric is not None, metric
            else:
                return metric is None, metric

        checkpoints.sort(reverse=reverse, key=key)

        return checkpoints

    def __repr__(self) -> str:
        return "Trial(id={})".format(self.id)

    def stream_training_metrics(self) -> Iterable[TrainingMetrics]:
        """
        Streams training metrics for this trial sorted by
        trial_id, trial_run_id and steps_completed.
        """
        return _stream_training_metrics(self._session, [self.id])

    def stream_validation_metrics(self) -> Iterable[ValidationMetrics]:
        """
        Streams validation metrics for this trial sorted by
        trial_id, trial_run_id and steps_completed.
        """
        return _stream_validation_metrics(self._session, [self.id])


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
    CHECKPOINT_SIZE = _tsb.SORT_BY_CHECKPOINT_SIZE.value

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


def _stream_training_metrics(
    session: api.Session, trial_ids: List[int]
) -> Iterable[TrainingMetrics]:
    for i in bindings.get_GetTrainingMetrics(session, trialIds=trial_ids):
        for m in i.metrics:
            yield TrainingMetrics._from_bindings(m)


def _stream_validation_metrics(
    session: api.Session, trial_ids: List[int]
) -> Iterable[ValidationMetrics]:
    for i in bindings.get_GetValidationMetrics(session, trialIds=trial_ids):
        for m in i.metrics:
            yield ValidationMetrics._from_bindings(m)
