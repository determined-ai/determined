import logging
from typing import Any, Optional

import appdirs

import determined as det
from determined import _core, tensorboard
from determined.common import constants, storage
from determined.common.api import certs
from determined.common.experimental.session import Session

logger = logging.getLogger("determined.core")


class Context:
    """
    core.Context is a simple composition of several component APIs.

    core.Context is a tool for integrating arbitrary distributed tasks into a Determined cluster.

    You should always use core.init() instead of creating a core.Context manually.
    """

    def __init__(
        self,
        checkpoint: _core.CheckpointContext,
        distributed: Optional[_core.DistributedContext] = None,
        preempt: Optional[_core.PreemptContext] = None,
        train: Optional[_core.TrainContext] = None,
        searcher: Optional[_core.SearcherContext] = None,
    ) -> None:
        self.checkpoint = checkpoint
        self.distributed = distributed or _core.DummyDistributedContext()
        self.preempt = preempt or _core.DummyPreemptContext(self.distributed)
        self.train = train or _core.DummyTrainContext()
        self.searcher = searcher or _core.DummySearcherContext(self.distributed)

    def __enter__(self) -> "Context":
        self.preempt.start()
        return self

    def __exit__(self, typ: type, value: Exception, tb: Any) -> None:
        self.preempt.close()
        self.distributed.close()
        # Detect some specific exceptions that are part of the user-facing API.
        if isinstance(value, det.InvalidHP):
            self.train.report_early_exit(_core.EarlyExitReason.INVALID_HP)
            logger.info("InvalidHP detected during Trial init, converting InvalidHP to exit(0)")
            exit(0)


def _dummy_init(
    *,
    distributed: Optional[_core.DistributedContext] = None,
    # TODO(DET-6153): allow a Union[StorageManager, str] here.
    storage_manager: Optional[storage.StorageManager] = None,
    preempt_mode: _core.PreemptMode = _core.PreemptMode.WorkersAskChief,
) -> Context:
    """
    Build a core.Context suitable for running off-cluster.  This is normally called by init()
    when it is detected that there is no ClusterInfo available, but can be invoked directly for
    e.g. local test mode.
    """
    distributed = distributed or _core.DummyDistributedContext()
    preempt = _core.DummyPreemptContext(distributed, preempt_mode)

    if storage_manager is None:
        base_path = appdirs.user_data_dir("determined")
        logger.info("no storage_manager provided; storing checkpoints in {base_path}")
        storage_manager = storage.SharedFSStorageManager(base_path)
    checkpoint = _core.DummyCheckpointContext(distributed, storage_manager)

    train = _core.DummyTrainContext()
    searcher = _core.DummySearcherContext(distributed)

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
    distributed: Optional[_core.DistributedContext] = None,
    preempt_mode: _core.PreemptMode = _core.PreemptMode.WorkersAskChief,
    # TODO: figure out a better way to deal with checkpointing in the local training case.
    storage_manager: Optional[storage.StorageManager] = None,
) -> Context:
    """
    core.init() builds a core.Context for use with the Core API.

    Always use core.init() instead of instantiating a core.Context directly.  Certain components of
    the Core API may be configured directly by passing arguments to core.init().  The only arg that
    is required is a DistributedContext, and even that is only required for for multi-slot tasks.

    Arguments:
        distributed (``core.DistributedContext``, default: ``None``): Passing a DistributedContext
            is required for multi-slot training, but unnecessary for single-slot training.
        preempt_mode (``core.PreemptMode``, default: ``WorkersAskChief``): Configure the calling
            pattern for the core_context.preempt.should_preempt() method.  See
            :class:`~determined.core.PremptMode` for more detail.
        storage_manager: Internal use only.
    """
    info = det.get_cluster_info()
    if info is None:
        return _dummy_init(distributed=distributed, storage_manager=storage_manager)

    # We are on the cluster.
    cert = certs.default_load(info.master_url)
    session = Session(info.master_url, None, None, cert)

    if distributed is None:
        if len(info.container_addrs) > 1 or len(info.slot_ids) > 1:
            raise ValueError("you must provide a valid DistributedContext for a multi-slot task")

    distributed = distributed or _core.DummyDistributedContext()

    preempt = _core.PreemptContext(session, info.allocation_id, distributed, preempt_mode)

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

        train = _core.TrainContext(
            session,
            info.trial.trial_id,
            info.trial._trial_run_id,
            info.trial.experiment_id,
            tbd_mgr,
            tbd_writer,
        )
        units = _core._parse_searcher_units(info.trial._config)
        searcher = _core.SearcherContext(
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

        api_path = f"/api/v1/trials/{info.trial.trial_id}/checkpoint_metadata"
        static_metadata = {
            "trial_id": info.trial.trial_id,
            "trial_run_id": info.trial._trial_run_id,
        }

        checkpoint = _core.CheckpointContext(
            distributed, storage_manager, session, api_path, static_metadata, tbd_mgr
        )

    else:
        # TODO: support checkpointing for non-trial tasks.
        if storage_manager is None:
            base_path = appdirs.user_data_dir("determined")
            logger.info("no storage_manager provided; storing checkpoints in {base_path}")
            storage_manager = storage.SharedFSStorageManager(base_path)
        checkpoint = _core.DummyCheckpointContext(distributed, storage_manager)

    return Context(
        distributed=distributed,
        checkpoint=checkpoint,
        preempt=preempt,
        train=train,
        searcher=searcher,
    )
