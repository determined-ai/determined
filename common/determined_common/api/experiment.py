import uuid
from typing import Any, Dict, Optional

from ruamel import yaml

from determined_common import context
from determined_common.api import request as req


def patch_experiment(master_url: str, exp_id: int, patch_doc: Dict[str, Any]) -> None:
    path = "experiments/{}".format(exp_id)
    headers = {"Content-Type": "application/merge-patch+json"}
    req.patch(master_url, path, body=patch_doc, headers=headers)


def activate_experiment(master_url: str, exp_id: int) -> None:
    patch_experiment(master_url, exp_id, {"state": "ACTIVE"})


def create_experiment(
    master_url: str,
    config: Dict[str, Any],
    model_context: context.Context,
    template: Optional[str] = None,
    validate_only: bool = False,
    archived: bool = False,
    activate: bool = True,
    additional_body_fields: Optional[Dict[str, Any]] = None,
) -> int:
    body = {
        "experiment_config": yaml.safe_dump(config),
        "model_definition": [e.dict() for e in model_context.entries],
        "validate_only": validate_only,
    }
    if template:
        body["template"] = template
    if archived:
        body["archived"] = archived
    if additional_body_fields:
        body.update(additional_body_fields)

    r = req.post(master_url, "experiments", body=body)
    if not hasattr(r, "headers"):
        raise Exception(r)

    if validate_only:
        return 0

    new_resource = r.headers["Location"]
    experiment_id = int(new_resource.split("/")[-1])

    if activate:
        activate_experiment(master_url, experiment_id)

    return experiment_id


def make_test_experiment_config(config: Dict[str, Any]) -> Dict[str, Any]:
    """
    Create a short experiment that based on a modified version of the
    experiment config of the request and monitors its progress for success.
    The short experiment is created as archived to be not user-visible by
    default.

    The experiment configuration is modified such that:
    1. The training step takes a minimum amount of time.
    2. All checkpoints are GC'd after experiment finishes.
    3. The experiment does not attempt restarts on failure.
    """
    config_test = config.copy()
    config_test.update(
        {
            "description": "[test-mode] {}".format(
                config_test.get("description", str(uuid.uuid4()))
            ),
            "batches_per_step": 1,
            "min_validation_period": 1,
            "checkpoint_storage": {
                **config_test.get("checkpoint_storage", {}),
                "save_experiment_best": 0,
                "save_trial_best": 0,
                "save_trial_latest": 0,
            },
            "searcher": {
                "name": "single",
                "metric": config_test["searcher"]["metric"],
                "max_steps": 1,
            },
            "resources": {**config_test.get("resources", {"slots_per_trial": 1})},
            "max_restarts": 0,
        }
    )
    return config_test


def create_test_experiment(
    master_url: str,
    config: Dict[str, Any],
    model_context: context.Context,
    template: Optional[str] = None,
    additional_body_fields: Optional[Dict[str, Any]] = None,
) -> int:
    return create_experiment(
        master_url=master_url,
        config=make_test_experiment_config(config),
        model_context=model_context,
        template=template,
        archived=True,
        activate=True,
        additional_body_fields=additional_body_fields,
    )
