import logging
import pathlib
import signal
import sys
import threading
import traceback
import types
from typing import Any, Dict, Optional, Union

import appdirs

import determined as det
from determined import core, tensorboard
from determined.common import api, constants, storage, util
from determined.common.api import authentication, bindings, certs
from determined.common.storage import shared

logger = logging.getLogger("determined.core")


class Context:
    """
    ``core.Context`` is a simple composition of several component APIs, with the following public
    members:

    -  ``.checkpoint``, a :class:`~CheckpointContext`
    -  ``.distributed``, a :class:`~DistributedContext`
    -  ``.preempt``, a :class:`~PreemptContext`
    -  ``.searcher``, a :class:`~SearcherContext`
    -  ``.train``, a :class:`~TrainContext`
    -  ``.profiler``, a :class:`~ProfilerContext`

    ``core.Context`` is a tool for integrating arbitrary distributed tasks into a Determined
    cluster.

    You should always use :meth:`core.init() <determined.core.init>` instead of creating a
    core.Context manually.
    """

    def __init__(
        self,
        checkpoint: core.CheckpointContext,
        _session: Optional[api.Session] = None,
        distributed: Optional[core.DistributedContext] = None,
        preempt: Optional[core.PreemptContext] = None,
        train: Optional[core.TrainContext] = None,
        searcher: Optional[core.SearcherContext] = None,
        info: Optional[det.ClusterInfo] = None,
        experimental: Optional[core.ExperimentalCoreContext] = None,
        profiler: Optional[core.ProfilerContext] = None,
        _metrics: Optional[core._MetricsContext] = None,
        _tensorboard_manager: Optional[tensorboard.TensorboardManager] = None,
        _heartbeat: Optional[core._Heartbeat] = None,
        _log_shipper: Optional[core._LogShipper] = None,
    ) -> None:
        self.checkpoint = checkpoint
        self.distributed = distributed or core.DummyDistributedContext()
        self.preempt = preempt or core.DummyPreemptContext(self.distributed)
        self.train = train or core.DummyTrainContext()
        self._metrics = _metrics or core._DummyMetricsContext()
        self.searcher = searcher or core.DummySearcherContext(self.distributed)
        self.info = info
        self.experimental = experimental or core.DummyExperimentalCoreContext()
        self.profiler = profiler or core.DummyProfilerContext()
        self._tensorboard_manager = _tensorboard_manager
        self._heartbeat = _heartbeat
        self._log_shipper = _log_shipper
        self._session = _session

    def start(self) -> None:
        if self._session is not None:
            self._session._persist_http_session()
        self.preempt.start()
        self._metrics.start()
        if self._tensorboard_manager is not None:
            self._tensorboard_manager.start()
        if self._heartbeat is not None:
            self._heartbeat.start()
        if self._log_shipper is not None:
            self._log_shipper.start()

    def __enter__(self) -> "Context":
        self.start()
        return self

    def close(
        self,
        exc_type: Optional[type] = None,
        exc_val: Optional[BaseException] = None,
        exc_tb: Optional[types.TracebackType] = None,
    ) -> None:
        self.preempt.close()
        self.distributed.close()
        self._metrics.close()
        self.profiler._close()
        if self._tensorboard_manager is not None:
            self._tensorboard_manager.close()
        if self._heartbeat is not None:
            self._heartbeat.close(exc_type, exc_val, exc_tb)
        if self._log_shipper is not None:
            self._log_shipper.close(exc_type, exc_val, exc_tb)
        if self._session is not None:
            self._session.close()

    def __exit__(
        self,
        exc_type: Optional[type],
        exc_val: Optional[BaseException],
        exc_tb: Optional[types.TracebackType],
    ) -> None:
        self.close(exc_type, exc_val, exc_tb)
        # Detect some specific exceptions that are part of the user-facing API.
        if isinstance(exc_val, det.InvalidHP):
            self.train.report_early_exit(core.EarlyExitReason.INVALID_HP)
            logger.info("InvalidHP detected during Trial init, converting InvalidHP to exit(0)")
            exit(0)


def _install_stacktrace_on_sigusr1() -> None:
    """Install a SIGUSR1 handler that prints a stack trace to stderr."""
    if not hasattr(signal, "SIGUSR1"):
        return

    # Signal handlers can only be registered on main threads.
    if threading.current_thread() is not threading.main_thread():
        return

    old_handler = None

    def stacktrace_on_sigusr1(signum: Any, frame: Any) -> None:
        traceback.print_stack(frame, file=sys.stderr)
        # old_handler may be None, SIG_IGN, or SIG_DFL.  It happens that SIG_DFL would be a noop for
        # SIGUSR1 so we don't have to worry about that case.
        if callable(old_handler):
            old_handler(signum, frame)

    old_handler = signal.signal(signal.SIGUSR1, stacktrace_on_sigusr1)


def _get_storage_manager(
    checkpoint_storage: Optional[Union[str, Dict[str, Any]]]
) -> Optional[storage.StorageManager]:
    if checkpoint_storage is None:
        return None
    if isinstance(checkpoint_storage, str):
        return storage.from_string(checkpoint_storage)
    if isinstance(checkpoint_storage, dict):
        if checkpoint_storage["type"] == "shared_fs":
            raise ValueError(
                "Cannot configure a shared_fs checkpoint storage with a "
                "dictionary. Use a string or a configuration file."
            )
        return storage.build(checkpoint_storage, container_path=None)
    raise TypeError("checkpoint_storage must be a string, dictionary, or None")


def _dummy_init(
    *,
    distributed: Optional[core.DistributedContext] = None,
    checkpoint_storage: Optional[Union[str, Dict[str, Any]]] = None,
    tensorboard_path: Optional[pathlib.Path] = None,
    preempt_mode: core.PreemptMode = core.PreemptMode.WorkersAskChief,
) -> Context:
    """
    Build a core.Context suitable for running off-cluster.  This is normally called by init()
    when it is detected that there is no ClusterInfo available, but can be invoked directly for
    e.g. local test mode.
    """
    distributed = distributed or core.DummyDistributedContext()
    preempt = core.DummyPreemptContext(distributed, preempt_mode)

    storage_manager = _get_storage_manager(checkpoint_storage)

    if storage_manager is None:
        base_path = appdirs.user_data_dir("determined")
        logger.info(f"no storage_manager provided; storing checkpoints in {base_path}")
        storage_manager = storage.SharedFSStorageManager(base_path)
    checkpoint = core.DummyCheckpointContext(distributed, storage_manager)

    train = core.DummyTrainContext(tensorboard_path)
    searcher = core.DummySearcherContext(distributed)
    profiler = core.DummyProfilerContext()

    _install_stacktrace_on_sigusr1()

    return Context(
        distributed=distributed,
        checkpoint=checkpoint,
        preempt=preempt,
        train=train,
        searcher=searcher,
        profiler=profiler,
    )


# The '*' is because we expect to add parameters to this method.  To keep a backwards-compatible
# API, we either need to always append to the parameters (preserving order of positional parameters)
# or force users to always use kwargs.  We haven't decided what the right positional arguments are
# yet, so the '*' lets us delay that decision until we are ready.
def init(
    *,
    distributed: Optional[core.DistributedContext] = None,
    checkpoint_storage: Optional[Union[str, Dict[str, Any]]] = None,
    preempt_mode: core.PreemptMode = core.PreemptMode.WorkersAskChief,
    tensorboard_mode: core.TensorboardMode = core.TensorboardMode.AUTO,
) -> Context:
    """
    ``core.init()`` builds a :class:`core.Context <determined.core.Context>` for use with the Core
    API.

    Always use ``with core.init() as context`` instead of instantiating a ``core.Context`` directly.
    Certain components of the Core API may be configured by passing arguments to ``core.init()``.
    The only arg that is required is a ``DistributedContext``, and even that is only required for
    multi-slot tasks.

    All of your training must occur within the scope of the ``with core.init() as core_context``, as
    there are resources necessary for training which start in the ``core.Context``'s ``__enter__``
    method and must be cleaned up in its ``__exit__()`` method.

    Arguments:
        distributed (``core.DistributedContext``, optional): Passing a ``DistributedContext`` is
            required for multi-slot training, but unnecessary for single-slot training.  Defaults to
            ``None``.
        preempt_mode (``core.PreemptMode``, optional): Configure the calling pattern for the
            ``core_context.preempt.should_preempt()`` method.  See
            :class:`~determined.core.PreemptMode` for more detail.  Defaults to ``WorkersAskChief``.
        checkpoint_storage (``Union[str, dict]``, optional): A directory path or a cloud storage URI
            of the form ``s3://<bucket>[/<prefix>]`` (AWS) or ``gs://<bucket>[/<prefix>]`` (GCP).
            This should only be used when IAM permissions can be assumed. You may also pass a
            dictionary matching the ``checkpoint_storage`` field of the experiment config, with the
            exception that ``type: shared_fs`` configs are not allowed.
        tensorboard_mode (``core.TensorboardMode``, optional): Define how Tensorboard
            metrics and profiling data are retained. See
            :class:`~determined.core.TensorboardMode`` for more detail. Defaults to ``AUTO``.
    """
    info = det.get_cluster_info()
    if info is None:
        return _dummy_init(
            distributed=distributed,
            checkpoint_storage=checkpoint_storage,
        )

    # We are on the cluster.
    cert = certs.default_load(info.master_url)
    session = authentication.login_with_cache(info.master_url, cert=cert).with_retry(
        util.get_max_retries_config()
    )

    if distributed is None:
        if len(info.container_addrs) > 1 or len(info.slot_ids) > 1:
            raise ValueError("you must provide a valid DistributedContext for a multi-slot task")

    distributed = distributed or core.DummyDistributedContext()

    # At present, we only support tensorboards in Trial tasks.
    tbd_writer = None

    train = None
    searcher = None
    tensorboard_manager = None
    experimental = None
    profiler = None
    metrics = None

    storage_manager = _get_storage_manager(checkpoint_storage)

    if info.task_type == "TRIAL":
        # Prepare the tensorboard hooks.
        tensorboard_manager = tensorboard.build(
            info.cluster_id,
            str(info.trial.experiment_id),
            str(info.trial.trial_id),
            info.trial._config["checkpoint_storage"],
            container_path=constants.SHARED_FS_CONTAINER_PATH,
            async_upload=True,
            sync_on_close=(tensorboard_mode == core.TensorboardMode.AUTO),
        )
        if tensorboard_mode == core.TensorboardMode.AUTO:
            tbd_writer = tensorboard.get_metric_writer()

        metrics = core._MetricsContext(
            session,
            info.trial.trial_id,
            info.trial._trial_run_id,
        )

        train = core.TrainContext(
            session,
            info.trial.trial_id,
            info.trial.experiment_id,
            metrics,
            distributed,
            tensorboard_mode,
            tensorboard_manager,
            tbd_writer,
        )

        units = core._parse_searcher_units(info.trial._config)
        searcher = core.SearcherContext(
            session,
            distributed,
            info.trial.trial_id,
            info.trial._trial_run_id,
            info.allocation_id,
            units,
        )

        if storage_manager is None:
            storage_manager = storage.build(
                info.trial._config["checkpoint_storage"],
                container_path=constants.SHARED_FS_CONTAINER_PATH,
            )

        storage_used = checkpoint_storage
        if storage_used is None:
            storage_used = info.trial._config["checkpoint_storage"]

        run_prepare_response = _run_prepare(distributed, session, info.trial.trial_id, storage_used)

        checkpoint = core.CheckpointContext(
            distributed,
            storage_manager,
            session,
            info.task_id,
            info.allocation_id,
            tensorboard_mode,
            tensorboard_manager,
            run_prepare_response.storageId,
        )

        preempt = core.PreemptContext(session, info.allocation_id, distributed, preempt_mode)
        experimental = core.ExperimentalCoreContext(session, info.trial.trial_id)
        profiler = core.ProfilerContext(
            agent_id=info.agent_id,
            metrics=metrics,
            distributed=distributed,
        )

    else:
        # TODO: support checkpointing for non-trial tasks.
        if storage_manager is None:
            base_path = appdirs.user_data_dir("determined")
            logger.info(f"no storage_manager provided; storing checkpoints in {base_path}")
            storage_manager = storage.SharedFSStorageManager(base_path)
        checkpoint = core.DummyCheckpointContext(distributed, storage_manager)
        preempt = core.DummyPreemptContext(distributed, preempt_mode)

    _install_stacktrace_on_sigusr1()

    return Context(
        distributed=distributed,
        checkpoint=checkpoint,
        preempt=preempt,
        train=train,
        searcher=searcher,
        experimental=experimental,
        profiler=profiler,
        _metrics=metrics,
        _tensorboard_manager=tensorboard_manager,
        _session=session,
    )


def _run_prepare(
    distributed: Optional[core.DistributedContext],
    sess: api.Session,
    run_id: int,
    checkpoint_storage: Optional[Union[str, Dict[str, Any]]],
) -> bindings.v1RunPrepareForReportingResponse:
    cs = None
    if isinstance(checkpoint_storage, str):
        cs = shared._shortcut_to_config(checkpoint_storage)
    elif isinstance(checkpoint_storage, dict):
        cs = checkpoint_storage

    return bindings.post_RunPrepareForReporting(
        sess,
        body=bindings.v1RunPrepareForReportingRequest(
            runId=run_id,
            checkpointStorage=cs,
        ),
    )
