import enum
import logging
import warnings
from typing import Any, Iterator, Optional

import determined as det
from determined import core
from determined.common import api

logger = logging.getLogger("determined.core")


class Unit(enum.Enum):
    EPOCHS = "EPOCHS"
    RECORDS = "RECORDS"
    BATCHES = "BATCHES"


def _parse_searcher_max_length(experiment_config: dict) -> Optional[int]:
    searcher = experiment_config.get("searcher", {})

    max_length = searcher.get("max_length")
    if max_length is None:
        return None

    if isinstance(max_length, int):
        return max_length

    # assume something like {"epochs": 10}
    assert isinstance(max_length, dict), max_length
    values = max_length.values()
    if not values:
        return None
    out = next(iter(values))
    assert isinstance(out, int), max_length
    return out


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
    .. warning::
        SearcherOperation is deprecated in 0.38.0, and will be removed in a future version.

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
        .. warning::
            SearcherOperation.length is deprecated in 0.38.0, and will be removed in a future
            version.  Instead, you should directly specify your training length in your training
            code.

        ``length`` represents the total amount of training which should be reached by the train step
        before the validate-report steps.
        """
        return self._length

    def report_progress(self, length: float) -> None:
        """
        .. warning::
            SearcherOperation.report_progress is deprecated in 0.38.0, and will be removed in a
            future version.  Instead, report progess with
            :meth:`~determined.core.TrainContext.report_progress`.

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
        # get the floating point progress
        progress = min(1.0, max(0.0, length / self._length))
        self._session.post(
            f"/api/v1/trials/{self._trial_id}/progress",
            data=det.util.json_encode({"progress": progress, "is_raw": True}),
        )

    def report_completed(self, searcher_metric: Any) -> None:
        """
        .. warning::
            SearcherOperation.report_completed is deprecated in 0.38.0, and will be removed in a
            future version.  Instead, just exit 0 when your training is complete.

        ``report_completed()`` is the final step of a train-validate-report cycle.

        ``report_completed()`` requires the value of the metric you are searching over.  This value
        is typically the output of the "validate" step of the train-validate-report cycle.
        In most cases `searcher_metric` should be a `float` but custom search methods
        may use any json-serializable type as searcher metric.
        """
        if not self._is_chief:
            raise RuntimeError("you must only call op.report_completed() from the chief worker")
        if self._completed:
            raise RuntimeError("you may only call op.report_completed() once")
        self._completed = True


class SearcherMode(enum.Enum):
    """
    .. warning::
        SearcherMode is deprecated in 0.38.0, and will be removed in a future version.

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
    .. warning::
        SearcherContext is deprecated in 0.38.0, and will be removed in a future version.  Instead
        of using ``SearcherContext.operations()`` to decide how long to train for, you should set
        your training length directly in your training code.

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

       for op in core_context.searcher.operations():
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
        max_length: int,
        units: Optional[Unit] = None,
    ) -> None:
        self._session = session
        self._dist = dist
        self._trial_id = trial_id
        self._length = max_length
        self._units = units

    def operations(
        self,
        searcher_mode: SearcherMode = SearcherMode.WorkersAskChief,
        auto_ack: bool = True,
    ) -> Iterator[SearcherOperation]:
        """
        .. warning::
            SearcherContext.operations is deprecated in 0.38.0, and will be removed in a future
            version.  Instead, you should set your training length directly in your training code.

        This method no longer talks to the Determined master; it just yields a single
        ``SearcherOperation`` objects based on the ``searcher.max_length`` in the experiment config
        (which is also deprecated).
        """

        warnings.warn(
            "SearcherContext.operations() was deprecated in Determined 0.38.0 and will be removed "
            "in a future version.  Instead, you should set your training length directly in your "
            "training code.",
            FutureWarning,
            stacklevel=2,
        )

        yield from self._operations(searcher_mode)

    def _operations(
        self,
        searcher_mode: SearcherMode = SearcherMode.WorkersAskChief,
    ) -> Iterator[SearcherOperation]:
        """
        The internal-only version of .operations which doesn't show a warning.

        This is meant to be called by other, deprecated things which internally depend on
        .operations() and have their own deprecation warning.  That way the user gets the
        deprecation warning for what they actually used.
        """

        searcher_mode = SearcherMode(searcher_mode)
        # Force the same synchronization behavior we used to have before fabricating operations.
        if self._dist.rank == 0:
            # Chief fabricates an op.
            op = SearcherOperation(self._session, self._trial_id, self._length, True)
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
                yield SearcherOperation(self._session, self._trial_id, op_length, False)

    def acknowledge_out_of_ops(self) -> None:
        """
        .. warning::
            SearcherContext.acknowledge_out_of_ops() is deprecated in 0.38.0, and will be removed in
            a future version.  Current calls to this function are ignored, and there should not need
            to be a replacement.
        """
        pass

    def get_configured_units(self) -> Optional[Unit]:
        """
        .. warning::
            SearcherContext.get_configured_units() is deprecated in 0.38.0, and will be removed in
            a future version.  Note that the ``searcher.max_length`` filed of the experiment config
            is also deprecated and will be removed as well.  Instead, you should directly specify
            your training length in your training code.

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
        logger.info(f"progress report: {length}/{self._length}")

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
        warnings.warn(
            "SearcherContext.operations() was deprecated in Determined 0.38.0 and will be removed "
            "in a future version.  Instead, you should set your training length directly in your "
            "training code.",
            FutureWarning,
            stacklevel=2,
        )

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


class SearcherContextMissing(SearcherContext):
    def __init__(self) -> None:
        pass

    def operations(
        self,
        searcher_mode: SearcherMode = SearcherMode.WorkersAskChief,
        auto_ack: bool = True,
    ) -> Iterator[SearcherOperation]:
        raise ValueError(
            "SearcherContext was not created because your experiment config does not have the "
            "searcher.max_length set.  Both the searcher.max_length and the SearcherContext are "
            "deprecated.  Instead, you should specify your training length directly in your "
            "training code and avoid all calls to core_context.searcher."
        )

    def acknowledge_out_of_ops(self) -> None:
        raise ValueError(
            "SearcherContext was not created because your experiment config does not have the "
            "searcher.max_length set.  Both the searcher.max_length and the SearcherContext are "
            "deprecated.  Instead, you should specify your training length directly in your "
            "training code and avoid all calls to core_context.searcher."
        )

    def get_configured_units(self) -> Optional[Unit]:
        raise ValueError(
            "SearcherContext was not created because your experiment config does not have the "
            "searcher.max_length set.  Both the searcher.max_length and the SearcherContext are "
            "deprecated.  Instead, you should specify your training length directly in your "
            "training code and avoid all calls to core_context.searcher."
        )
