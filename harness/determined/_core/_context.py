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
    core.Context is a simple composition of several other APIs.
    """

    def __init__(
        self,
        checkpointing: _core.Checkpointing,
        distributed: Optional[_core.DistributedContext] = None,
        preemption: Optional[_core.Preemption] = None,
        training: Optional[_core.Training] = None,
        searcher: Optional[_core.Searcher] = None,
    ) -> None:
        self.checkpointing = checkpointing
        self.distributed = distributed or _core.DummyDistributed()
        self.preemption = preemption or _core.DummyPreemption()
        self.training = training or _core.DummyTraining()
        self.searcher = searcher or _core.DummySearcher()

    def __enter__(self) -> "Context":
        self.preemption.start()
        return self

    def __exit__(self, typ: type, value: Exception, tb: Any) -> None:
        self.preemption.close()
        self.distributed.close()
        # Detect some specific exceptions that are part of the user-facing API.
        if isinstance(value, det.InvalidHP):
            self.training.report_early_exit(_core.EarlyExitReason.INVALID_HP)
            logger.info("InvalidHP detected during Trial init, converting InvalidHP to exit(0)")
            exit(0)


def _dummy_init(
    *,
    distributed: Optional[_core.DistributedContext] = None,
    # TODO(DET-6153): allow a Union[StorageManager, str] here.
    storage_manager: Optional[storage.StorageManager] = None,
) -> Context:
    """
    Build a core.Context suitable for running off-cluster.  This is normally called by init()
    when it is detected that there is no ClusterInfo available, but can be invoked directly for
    e.g. local test mode.
    """
    distributed = distributed or _core.DummyDistributed()
    preemption = _core.DummyPreemption()

    if storage_manager is None:
        base_path = appdirs.user_data_dir("determined")
        logger.info("no storage_manager provided; storing checkpoints in {base_path}")
        storage_manager = storage.SharedFSStorageManager(base_path)
    checkpointing = _core.DummyCheckpointing(distributed, storage_manager)

    training = _core.DummyTraining()
    searcher = _core.DummySearcher()

    return Context(
        distributed=distributed,
        checkpointing=checkpointing,
        preemption=preemption,
        training=training,
        searcher=searcher,
    )


# The '*' is because we expect to add parameters to this method.  To keep a backwards-compatible
# API, we either need to always append to the parameters (preserving order of positional parameters)
# or force users to always use kwargs.  We haven't decided what the right positional arguments are
# yet, so the '*' lets us delay that decision until we are ready.
def init(
    *,
    distributed: Optional[_core.DistributedContext] = None,
    # TODO: figure out a better way to deal with checkpointing in the local training case.
    storage_manager: Optional[storage.StorageManager] = None,
) -> Context:
    info = det.get_cluster_info()
    if info is None:
        return _dummy_init(distributed=distributed, storage_manager=storage_manager)

    # We are on the cluster.
    cert = certs.default_load(info.master_url)
    session = Session(info.master_url, None, None, cert)

    distributed = distributed or _core.DummyDistributed()

    naddrs = len(info.container_addrs)
    if naddrs > 1 and isinstance(distributed, _core.DummyDistributed):
        raise ValueError("you must provide a valid DistributedContext for a multi-container task")

    preemption = _core.Preemption(session, info.allocation_id, distributed)

    # At present, we only support tensorboards in Trial tasks.
    tbd_mgr = None
    tbd_writer = None

    training = None
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

        training = _core.Training(
            session,
            info.trial.trial_id,
            info.trial._trial_run_id,
            info.trial.experiment_id,
            tbd_mgr,
            tbd_writer,
        )
        searcher = _core.Searcher(
            session, info.trial.trial_id, info.trial._trial_run_id, info.allocation_id
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

        checkpointing = _core.Checkpointing(
            distributed, storage_manager, session, api_path, static_metadata, tbd_mgr
        )

    else:
        # TODO: support checkpointing for non-trial tasks.
        if storage_manager is None:
            base_path = appdirs.user_data_dir("determined")
            logger.info("no storage_manager provided; storing checkpoints in {base_path}")
            storage_manager = storage.SharedFSStorageManager(base_path)
        checkpointing = _core.DummyCheckpointing(distributed, storage_manager)

    return Context(
        distributed=distributed,
        checkpointing=checkpointing,
        preemption=preemption,
        training=training,
        searcher=searcher,
    )
