import logging
from typing import Optional, Tuple, Union

import determined as det
from determined import core, experimental
from determined.common import api, util
from determined.common.api import bindings

logger = logging.getLogger("determined.unmanaged")


def _create_unmanaged_experiment_inner(
    client: experimental.Determined,
    config_text: str,
) -> int:
    sess = client._session

    req1 = bindings.v1CreateExperimentRequest(config=config_text, unmanaged=True)
    resp1 = bindings.post_CreateExperiment(session=sess, body=req1)

    exp_id = resp1.experiment.id

    return exp_id


def _create_unmanaged_experiment(
    client: experimental.Determined,
    config_text: str,
    distributed: Optional[core.DistributedContext] = None,
) -> int:
    return core._run_on_rank_0_and_broadcast(
        lambda: _create_unmanaged_experiment_inner(client, config_text), distributed
    )


def _put_unmanaged_experiment_inner(
    client: experimental.Determined,
    config_text: str,
    external_experiment_id: str,
) -> int:
    sess = client._session

    req = bindings.v1CreateExperimentRequest(config=config_text, unmanaged=True)
    resp = bindings.put_PutExperiment(
        session=sess, body=req, externalExperimentId=external_experiment_id
    )
    exp_id = resp.experiment.id

    return exp_id


def _put_unmanaged_experiment(
    client: experimental.Determined,
    config_text: str,
    external_experiment_id: str,
    distributed: Optional[core.DistributedContext] = None,
) -> int:
    return core._run_on_rank_0_and_broadcast(
        lambda: _put_unmanaged_experiment_inner(client, config_text, external_experiment_id),
        distributed,
    )


# TODO(ilia): add a singleton helper to get the URL for the current experiment / trial.
def _url_reverse_webui_exp_view(client: experimental.Determined, exp_id: int) -> str:
    return f"{client._master}/det/experiments/{exp_id}"


def _get_cluster_id(sess: api.Session) -> str:
    resp = bindings.get_GetMaster(session=sess)
    return resp.clusterId


def _create_unmanaged_trial_inner(
    client: experimental.Determined,
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


def _create_unmanaged_trial(
    client: experimental.Determined,
    exp_id: int,
    hparams: Optional[dict] = None,
    distributed: Optional[core.DistributedContext] = None,
) -> Tuple[int, str]:
    return core._run_on_rank_0_and_broadcast(
        lambda: _create_unmanaged_trial_inner(client, exp_id=exp_id, hparams=hparams), distributed
    )


def _put_unmanaged_trial_inner(
    client: experimental.Determined,
    exp_id: int,
    external_trial_id: str,
    hparams: Optional[dict] = None,
) -> Tuple[int, str]:
    sess = client._session
    assert sess

    req_create = bindings.v1CreateTrialRequest(
        experimentId=exp_id,
        hparams=hparams,
        unmanaged=True,
    )
    req_put = bindings.v1PutTrialRequest(
        createTrialRequest=req_create,
        externalTrialId=external_trial_id,
    )
    resp = bindings.put_PutTrial(session=sess, body=req_put)

    trial_id = resp.trial.id
    task_id = resp.trial.taskId
    assert task_id
    return trial_id, task_id


def _put_unmanaged_trial(
    client: experimental.Determined,
    exp_id: int,
    external_trial_id: str,
    hparams: Optional[dict] = None,
    distributed: Optional[core.DistributedContext] = None,
) -> Tuple[int, str]:
    return core._run_on_rank_0_and_broadcast(
        lambda: _put_unmanaged_trial_inner(client, exp_id, external_trial_id, hparams), distributed
    )


def _start_trial_inner(
    client: experimental.Determined,
    trial_id: int,
    resume: bool,
) -> bindings.v1StartTrialResponse:
    sess = client._session
    assert sess

    req = bindings.v1StartTrialRequest(trialId=trial_id, resume=resume)
    resp = bindings.post_StartTrial(session=sess, body=req, trialId=trial_id)

    return resp


def _start_trial(
    client: experimental.Determined,
    trial_id: int,
    resume: bool,
    distributed: Optional[core.DistributedContext] = None,
) -> bindings.v1StartTrialResponse:
    return core._run_on_rank_0_and_broadcast(
        lambda: _start_trial_inner(client, trial_id, resume), distributed
    )


def _create_unmanaged_trial_cluster_info(
    client: experimental.Determined,
    config_text: str,
    exp_id: int,
    hparams: Optional[dict] = None,
    distributed: Optional[core.DistributedContext] = None,
) -> det.ClusterInfo:
    trial_id, task_id = _create_unmanaged_trial(
        client, exp_id=exp_id, hparams=hparams, distributed=distributed
    )

    return _build_unmanaged_trial_cluster_info(
        client,
        exp_id,
        trial_id,
        task_id,
        config_text,
        hparams,
        distributed=distributed,
    )


def _build_unmanaged_trial_cluster_info(
    client: experimental.Determined,
    exp_id: int,
    trial_id: int,
    task_id: str,
    config_text: str,  # TODO(ilia): we could load it from server.
    hparams: Optional[dict] = None,
    resume: bool = True,
    distributed: Optional[core.DistributedContext] = None,
) -> det.ClusterInfo:
    sess = client._session
    assert sess

    cluster_id = _get_cluster_id(sess)
    token = sess.token

    resp = _start_trial(client, trial_id, resume, distributed)

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
            config=util.yaml_safe_load(config_text),
            steps_completed=resp.stepsCompleted,
            trial_run_id=resp.trialRunId,
            debug=False,
            inter_node_network_interface=None,
        ),
        latest_checkpoint=resp.latestCheckpoint,
        rendezvous_info=det.RendezvousInfo(["127.0.0.1"], 0, [0]),
    )


def _create_unmanaged_cluster_info(
    client: experimental.Determined,
    config_text: str,
    hparams: Optional[dict] = None,
    distributed: Optional[core.DistributedContext] = None,
) -> det.ClusterInfo:
    exp_id = _create_unmanaged_experiment(client, config_text=config_text, distributed=distributed)
    return _create_unmanaged_trial_cluster_info(
        client, config_text=config_text, exp_id=exp_id, hparams=hparams, distributed=distributed
    )


def _get_or_create_experiment_and_trial(
    client: experimental.Determined,
    config_text: str,
    experiment_id: Optional[Union[str, int]] = None,
    trial_id: Optional[Union[str, int]] = None,
    hparams: Optional[dict] = None,
    distributed: Optional[core.DistributedContext] = None,
) -> det.ClusterInfo:
    if experiment_id is None:
        if trial_id is None:
            exp_id = _create_unmanaged_experiment(client, config_text, distributed)
            trial_id, task_id = _create_unmanaged_trial(client, exp_id, hparams, distributed)
            return _build_unmanaged_trial_cluster_info(
                client,
                exp_id,
                trial_id,
                task_id,
                config_text,
                hparams,
                distributed=distributed,
            )
        elif isinstance(trial_id, int):
            raise NotImplementedError
        elif isinstance(trial_id, str):
            # TODO(ilia): add GetExperimentByExternalTrialId.
            raise NotImplementedError
    elif isinstance(experiment_id, int):
        if trial_id is None:
            raise NotImplementedError
        elif isinstance(trial_id, int):
            raise NotImplementedError
        elif isinstance(trial_id, str):
            raise NotImplementedError
    elif isinstance(experiment_id, str):
        # TODO(ilia): Detect if >1 trials exist in the experiment.
        # If yes, and the experiment searcher config is `single`, patch the experiment
        # to switch the search config to `custom` searcher.
        # This should switch WebUI into hp search UI for these experiments.
        if trial_id is None:
            exp_id = _put_unmanaged_experiment(client, config_text, experiment_id, distributed)
            trial_id, task_id = _create_unmanaged_trial(client, exp_id, hparams, distributed)
            return _build_unmanaged_trial_cluster_info(
                client,
                exp_id,
                trial_id,
                task_id,
                config_text,
                hparams,
                distributed=distributed,
            )
        elif isinstance(trial_id, int):
            raise NotImplementedError
        elif isinstance(trial_id, str):
            exp_id = _put_unmanaged_experiment(client, config_text, experiment_id, distributed)
            trial_id, task_id = _put_unmanaged_trial(client, exp_id, trial_id, hparams, distributed)
            return _build_unmanaged_trial_cluster_info(
                client,
                exp_id,
                trial_id,
                task_id,
                config_text,
                hparams,
                distributed=distributed,
            )
    else:
        raise ValueError(f"experiment_id is neither int nor string: {type(experiment_id)}")
    raise ValueError(f"trial_id is neither int nor string: {type(trial_id)}")
