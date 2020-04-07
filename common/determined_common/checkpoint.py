import pathlib
import shutil
from typing import Any, Dict, List, Optional, Tuple

from determined_common import api, check, constants, storage, util
from determined_common.api import authentication as auth
from determined_common.api import gql


def ckpt_path_for_step(step_id: int, trial_id: int) -> pathlib.Path:
    trial_path = "trial_{}".format(trial_id)
    step_path = "step_{}".format(step_id)
    return pathlib.Path(constants.DEFAULT_CHECKPOINT_PATH, trial_path, step_path)


def get_metric(step: gql.steps, name: str) -> Any:
    return step.validation.metrics["validation_metrics"][name]


def select_checkpoint(
    master: Optional[str],
    trial_id: int,
    latest: bool,
    best: bool,
    uuid: Optional[str],
    metric_name: Optional[str] = None,
    smaller_is_better: Optional[bool] = None,
) -> gql.checkpoints:
    check.eq(
        sum([int(latest), int(best), int(uuid is not None)]),
        1,
        "Exactly one of latest, best, or uuid must be set",
    )

    check.eq(
        metric_name is None,
        smaller_is_better is None,
        "metric_name and smaller_is_better must be set together",
    )

    if smaller_is_better is not None:
        check.true(best, "smaller_is_better and metric_name are only valid when best is also set")

    if not master:
        master = util.get_default_master_address()

    auth.initialize_session(master, None, try_reauth=True)

    q = api.GraphQLQuery(master)

    if metric_name is not None:
        checkpoint = q.op.best_checkpoint_by_metric(
            args={"tid": trial_id, "metric": metric_name, "smaller_is_better": smaller_is_better},
        )
    else:
        where = gql.checkpoints_bool_exp(
            state=gql.checkpoint_state_comparison_exp(_eq="COMPLETED"),
            trial_id=gql.Int_comparison_exp(_eq=trial_id),
        )

        order_by = []  # type: List[gql.checkpoints_order_by]
        if uuid is not None:
            where.uuid = gql.uuid_comparison_exp(_eq=uuid)
        elif latest:
            order_by = [gql.checkpoints_order_by(end_time=gql.order_by.desc)]
        elif best:
            where.validation = gql.validations_bool_exp(
                state=gql.validation_state_comparison_exp(_eq="COMPLETED")
            )
            order_by = [
                gql.checkpoints_order_by(
                    validation=gql.validations_order_by(
                        metric_values=gql.validation_metrics_order_by(signed=gql.order_by.asc)
                    )
                )
            ]

        checkpoint = q.op.checkpoints(where=where, order_by=order_by, limit=1)

    checkpoint.state()
    checkpoint.uuid()
    checkpoint.resources()

    validation = checkpoint.validation()
    validation.metrics()
    validation.state()

    step = checkpoint.step()
    step.end_time()
    step.id()
    step.start_time()
    step.trial.experiment.config(path="checkpoint_storage")

    resp = q.send()

    result = resp.best_checkpoint_by_metric if metric_name is not None else resp.checkpoints

    if not result:
        raise AssertionError("No checkpoint found for trial {}".format(trial_id))

    return result[0]  # type: ignore


def _download(
    checkpoint: gql.checkpoints, storage_config: Dict[str, Any], path: pathlib.Path
) -> None:
    manager = storage.build(storage_config)
    if not isinstance(manager, (storage.S3StorageManager, storage.GCSStorageManager)):
        raise AssertionError(
            "Downloading from S3 or GCS requires the experiment to be configured with "
            "S3 or GCS checkpointing, {} found instead".format(storage_config["type"])
        )

    metadata = storage.StorageMetadata.from_json(checkpoint.__to_json_value__())
    manager.download(metadata, str(path))


def _find_shared_fs_path(
    storage_config: Dict[str, Any], checkpoint: gql.checkpoints
) -> pathlib.Path:
    potential_paths = [
        [
            storage_config["container_path"],
            storage_config.get("storage_path", ""),
            checkpoint.uuid,
        ],
        [storage_config["host_path"], storage_config.get("storage_path", ""), checkpoint.uuid],
    ]

    for path in potential_paths:
        maybe_ckpt = pathlib.Path(*path)
        if maybe_ckpt.exists():
            return maybe_ckpt

    raise FileNotFoundError("Checkpoint {} not found".format(checkpoint.uuid))


def download(
    trial_id: int,
    latest: bool = False,
    best: bool = False,
    uuid: Optional[str] = None,
    output_dir: Optional[str] = None,
    master: Optional[str] = None,
    metric_name: Optional[str] = None,
    smaller_is_better: Optional[bool] = None,
) -> Tuple[str, gql.checkpoints]:
    checkpoint = select_checkpoint(
        master,
        trial_id,
        latest=latest,
        best=best,
        uuid=uuid,
        metric_name=metric_name,
        smaller_is_better=smaller_is_better,
    )
    storage_config = checkpoint.step.trial.experiment.config

    if output_dir is not None:
        local_ckpt_dir = pathlib.Path(output_dir)
    else:
        local_ckpt_dir = ckpt_path_for_step(checkpoint.step.id, trial_id)

    potential_metadata_paths = ["metadata.json", "MLmodel"]
    is_ckpt_cached = False
    for metadata_file in potential_metadata_paths:
        maybe_ckpt = local_ckpt_dir.joinpath(metadata_file)
        if maybe_ckpt.exists():
            is_ckpt_cached = True
            break

    if not is_ckpt_cached and storage_config["type"] == "shared_fs":
        src_ckpt_dir = _find_shared_fs_path(storage_config, checkpoint)
        shutil.copytree(str(src_ckpt_dir), str(local_ckpt_dir))

        return str(local_ckpt_dir), checkpoint

    if not is_ckpt_cached:
        local_ckpt_dir.mkdir(parents=True, exist_ok=True)
        _download(checkpoint, storage_config, local_ckpt_dir)

    return str(local_ckpt_dir), checkpoint
