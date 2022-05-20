import logging
from typing import Any, Optional

import appdirs

import determined as det
from determined import core, tensorboard
from determined.common import constants, storage
from determined.common.api import certs
from determined.common.experimental.session import Session, get_max_retries_config

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

    ``core.Context`` is a tool for integrating arbitrary distributed tasks into a Determined
    cluster.

    You should always use :meth:`core.init() <determined.core.init>` instead of creating a
    core.Context manually.
    """

    def __init__(
        self,
        checkpoint: core.CheckpointContext,
        distributed: Optional[core.DistributedContext] = None,
        preempt: Optional[core.PreemptContext] = None,
        train: Optional[core.TrainContext] = None,
        searcher: Optional[core.SearcherContext] = None,
    ) -> None:
        self.checkpoint = checkpoint
        self.distributed = distributed or core.DummyDistributedContext()
        self.preempt = preempt or core.DummyPreemptContext(self.distributed)
        self.train = train or core.DummyTrainContext()
        self.searcher = searcher or core.DummySearcherContext(self.distributed)

    def __enter__(self) -> "Context":
        self.preempt.start()
        return self

    def __exit__(self, typ: type, value: Exception, tb: Any) -> None:
        self.preempt.close()
        self.distributed.close()
        # Detect some specific exceptions that are part of the user-facing API.
        if isinstance(value, det.InvalidHP):
            self.train.report_early_exit(core.EarlyExitReason.INVALID_HP)
            logger.info("InvalidHP detected during Trial init, converting InvalidHP to exit(0)")
            exit(0)


def _dummy_init(
    *,
    distributed: Optional[core.DistributedContext] = None,
    # TODO(DET-6153): allow a Union[StorageManager, str] here.
    storage_manager: Optional[storage.StorageManager] = None,
    preempt_mode: core.PreemptMode = core.PreemptMode.WorkersAskChief,
) -> Context:
    """
    Build a core.Context suitable for running off-cluster.  This is normally called by init()
    when it is detected that there is no ClusterInfo available, but can be invoked directly for
    e.g. local test mode.
    """
    distributed = distributed or core.DummyDistributedContext()
    preempt = core.DummyPreemptContext(distributed, preempt_mode)

    if storage_manager is None:
        base_path = appdirs.user_data_dir("determined")
        logger.info("no storage_manager provided; storing checkpoints in {base_path}")
        storage_manager = storage.SharedFSStorageManager(base_path)
    checkpoint = core.DummyCheckpointContext(distributed, storage_manager)

    train = core.DummyTrainContext()
    searcher = core.DummySearcherContext(distributed)

    return Context(
        distributed=distributed,
        checkpoint=checkpoint,
        preempt=preempt,
        train=train,
        searcher=searcher,
    )


# The '*' is because we expect to add parameters to this method.  To keep a backwards-compatible
# API, we either need to always append to the parameters (preserving order of positional parameters)
# or force users to always use kwargs.  We haven't decided what the right positional arguments are
# yet, so the '*' lets us delay that decision until we are ready.
def init(
    *,
    distributed: Optional[core.DistributedContext] = None,
    # TODO: figure out a better way to deal with checkpointing in the local training case.
    storage_manager: Optional[storage.StorageManager] = None,
    preempt_mode: core.PreemptMode = core.PreemptMode.WorkersAskChief,
) -> Context:
    """
    ``core.init()`` builds a :class:`core.Context <determined.core.Context>` for use with the Core
    API.

    Always use ``with core.init() as context`` instead of instantiating a ``core.Context`` directly.
    Certain components of the Core API may be configured by passing arguments to ``core.init()``.
    The only arg that is required is a ``DistributedContext``, and even that is only required for
    for multi-slot tasks.

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
        storage_manager: Internal use only.
    """
    info = det.get_cluster_info()
    if info is None:
        return _dummy_init(distributed=distributed, storage_manager=storage_manager)

    # We are on the cluster.
    cert = certs.default_load(info.master_url)
    session = Session(info.master_url, None, None, cert, max_retries=get_max_retries_config())

    if distributed is None:
        if len(info.container_addrs) > 1 or len(info.slot_ids) > 1:
            raise ValueError("you must provide a valid DistributedContext for a multi-slot task")

    distributed = distributed or core.DummyDistributedContext()

    preempt = core.PreemptContext(session, info.allocation_id, distributed, preempt_mode)

    # At present, we only support tensorboards in Trial tasks.
    tbd_mgr = None
    tbd_writer = None

    train = None
    searcher = None

    if info.task_type == "TRIAL":
        # Prepare the tensorboard hooks.
        tbd_mgr = tensorboard.build(
            info.cluster_id,
            str(info.trial.experiment_id),
            str(info.trial.trial_id),
            info.trial._config["checkpoint_storage"],
            container_path=constants.SHARED_FS_CONTAINER_PATH,
        )
        tbd_writer = tensorboard.get_metric_writer()

        train = core.TrainContext(
            session,
            info.trial.trial_id,
            info.trial._trial_run_id,
            info.trial.experiment_id,
            tbd_mgr,
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

        checkpoint = core.CheckpointContext(
            distributed, storage_manager, session, info.task_id, info.allocation_id, tbd_mgr
        )

    else:
        # TODO: support checkpointing for non-trial tasks.
        if storage_manager is None:
            base_path = appdirs.user_data_dir("determined")
            logger.info("no storage_manager provided; storing checkpoints in {base_path}")
            storage_manager = storage.SharedFSStorageManager(base_path)
        checkpoint = core.DummyCheckpointContext(distributed, storage_manager)

    return Context(
        distributed=distributed,
        checkpoint=checkpoint,
        preempt=preempt,
        train=train,
        searcher=searcher,
    )
