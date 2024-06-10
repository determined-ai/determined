import datetime
import enum
import inspect
import warnings
from typing import Any, Dict, Iterable, List, Optional, Union

from determined.common import api, util
from determined.common.api import bindings, logs
from determined.common.experimental import checkpoint, experiment, metrics

# TODO (MLG-1087): move OrderBy to experimental.client namespace
from determined.common.experimental._util import OrderBy  # noqa: I2041


class LogLevel(enum.Enum):
    TRACE = bindings.v1LogLevel.TRACE.value
    DEBUG = bindings.v1LogLevel.DEBUG.value
    INFO = bindings.v1LogLevel.INFO.value
    WARNING = bindings.v1LogLevel.WARNING.value
    ERROR = bindings.v1LogLevel.ERROR.value
    CRITICAL = bindings.v1LogLevel.CRITICAL.value

    def _to_bindings(self) -> bindings.v1LogLevel:
        return bindings.v1LogLevel(self.value)


class TrialState(enum.Enum):
    # UNSPECIFIED is internal to the bound API and is not be exposed to the front end
    ACTIVE = bindings.trialv1State.ACTIVE.value
    PAUSED = bindings.trialv1State.PAUSED.value
    STOPPING_CANCELED = bindings.trialv1State.STOPPING_CANCELED.value
    STOPPING_KILLED = bindings.trialv1State.STOPPING_KILLED.value
    STOPPING_COMPLETED = bindings.trialv1State.STOPPING_COMPLETED.value
    STOPPING_ERROR = bindings.trialv1State.STOPPING_ERROR.value
    CANCELED = bindings.trialv1State.CANCELED.value
    COMPLETED = bindings.trialv1State.COMPLETED.value
    ERROR = bindings.trialv1State.ERROR.value
    QUEUED = bindings.trialv1State.QUEUED.value
    PULLING = bindings.trialv1State.PULLING.value
    STARTING = bindings.trialv1State.STARTING.value
    RUNNING = bindings.trialv1State.RUNNING.value


class Trial:
    """
    A class representing a Trial object.

    A Trial object is usually obtained from :func:`determined.experimental.client.get_trial`.
    Trial reference class used for querying relevant :class:`~determined.experimental.Checkpoint`
    instances.

    Attributes:
        trial_id: ID of trial.
        session: HTTP request session.
        experiment_id: (Mutable, Optional[int]) ID of the experiment this trial belongs to.
        hparams: (Mutable, Optional[Dict]) Dict[name, value] of the trial's hyperparameters.
            This is an instance of the hyperparameter space defined by the experiment.
        state: (Mutable, Optional[TrialState]) Trial state (ex: ACTIVE, PAUSED, COMPLETED).
        summary_metrics: (Mutable, Optional[Dict]) Summary metrics for the trial. Includes
            aggregated metrics for training and validation steps for each reported metric name.
            Example:

            .. code::

                {
                    "avg_metrics": {
                        "loss": {
                            "count": 100,
                            "last": 0.2,
                            "max": 0.4,
                            "min", 0.2,
                            "sum": 1.2,
                            "type": "number",
                        }
                }

    Note:
        All attributes are cached by default.

        The :attr:`hparams` and :attr:`summary_metrics` attributes are mutable and may be changed
        by methods that update these values, either automatically or explicitly with :meth:`reload`.

    """

    def __init__(self, trial_id: int, session: api.Session):
        self.id = trial_id
        self._session = session

        self.experiment_id: Optional[int] = None
        self.hparams: Optional[Dict[str, Any]] = None
        self.summary_metrics: Optional[Dict[str, Any]] = None
        self.state: Optional[TrialState] = None

    def iter_logs(
        self,
        follow: bool = False,
        *,
        head: Optional[int] = None,
        tail: Optional[int] = None,
        container_ids: Optional[List[str]] = None,
        rank_ids: Optional[List[int]] = None,
        stdtypes: Optional[List[str]] = None,
        min_level: Optional[LogLevel] = None,
        timestamp_before: Optional[Union[str, int]] = None,
        timestamp_after: Optional[Union[str, int]] = None,
        sources: Optional[List[str]] = None,
        search_text: Optional[str] = None,
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
            min_level (LogLevel, optional): When set, defines the minimum log priority for lines
                that will be returned.  Defaults to ``None`` (all logs returned).
            timestamp_before (Union[str, int], optional): Specifies a timestamp that returns only
                logs before a certain time. Accepts either a string in RFC 3339 format
                (eg. ``2021-10-26T23:17:12Z``) or an int representing the epoch second.
            timestamp_after (Union[str, int], optional): Specifies a timestamp that returns only
                logs after a certain time. Accepts either a string in RFC 3339 format
                (eg. ``2021-10-26T23:17:12Z``) or an int representing the epoch second.
            sources (List[str], optional): When set, returns only logs originating from specified
                node name(s) (eg. ``master`` or ``agent``).
            search_text (str, Optional): Filters individual logs to only return logs containing
                the specified string.

        """
        if head is not None and head < 0:
            raise ValueError(f"head must be non-negative, got {head}")
        if tail is not None and tail < 0:
            raise ValueError(f"tail must be non-negative, got {tail}")

        if (
            timestamp_before is not None
            and not isinstance(timestamp_before, (str, int))
            or timestamp_after is not None
            and not isinstance(timestamp_after, (str, int))
        ):
            raise ValueError(
                "timestamp_before and timestamp_after must be either str or int types."
            )

        # Validate and convert epoch timestamps to RFC 3339-formatted datetime strings.
        if isinstance(timestamp_before, str) and not util.is_protobuf_timestamp(timestamp_before):
            raise ValueError(f"Timestamp {timestamp_before} has an invalid format.")
        if isinstance(timestamp_after, str) and not util.is_protobuf_timestamp(timestamp_after):
            raise ValueError(f"Timestamp {timestamp_after} has an invalid format.")

        if isinstance(timestamp_before, int):
            datetime_before = datetime.datetime.fromtimestamp(timestamp_before)
            timestamp_before = datetime_before.isoformat("T") + "Z"
        if isinstance(timestamp_after, int):
            datetime_after = datetime.datetime.fromtimestamp(timestamp_after)
            timestamp_after = datetime_after.isoformat("T") + "Z"

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
            sources=sources,
            stdtypes=stdtypes,
            min_level=None if min_level is None else min_level._to_bindings(),
            timestamp_before=timestamp_before,
            timestamp_after=timestamp_after,
            search_text=search_text,
        ):
            yield log.message

    def logs(self, *args: Any, **kwargs: Any) -> Iterable[str]:
        """DEPRECATED: Use iter_logs instead."""
        warnings.warn(
            "Trial.logs() has been deprecated and will be removed in a future version."
            "Please call Trial.iter_logs() instead.",
            FutureWarning,
            stacklevel=2,
        )
        return self.iter_logs(*args, **kwargs)

    # type-checking suppression can be removed when mypy issue #12472 is resolved (or logs removed)
    logs.__signature__ = inspect.signature(iter_logs)  # type: ignore

    def kill(self) -> None:
        bindings.post_KillTrial(self._session, id=self.id)

    def list_checkpoints(
        self,
        sort_by: Optional[Union[str, checkpoint.CheckpointSortBy]] = None,
        order_by: Optional[OrderBy] = None,
        max_results: Optional[int] = None,
    ) -> List[checkpoint.Checkpoint]:
        """Returns an iterator of sorted :class:`~determined.experimental.Checkpoint` instances.

        Requires either both `sort_by` and `order_by` to be defined, or neither. If neither are
        specified, will default to sorting by the experiment's configured searcher metric, and
        ordering by `smaller_is_better`.

        Only checkpoints in a ``COMPLETED`` state with a matching ``COMPLETED`` validation
        are considered.

        Arguments:
            sort_by: (Optional) Parameter to sort checkpoints by. Accepts either
                ``checkpoint.CheckpointSortBy`` or a string representing a validation metric name.
            order_by: (Optional) Order of sorted checkpoints (ascending or descending).
            max_results: (Optional) Maximum number of results to return. Defaults to no maximum.

        Returns:
            A list of sorted and ordered checkpoints.
        """
        if (sort_by is None) != (order_by is None):
            raise AssertionError("sort_by and order_by must be either both set, or neither.")

        if sort_by and not isinstance(sort_by, (checkpoint.CheckpointSortBy, str)):
            raise ValueError("sort_by must be of type CheckpointSortBy or str")

        if not sort_by:
            sort_by = checkpoint.CheckpointSortBy.SEARCHER_METRIC

        def get_trial_checkpoints(offset: int) -> bindings.v1GetTrialCheckpointsResponse:
            return bindings.get_GetTrialCheckpoints(
                self._session,
                id=self.id,
                orderBy=order_by._to_bindings() if order_by else None,
                sortByAttr=sort_by._to_bindings()
                if isinstance(sort_by, checkpoint.CheckpointSortBy)
                else None,
                sortByMetric=sort_by if isinstance(sort_by, str) else None,
                offset=offset,
                limit=max_results,
            )

        resps = api.read_paginated(
            get_with_offset=get_trial_checkpoints,
            pages=api.PageOpts.single if max_results else api.PageOpts.all,
        )

        return [
            checkpoint.Checkpoint._from_bindings(c, self._session)
            for r in resps
            for c in r.checkpoints
        ]

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
        warnings.warn(
            "Trial.top_checkpoint() has been deprecated and will be removed in a future "
            "version."
            "Please call Trial.list_checkpoints(...,max_results=1) instead.",
            FutureWarning,
            stacklevel=2,
        )
        order_by = None
        if sort_by:
            order_by = OrderBy.ASC if smaller_is_better else OrderBy.DESC

        checkpoints = self.list_checkpoints(
            sort_by=sort_by,
            order_by=order_by,
            max_results=1,
        )
        if not checkpoints:
            raise ValueError("No checkpoints found for criteria.")
        return checkpoints[0]

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
        warnings.warn(
            "Trial.select_checkpoint() has been deprecated and will be removed in a future "
            "version."
            "Please call Trial.list_checkpoints() instead.",
            FutureWarning,
            stacklevel=2,
        )
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
            sort_by = checkpoint.CheckpointSortBy.BATCH_NUMBER  # type: ignore
            order_by = OrderBy.DESC

        if sort_by:
            order_by = OrderBy.ASC if smaller_is_better else OrderBy.DESC

        checkpoints = self.list_checkpoints(sort_by=sort_by, order_by=order_by, max_results=1)

        if not checkpoints:
            raise ValueError("No checkpoints found for criteria.")
        return checkpoints[0]

    def get_checkpoints(
        self,
        sort_by: Optional[Union[str, checkpoint.CheckpointSortBy]] = None,
        order_by: Optional[OrderBy] = None,
    ) -> List[checkpoint.Checkpoint]:
        """
        Return a list of :class:`~determined.experimental.Checkpoint` instances for the current
        trial.

        Either ``sort_by`` and ``order_by`` are both specified or neither are.

        Arguments:
            sort_by (string, :class:`~determined.experimental.CheckpointSortBy`): Which field to
                sort by. Strings are assumed to be validation metric names.
            order_by (:class:`~determined.experimental.OrderBy`): Whether to sort in
                ascending or descending order.
        """

        warnings.warn(
            "Trial.get_checkpoints() has been deprecated and will be removed in a future "
            "version."
            "Please call Experiment.list_checkpoints() instead.",
            FutureWarning,
            stacklevel=2,
        )

        if (sort_by is None) != (order_by is None):
            raise AssertionError("sort_by and order_by must be set together")

        def get_trial_checkpoints(offset: int) -> bindings.v1GetTrialCheckpointsResponse:
            return bindings.get_GetTrialCheckpoints(
                self._session,
                id=self.id,
                orderBy=order_by._to_bindings() if order_by else None,
                sortByAttr=sort_by._to_bindings()
                if isinstance(sort_by, checkpoint.CheckpointSortBy)
                else None,
                offset=offset,
            )

        resps = api.read_paginated(get_trial_checkpoints)

        checkpoints = [
            checkpoint.Checkpoint._from_bindings(c, self._session)
            for r in resps
            for c in r.checkpoints
        ]

        # If sort_by was a defined field, we already sorted and ordered.
        if isinstance(sort_by, checkpoint.CheckpointSortBy) or not checkpoints:
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
            order_by = OrderBy.ASC if smaller_is_better else OrderBy.DESC

        assert sort_by is not None and order_by is not None, "sort_by and order_by not defined."

        reverse = order_by == OrderBy.DESC

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

    def stream_metrics(self, group: str) -> Iterable[metrics.TrialMetrics]:
        """Streams metrics for this trial.

        DEPRECATED: Use iter_metrics instead
        """
        warnings.warn(
            "Experiment.stream_metrics() has been deprecated and will be removed in a"
            " future version. Please call Experiment.iter_trials() instead.",
            FutureWarning,
            stacklevel=2,
        )
        return self.iter_metrics(group)

    def iter_metrics(self, group: str) -> Iterable[metrics.TrialMetrics]:
        """Generate an iterator of metrics for this trial.

        Arguments:
            group: The metric group to iterate over.  Common values are "validation" and "training",
                but group can be any value passed to master when reporting metrics during training
                (usually via a context's `report_metrics`).

        Returns:
            An iterable of :class:`~determined.experimental.TrialMetrics` objects.
        """
        return _stream_trials_metrics(self._session, [self.id], group=group)

    def _hydrate(self, trial: bindings.trialv1Trial) -> None:
        self.experiment_id = trial.experimentId
        self.hparams = trial.hparams
        self.state = TrialState(trial.state.value)
        self.summary_metrics = trial.summaryMetrics

    def reload(self) -> None:
        """
        Explicit refresh of cached properties.
        """
        resp = bindings.get_GetTrial(session=self._session, trialId=self.id).trial
        self._hydrate(resp)

    def stream_training_metrics(self) -> Iterable[metrics.TrainingMetrics]:
        """Streams training metrics for this trial.

        DEPRECATED: Use iter_metrics instead with `group` set to "training"
        """
        warnings.warn(
            "Trial.stream_training_metrics is deprecated."
            "Use Trial.iter_metrics instead with `group` set to 'training'",
            FutureWarning,
            stacklevel=2,
        )

        return _stream_training_metrics(self._session, [self.id])

    def stream_validation_metrics(self) -> Iterable[metrics.ValidationMetrics]:
        """Streams validation metrics for this trial.

        DEPRECATED: Use iter_metrics instead with `group` set to "validation"
        """
        warnings.warn(
            "Trial.stream_validation_metrics is deprecated."
            "Use Trial.iter_metrics instead with `group` set to 'validation'",
            FutureWarning,
            stacklevel=2,
        )
        return _stream_validation_metrics(self._session, [self.id])

    def get_experiment(self) -> "experiment.Experiment":
        """Return the parent :class:`~determined.experimental.Experiment` for this trial."""
        if not self.experiment_id:
            # In the case that Trial was constructed manually, reload to populate attributes.
            self.reload()

        assert self.experiment_id  # for mypy
        resp = bindings.get_GetExperiment(session=self._session, experimentId=self.experiment_id)
        return experiment.Experiment._from_bindings(resp.experiment, self._session)

    @classmethod
    def _from_bindings(cls, trial_bindings: bindings.trialv1Trial, session: api.Session) -> "Trial":
        trial = cls(trial_bindings.id, session)
        trial._hydrate(trial_bindings)
        return trial


# This is to shorten line lengths of the TrialSortBy definition.
_tsb = bindings.v1GetExperimentTrialsRequestSortBy


class TrialSortBy(enum.Enum):
    """
    Specifies the field to sort a list of trials on.
    """

    ID = _tsb.ID.value
    START_TIME = _tsb.START_TIME.value
    END_TIME = _tsb.END_TIME.value
    STATE = _tsb.STATE.value
    BEST_VALIDATION_METRIC = _tsb.BEST_VALIDATION_METRIC.value
    LATEST_VALIDATION_METRIC = _tsb.LATEST_VALIDATION_METRIC.value
    BATCHES_PROCESSED = _tsb.BATCHES_PROCESSED.value
    DURATION = _tsb.DURATION.value
    RESTARTS = _tsb.RESTARTS.value
    CHECKPOINT_SIZE = _tsb.CHECKPOINT_SIZE.value
    LOG_RETENTION_DAYS = _tsb.LOG_RETENTION_DAYS.value

    def _to_bindings(self) -> bindings.v1GetExperimentTrialsRequestSortBy:
        return _tsb(self.value)


class TrialOrderBy(enum.Enum):
    """
    Specifies whether a sorted list of trials should be in ascending or
    descending order.

    This class is deprecated in favor of ``OrderBy`` and will be removed in a future
    release.
    """

    def __getattribute__(self, name: str) -> Any:
        warnings.warn(
            "'TrialOrderBy' is deprecated and will be removed in a future "
            "release. Please use 'experimental.OrderBy' instead.",
            FutureWarning,
            stacklevel=1,
        )
        return super().__getattribute__(name)

    ASCENDING = bindings.v1OrderBy.ASC.value
    ASC = bindings.v1OrderBy.ASC.value
    DESCENDING = bindings.v1OrderBy.DESC.value
    DESC = bindings.v1OrderBy.DESC.value

    def _to_bindings(self) -> bindings.v1OrderBy:
        return bindings.v1OrderBy(self.value)


def _stream_trials_metrics(
    session: api.Session, trial_ids: List[int], group: str
) -> Iterable[metrics.TrialMetrics]:
    for i in bindings.get_GetMetrics(session, trialIds=trial_ids, group=group):
        for m in i.metrics:
            yield metrics.TrialMetrics._from_bindings(m, group=group)


def _stream_training_metrics(
    session: api.Session, trial_ids: List[int]
) -> Iterable[metrics.TrainingMetrics]:
    for i in bindings.get_GetTrainingMetrics(session, trialIds=trial_ids):
        for m in i.metrics:
            yield metrics.TrainingMetrics._from_bindings(m)


def _stream_validation_metrics(
    session: api.Session, trial_ids: List[int]
) -> Iterable[metrics.ValidationMetrics]:
    for i in bindings.get_GetValidationMetrics(session, trialIds=trial_ids):
        for m in i.metrics:
            yield metrics.ValidationMetrics._from_bindings(m)


class TrialReference(Trial):
    """A legacy class representing an Trial object.

    This class was renamed to :class:`~determined.experimental.Trial` and will be removed
    in a future release.
    """

    def __init__(
        self,
        trial_id: int,
        session: api.Session,
    ):
        warnings.warn(
            "'TrialReference' was renamed to 'Trial' and will be removed in a future "
            "release. Please consider replacing any code references to 'TrialReference' "
            "with 'Trial'.",
            FutureWarning,
            stacklevel=2,
        )
        Trial.__init__(self, trial_id=trial_id, session=session)
