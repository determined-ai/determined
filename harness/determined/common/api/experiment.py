import collections
import math
import random
import sys
import time
import uuid
from typing import Any, Dict, List, Optional
from urllib.parse import urlencode

import simplejson
from termcolor import colored

from determined.common import api, constants, context, yaml
from determined.common.api import request as req


def patch_experiment_v1(master_url: str, exp_id: int, patch_doc: Dict[str, Any]) -> None:
    path = "/api/v1/experiments/{}".format(exp_id)
    headers = {"Content-Type": "application/merge-patch+json"}
    req.patch(master_url, path, json=patch_doc, headers=headers)


def patch_experiment(master_url: str, exp_id: int, patch_doc: Dict[str, Any]) -> None:
    path = "experiments/{}".format(exp_id)
    headers = {"Content-Type": "application/merge-patch+json"}
    req.patch(master_url, path, json=patch_doc, headers=headers)


def activate_experiment(master_url: str, exp_id: int) -> None:
    patch_experiment(master_url, exp_id, {"state": "ACTIVE"})


def trial_logs(
    master_url: str,
    trial_id: int,
    head: Optional[int] = None,
    tail: Optional[int] = None,
    follow: bool = False,
    agent_ids: Optional[List[str]] = None,
    container_ids: Optional[List[str]] = None,
    rank_ids: Optional[List[str]] = None,
    sources: Optional[List[str]] = None,
    stdtypes: Optional[List[str]] = None,
    level_above: Optional[str] = None,
    timestamp_before: Optional[str] = None,
    timestamp_after: Optional[str] = None,
) -> collections.abc.Iterable:
    def to_levels_above(level: str) -> List[str]:
        # We should just be using the generated client instead and this is why.
        levels = [
            "LOG_LEVEL_TRACE",
            "LOG_LEVEL_DEBUG",
            "LOG_LEVEL_INFO",
            "LOG_LEVEL_WARNING",
            "LOG_LEVEL_ERROR",
            "LOG_LEVEL_CRITICAL",
        ]
        try:
            return levels[levels.index("LOG_LEVEL_" + level) :]
        except ValueError:
            raise Exception("invalid log level: {}".format(level))

    reverse = False
    query = {}  # type: Dict[str, Any]
    if head is not None:
        query["limit"] = head
    elif tail is not None:
        query["limit"] = tail
        query["order_by"] = "ORDER_BY_DESC"
        reverse = True
    elif follow:
        query["follow"] = "true"

    for key, val in [
        ("agent_ids", agent_ids),
        ("container_ids", container_ids),
        ("rank_ids", rank_ids),
        ("sources", sources),
        ("stdtypes", stdtypes),
        ("timestamp_before", timestamp_before),
        ("timestamp_after", timestamp_after),
    ]:
        if val is not None:
            query[key] = val

    if level_above is not None:
        query["levels"] = to_levels_above(level_above)

    path = "/api/v1/trials/{}/logs?{}".format(trial_id, urlencode(query, doseq=True))
    with api.get(master_url, path, stream=True) as r:
        line_iter = r.iter_lines()
        if reverse:
            line_iter = reversed(list(line_iter))
        for line in line_iter:
            yield simplejson.loads(line)["result"]


def print_trial_logs(master_url: str, trial_id: int, **kwargs: Any) -> None:
    try:
        for log in trial_logs(master_url, trial_id, **kwargs):
            print(log["message"], end="")
    except KeyboardInterrupt:
        pass
    finally:
        print(
            colored(
                "Trial log stream ended. To reopen log stream, run: "
                "det trial logs -f {}".format(trial_id),
                "green",
            )
        )


def follow_experiment_logs(master_url: str, exp_id: int) -> None:
    # Get the ID of this experiment's first trial (i.e., the one with the lowest ID).
    print("Waiting for first trial to begin...")
    while True:
        r = api.get(master_url, "experiments/{}".format(exp_id))
        if len(r.json()["trials"]) > 0:
            break
        else:
            time.sleep(0.1)

    first_trial_id = sorted(t_id["id"] for t_id in r.json()["trials"])[0]
    print("Following first trial with ID {}".format(first_trial_id))
    print_trial_logs(master_url, first_trial_id, follow=True)


def follow_test_experiment_logs(master_url: str, exp_id: int) -> None:
    def print_progress(active_stage: int, ended: bool) -> None:
        # There are four sequential stages of verification. Track the
        # current stage with an index into this list.
        stages = [
            "Scheduling task",
            "Testing training",
            "Testing validation",
            "Testing checkpointing",
        ]

        for idx, stage in enumerate(stages):
            if active_stage > idx:
                color = "green"
                checkbox = "✔"
            elif active_stage == idx:
                color = "red" if ended else "yellow"
                checkbox = "✗" if ended else " "
            else:
                color = "white"
                checkbox = " "
            print(colored(stage + (25 - len(stage)) * ".", color), end="")
            print(colored(" [" + checkbox + "]", color), end="")

            if idx == len(stages) - 1:
                print("\n" if ended else "\r", end="")
            else:
                print(", ", end="")

    while True:
        r = api.get(master_url, f"experiments/{exp_id}").json()

        # Wait for experiment to start and initialize a trial.
        if len(r["trials"]) < 1:
            t = {}
        else:
            trial_id = r["trials"][0]["id"]
            t = api.get(master_url, f"trials/{trial_id}").json()

        # Update the active_stage by examining the result from master
        # /experiments/<experiment-id> endpoint.
        if r["state"] == constants.COMPLETED:
            active_stage = 4
        elif t.get("runner_state") == "checkpointing":
            active_stage = 3
        elif t.get("runner_state") == "validating":
            active_stage = 2
        elif t.get("runner_state") in ("UNSPECIFIED", "training"):
            active_stage = 1
        else:
            active_stage = 0

        # If the experiment is in a terminal state, output the appropriate
        # message and exit. Otherwise, sleep and repeat.
        if r["state"] == constants.COMPLETED:
            print_progress(active_stage, ended=True)
            print(colored("Model definition test succeeded! 🎉", "green"))
            return
        elif r["state"] == constants.CANCELED:
            print_progress(active_stage, ended=True)
            print(
                colored(
                    "Model definition test (ID: {}) canceled before "
                    "model test could complete. Please re-run the "
                    "command.".format(exp_id),
                    "yellow",
                )
            )
            sys.exit(1)
        elif r["state"] == constants.ERROR:
            print_progress(active_stage, ended=True)
            trial_id = r["trials"][0]["id"]
            print_trial_logs(master_url, trial_id)
            sys.exit(1)
        else:
            print_progress(active_stage, ended=False)
            time.sleep(0.2)


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

    r = req.post(master_url, "experiments", json=body)
    if not hasattr(r, "headers"):
        raise Exception(r)

    if validate_only:
        return 0

    new_resource = r.headers["Location"]
    experiment_id = int(new_resource.split("/")[-1])

    if activate:
        activate_experiment(master_url, experiment_id)

    return experiment_id


def generate_random_hparam_values(hparam_def: Dict[str, Any]) -> Dict[str, Any]:
    def generate_random_value(hparam: Any) -> Any:
        if isinstance(hparam, Dict):
            if "type" not in hparam:
                # In this case we have a dictionary of nested hyperparameters.
                return generate_random_hparam_values(hparam)
            elif hparam["type"] == "const":
                return hparam["val"]
            elif hparam["type"] == "int":
                return random.randint(hparam["minval"], hparam["maxval"])
            elif hparam["type"] == "double":
                return random.uniform(hparam["minval"], hparam["maxval"])
            elif hparam["type"] == "categorical":
                return hparam["vals"][random.randint(0, len(hparam["vals"]) - 1)]
            elif hparam["type"] == "log":
                return math.pow(hparam["base"], random.uniform(hparam["minval"], hparam["maxval"]))
            else:
                raise Exception("Wrong type of hyperparameter: {}".format(hparam["type"]))
        elif isinstance(hparam, (int, float, str, type(None))):
            return hparam
        else:
            raise Exception("Wrong type of hyperparameter: {}".format(type(hparam)))

    hparams = {name: generate_random_value(hparam_def[name]) for name in hparam_def}
    return hparams


def make_test_experiment_config(config: Dict[str, Any]) -> Dict[str, Any]:
    """
    Create a test experiment that based on a modified version of the
    experiment config of the request and monitors its progress for success.
    The test experiment is created as archived to be not user-visible by
    default.

    The experiment configuration is modified such that:
    1. Only train one batch.
    2. Only use one slot.
    3. The experiment does not attempt restarts on failure.
    4. All checkpoints are GC'd after experiment finishes.
    """
    config_test = config.copy()
    config_test.update(
        {
            "description": "[test-mode] {}".format(
                config_test.get("description", str(uuid.uuid4()))
            ),
            "scheduling_unit": 1,
            "min_validation_period": {"batches": 1},
            "checkpoint_storage": {
                **config_test.get("checkpoint_storage", {}),
                "save_experiment_best": 0,
                "save_trial_best": 0,
                "save_trial_latest": 0,
            },
            "searcher": {
                "name": "single",
                "metric": config_test["searcher"]["metric"],
                "max_length": {"batches": 1},
            },
            "hyperparameters": generate_random_hparam_values(config.get("hyperparameters", {})),
            "resources": {**config_test.get("resources", {"slots_per_trial": 1})},
            "max_restarts": 0,
        }
    )
    config.setdefault(
        "data_layer", {"type": "shared_fs", "container_storage_path": "/tmp/determined"}
    )

    return config_test


def create_experiment_and_follow_logs(
    master_url: str,
    config: Dict[str, Any],
    model_context: context.Context,
    template: Optional[str] = None,
    additional_body_fields: Optional[Dict[str, Any]] = None,
    activate: bool = True,
    follow_first_trial_logs: bool = True,
) -> int:
    exp_id = api.experiment.create_experiment(
        master_url,
        config,
        model_context,
        template=template,
        additional_body_fields=additional_body_fields,
        activate=activate,
    )
    print("Created experiment {}".format(exp_id))
    if activate and follow_first_trial_logs:
        api.follow_experiment_logs(master_url, exp_id)
    return exp_id


def create_test_experiment_and_follow_logs(
    master_url: str,
    config: Dict[str, Any],
    model_context: context.Context,
    template: Optional[str] = None,
    additional_body_fields: Optional[Dict[str, Any]] = None,
) -> int:
    print(colored("Validating experiment configuration...", "yellow"), end="\r")
    api.experiment.create_experiment(
        master_url,
        config,
        model_context,
        template=template,
        validate_only=True,
        additional_body_fields=additional_body_fields,
    )
    print(colored("Experiment configuration validation succeeded! 🎉", "green"))

    print(colored("Creating test experiment...", "yellow"), end="\r")
    exp_id = api.experiment.create_experiment(
        master_url,
        make_test_experiment_config(config),
        model_context,
        template=template,
        additional_body_fields=additional_body_fields,
        archived=True,
        activate=True,
    )
    print(colored("Created test experiment {}".format(exp_id), "green"))
    api.experiment.follow_test_experiment_logs(master_url, exp_id)
    return exp_id
