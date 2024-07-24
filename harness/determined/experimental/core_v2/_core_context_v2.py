import logging
import pathlib
from typing import Any, Dict, Optional, Union

import appdirs

import determined as det
from determined import core, experimental, tensorboard
from determined.common import constants, storage, util
from determined.common.api import authentication, certs

logger = logging.getLogger("determined.core")


def _default_storage_manager() -> storage.SharedFSStorageManager:
    base_path = pathlib.Path(appdirs.user_data_dir("determined")) / "checkpoints"
    if not base_path.exists():
        base_path.mkdir(parents=True)

    logger.info(f"no storage_manager provided; storing checkpoints in {str(base_path)}")
    storage_manager = storage.SharedFSStorageManager(str(base_path))
    return storage_manager


def _make_v2_context(
    *,
    distributed: Optional[core.DistributedContext] = None,
    checkpoint_storage: Optional[Union[str, Dict[str, Any]]] = None,
    preempt_mode: core.PreemptMode = core.PreemptMode.WorkersAskChief,
    tensorboard_mode: core.TensorboardMode = core.TensorboardMode.AUTO,
    unmanaged_info: Optional[det.ClusterInfo] = None,
    client: Optional[experimental.Determined] = None,
) -> core.Context:
    if unmanaged_info is None:
        unmanaged = False

        info = det.get_cluster_info()
        if info is None:
            raise ValueError(
                "Since the code is running outside of a determined-managed experiment, "
                "you must provide the `unmanaged_info` object."
            )

        # We are on the cluster.
        cert = certs.default_load(info.master_url)
        session = authentication.login_with_cache(info.master_url, cert=cert).with_retry(
            util.get_max_retries_config()
        )
    else:
        unmanaged = True

        info = unmanaged_info

        if client is None:
            session = experimental.client._get_singleton_session()
        else:
            session = client._session

    # TODO(ilia): we used to require explicit distributed context for distributed training jobs,
    # not anymore.
    distributed = distributed or core.DummyDistributedContext()

    # At present, we only support tensorboards in Trial tasks.
    tbd_writer = None

    metrics = None
    train = None
    searcher = None
    tensorboard_manager = None
    heartbeat = None
    log_shipper = None

    storage_manager = core._get_storage_manager(checkpoint_storage)

    if checkpoint_storage is None and not unmanaged:
        checkpoint_storage = info.trial._config.get("checkpoint_storage")
    # TODO(ilia): When checkpoint_storage is not specified, do this instead:
    # - if on-cluster: try using cluster storage, then appdirs
    # - off-cluster: appdirs.
    has_storage = checkpoint_storage is not None
    # No bind mounts for unmanaged tasks.
    container_path = constants.SHARED_FS_CONTAINER_PATH if not unmanaged else None
    if not has_storage:
        tensorboard_mode = core.TensorboardMode.MANUAL

    if info.task_type == "TRIAL":
        if has_storage:
            assert checkpoint_storage
            # Prepare the tensorboard hooks.
            tensorboard_manager = tensorboard.build(
                info.cluster_id,
                str(info.trial.experiment_id),
                str(info.trial.trial_id),
                checkpoint_storage,
                container_path=container_path,
                async_upload=True,
                sync_on_close=(tensorboard_mode == core.TensorboardMode.AUTO),
            )
            if tensorboard_mode == core.TensorboardMode.AUTO:
                tbd_writer = tensorboard.get_metric_writer()

        run_prepare_response = core._run_prepare(
            distributed,
            session,
            info.trial.trial_id,
            checkpoint_storage,
        )
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
            if has_storage:
                storage_manager = storage.build(
                    info.trial._config["checkpoint_storage"],
                    container_path=container_path,
                )
            else:
                storage_manager = _default_storage_manager()

        checkpoint = core.CheckpointContext(
            distributed,
            storage_manager,
            session,
            info.task_id,
            None,  # No allocations when off-cluster.
            tensorboard_mode,
            tensorboard_manager,
            run_prepare_response.storageId,
        )

        # At present, detached mode does not support preemption.
        preempt = core.DummyPreemptContext(distributed, preempt_mode)

        if unmanaged:
            log_shipper = core._UnmanagedTrialLogShipper(
                session=session,
                trial_id=info.trial.trial_id,
                task_id=info.task_id,
                distributed=distributed,
            )

            if distributed and distributed.rank == 0:
                heartbeat = core._UnmanagedTrialHeartbeat(
                    session=session,
                    trial_id=info.trial.trial_id,
                )
    else:
        if unmanaged:
            raise NotImplementedError("unmanaged mode is not supported for non-trial tasks")

        if storage_manager is None:
            storage_manager = _default_storage_manager()
        checkpoint = core.DummyCheckpointContext(distributed, storage_manager)
        preempt = core.DummyPreemptContext(distributed, preempt_mode)

    core._install_stacktrace_on_sigusr1()

    return core.Context(
        distributed=distributed,
        checkpoint=checkpoint,
        preempt=preempt,
        train=train,
        searcher=searcher,
        info=info,
        _metrics=metrics,
        _tensorboard_manager=tensorboard_manager,
        _heartbeat=heartbeat,
        _log_shipper=log_shipper,
        _session=session,
    )
