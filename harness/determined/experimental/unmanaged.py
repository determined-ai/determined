import io
import logging
from typing import Any, Dict, Optional, Tuple, Union

import determined as det
from determined import core, tensorboard
from determined.common import api, storage, yaml
from determined.common.api import bindings
from determined.core._context import Context, _get_storage_manager, _install_stacktrace_on_sigusr1
from determined.experimental import Determined

logger = logging.getLogger("determined.experimental.unmanaged")


# TODO: Missing unmanaged / detached mode features:
# - Add unmanaged experiment state management.
# - Make config.entrypoint optional.


def create_unmanaged_experiment(
    client: Determined,
    config_text: str,
    distributed: Optional[core.DistributedContext] = None,
) -> int:
    exp_id = None

    if distributed is None or distributed.rank == 0:
        sess = client._session

        req1 = bindings.v1CreateExperimentRequest(config=config_text, unmanaged=True)
        resp1 = bindings.post_CreateExperiment(session=sess, body=req1)

        exp_id = resp1.experiment.id
    if distributed is not None:
        exp_id = distributed.broadcast(exp_id)

    assert exp_id

    return exp_id


def url_reverse_webui_exp_view(client: Determined, exp_id: int) -> str:
    return api.request.make_url(client._master, f"/det/experiments/{exp_id}")


def _get_cluster_id(sess: api.Session) -> str:
    resp = bindings.get_GetMaster(session=sess)
    return resp.clusterId


def _create_unmanaged_trial(
    client: Determined,
    exp_id: int,
    hparams: Optional[dict] = None,
) -> Tuple[int, str]:
    sess = client._session
    assert sess

    req2 = bindings.v1CreateTrialRequest(experimentId=exp_id, hparams=hparams, unmanaged=True)
    resp2 = bindings.post_CreateTrial(session=sess, body=req2)

    trial_id = resp2.trial.id
    task_id = resp2.trial.taskId
    assert task_id
    return trial_id, task_id


def create_unmanaged_trial(
    client: Determined,
    exp_id: int,
    hparams: Optional[dict] = None,
    distributed: Optional[core.DistributedContext] = None,
) -> Tuple[int, str]:
    trial_id, task_id = None, None

    if distributed is None or distributed.rank == 0:
        trial_id, task_id = _create_unmanaged_trial(client, exp_id=exp_id, hparams=hparams)
    if distributed is not None:
        trial_id, task_id = distributed.broadcast([trial_id, task_id])

    assert trial_id
    assert task_id

    return trial_id, task_id


def create_unmanaged_trial_cluster_info(
    client: Determined,
    config_text: str,
    exp_id: int,
    hparams: Optional[dict] = None,
    distributed: Optional[core.DistributedContext] = None,
) -> det.ClusterInfo:
    trial_id, task_id = create_unmanaged_trial(
        client, exp_id=exp_id, hparams=hparams, distributed=distributed
    )

    return build_unmanaged_trial_cluster_info(
        client, exp_id, trial_id, task_id, config_text, hparams
    )


def build_unmanaged_trial_cluster_info(
    client: Determined,
    exp_id: int,
    trial_id: int,
    task_id: str,
    config_text: str,
    hparams: Optional[dict] = None,
) -> det.ClusterInfo:
    sess = client._session
    assert sess

    cluster_id = _get_cluster_id(sess)
    assert sess._auth
    token = sess._auth.get_session_token(True)

    return det.ClusterInfo(
        master_url=client._master,
        cluster_id=cluster_id,  # Required for tensorboard paths correctness.
        agent_id="unmanaged",  # TODO(ilia): when does this matter?
        slot_ids=[],
        task_id=task_id,
        allocation_id=task_id,  # TODO(ilia): when does this matter?
        session_token=token,
        task_type="TRIAL",
        trial_info=det.TrialInfo(
            trial_id=trial_id,
            experiment_id=exp_id,
            trial_seed=0,
            hparams=hparams or {},
            config=yaml.safe_load(io.StringIO(config_text)),
            steps_completed=0,
            trial_run_id=0,
            debug=False,
            inter_node_network_interface=None,
        ),
        rendezvous_info=det.RendezvousInfo(["127.0.0.1"], 0, [0]),
    )


def create_unmanaged_cluster_info(
    client: Determined,
    config_text: str,
    hparams: Optional[dict] = None,
    distributed: Optional[core.DistributedContext] = None,
) -> det.ClusterInfo:
    exp_id = create_unmanaged_experiment(client, config_text=config_text, distributed=distributed)
    return create_unmanaged_trial_cluster_info(
        client, config_text=config_text, exp_id=exp_id, hparams=hparams, distributed=distributed
    )


def init(
    *,
    distributed: Optional[core.DistributedContext] = None,
    checkpoint_storage: Optional[Union[str, Dict[str, Any]]] = None,
    preempt_mode: core.PreemptMode = core.PreemptMode.WorkersAskChief,
    tensorboard_mode: core.TensorboardMode = core.TensorboardMode.AUTO,
    unmanaged_info: Optional[det.ClusterInfo] = None,
    client: Optional[Determined] = None,
) -> Context:
    if unmanaged_info is None:
        raise ValueError(
            "for unmanaged mode context, you must provide the `unmanaged_info` object. "
            "Otherwise, use `det.core.init`."
        )

    if client is None:
        session = det.experimental.client._get_singleton_session()
    else:
        session = client._session

    # Reported, unmanaged, on- or off-cluster.
    info = unmanaged_info

    distributed = distributed or core.DummyDistributedContext()

    # At present, we only support tensorboards in Trial tasks.
    tbd_writer = None

    train = None
    searcher = None
    tensorboard_manager = None

    storage_manager = _get_storage_manager(checkpoint_storage)

    if info.task_type == "TRIAL":
        # Prepare the tensorboard hooks.
        tensorboard_manager = tensorboard.build(
            info.cluster_id,
            str(info.trial.experiment_id),
            str(info.trial.trial_id),
            info.trial._config["checkpoint_storage"],
            container_path=None,  # No bind mounts for unmanaged tasks.
            async_upload=True,
        )
        if tensorboard_mode == core.TensorboardMode.AUTO:
            tbd_writer = tensorboard.get_metric_writer()

        train = core.TrainContext(
            session,
            info.trial.trial_id,
            info.trial._trial_run_id,
            info.trial.experiment_id,
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
                container_path=None,  # No bind mounts for unmanaged tasks.
            )

        checkpoint = core.CheckpointContext(
            distributed,
            storage_manager,
            session,
            info.task_id,
            None,  # No allocations when off-cluster.
            tensorboard_mode,
            tensorboard_manager,
        )

        # At present, detached mode does not support preemption.
        preempt = core.DummyPreemptContext(distributed, preempt_mode)

    else:
        raise NotImplementedError("unmanaged mode is not supported for non-trial tasks")

    _install_stacktrace_on_sigusr1()

    return Context(
        distributed=distributed,
        checkpoint=checkpoint,
        preempt=preempt,
        train=train,
        searcher=searcher,
        _tensorboard_manager=tensorboard_manager,
    )
