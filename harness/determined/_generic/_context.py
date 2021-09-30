import logging
from typing import Any, Optional

import appdirs

import determined as det
from determined import _generic, tensorboard
from determined.common import constants, storage
from determined.common.api import certs
from determined.common.experimental.session import Session

logger = logging.getLogger("determined.generic")


class Context:
    """
    generic.Context will someday evolve into a core part of the Generic API.
    """

    def __init__(
        self,
        distributed: _generic.DistributedContext,
        checkpointing: _generic.Checkpointing,
        preemption: _generic.Preemption,
        training: Optional[_generic.Training],
        searcher: Optional[_generic.AdvancedSearcher],
    ) -> None:
        self.distributed = distributed
        self.checkpointing = checkpointing
        self.preemption = preemption
        self._training = training
        # XXX: why is the AdvancedSearcher called ".searcher"?
        self._searcher = searcher

    @property
    def training(self) -> _generic.Training:
        assert (
            self._training
        ), "this generic.Context has no .training attribute, are you in a training task?"
        return self._training

    @property
    def searcher(self) -> _generic.AdvancedSearcher:
        assert (
            self._searcher
        ), "this generic.Context has no .searcher attribute, are you in a training task?"
        return self._searcher

    def __enter__(self) -> "Context":
        self.preemption.start()
        return self

    def __exit__(self, typ: type, value: Exception, tb: Any) -> None:
        self.preemption.close()
        # Detect some specific exceptions that are part of the user-facing API.
        if isinstance(value, det.InvalidHP):
            self.training.report_early_exit(_generic.EarlyExitReason.INVALID_HP)
            logger.info("InvalidHP detected during Trial init, converting InvalidHP to exit(0)")
            exit(0)


def _dummy_init(
    *,
    rank_info: Optional[_generic.RankInfo] = None,
    chief_ip: Optional[str] = None,
    # TODO: figure out a better way to deal with checkpointing in the local training case.
    storage_manager: Optional[storage.StorageManager] = None,
) -> Context:
    """
    Build a generic.Context suitable for running off-cluster.  This is normally called by init()
    when it is detected that there is no ClusterInfo available, but can be invoked directly for
    e.g. local test mode.
    """
    distributed = _generic.DistributedContext(rank_info=rank_info, chief_ip=chief_ip)
    preemption = _generic.DummyPreemption()

    if storage_manager is None:
        base_path = appdirs.user_data_dir("determined")
        logger.info("no storage_manager provided; storing checkpoints in {base_path}")
        storage_manager = storage.SharedFSStorageManager(base_path)
    checkpointing = _generic.DummyCheckpointing(distributed, storage_manager)

    # XXX: when running off-cluster, do we give a dummy Training/Searcher, or None?
    training = _generic.DummyTraining()
    searcher = _generic.DummyAdvancedSearcher()

    return Context(
        distributed=distributed,
        checkpointing=checkpointing,
        preemption=preemption,
        training=training,
        searcher=searcher,
    )


# The '*' is because we expect to add parameters to this method.  To keep a backwards-compatible
# API, we either need to always append to the parameters (preserving order of positional parameters)
# or force users to always use kwargs.  Since rank_info and chief_ip seem like crappy first
# parameters in a future state where we let you pass in a master_url, it seems that whether or not
# we choose to keep the kwarg-only requirement forever, these two should be kwarg-only for now.
def init(
    *,
    rank_info: Optional[_generic.RankInfo] = None,
    chief_ip: Optional[str] = None,
    # TODO: figure out a better way to deal with checkpointing in the local training case.
    storage_manager: Optional[storage.StorageManager] = None,
) -> Context:
    info = det.get_cluster_info()
    if info is None:
        return _dummy_init(rank_info=rank_info, chief_ip=chief_ip, storage_manager=storage_manager)

    distributed = _generic.DistributedContext(
        rank_info=rank_info,
        chief_ip=chief_ip,
        port_offset=info.task_type == "TRIAL" and info.trial._unique_port_offset or 0,
    )

    # We are on the cluster.
    cert = certs.default_load(info.master_url)
    session = Session(info.master_url, None, None, cert)

    # XXX: what would off-cluster tensorboard support look like?
    tbd_mgr = None
    tbd_writer = None
    # XXX: when running in non-trial tasks, do we give a dummy Training/Searcher, or None?
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

        training = _generic.Training(
            session,
            info.trial.trial_id,
            info.trial._trial_run_id,
            info.trial.experiment_id,
            tbd_mgr,
            tbd_writer,
        )
        searcher = _generic.AdvancedSearcher(
            session, info.trial.trial_id, info.trial._trial_run_id, info.allocation_id
        )

        # XXX: should we even allow users to override the checkpoint manager for trials?
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

        checkpointing = _generic.Checkpointing(
            distributed, storage_manager, session, api_path, static_metadata, tbd_mgr
        )

    else:
        # TODO: support checkpointing for non-trial tasks.
        if storage_manager is None:
            base_path = appdirs.user_data_dir("determined")
            logger.info("no storage_manager provided; storing checkpoints in {base_path}")
            storage_manager = storage.SharedFSStorageManager(base_path)
        checkpointing = _generic.DummyCheckpointing(distributed, storage_manager)

    preemption = _generic.Preemption(session, info.allocation_id, distributed)

    return Context(
        distributed=distributed,
        checkpointing=checkpointing,
        preemption=preemption,
        training=training,
        searcher=searcher,
    )
