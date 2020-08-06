import math
import random
import sys
import time
import uuid
from argparse import Namespace
from typing import Any, Dict, Optional

from ruamel import yaml
from termcolor import colored

from determined_common import api, constants, context
from determined_common.api import request as req
from determined_common.api.authentication import authentication_required


def patch_experiment(master_url: str, exp_id: int, patch_doc: Dict[str, Any]) -> None:
    path = "experiments/{}".format(exp_id)
    headers = {"Content-Type": "application/merge-patch+json"}
    req.patch(master_url, path, body=patch_doc, headers=headers)


def activate_experiment(master_url: str, exp_id: int) -> None:
    patch_experiment(master_url, exp_id, {"state": "ACTIVE"})


@authentication_required
def logs(args: Namespace) -> None:
    last_offset, last_state = 0, None

    def print_logs(offset: Optional[int], limit: Optional[int] = 5000) -> Any:
        nonlocal last_offset, last_state
        path = "trials/{}/logsv2?".format(args.trial_id)
        if offset is not None:
            path += "&offset={}".format(offset)
        if limit is not None:
            path += "&limit={}".format(limit)
        logs = api.get(args.master, path).json()
        for log in logs:
            print(log["message"], end="")
            last_state = log["state"]
        return logs[-1]["id"] if logs else last_offset

    try:
        if args.head is not None:
            print_logs(0, args.head)
            return
        if args.tail is not None:
            last_offset = print_logs(None, args.tail)
        else:
            while True:
                new_offset = print_logs(last_offset)
                if last_offset == new_offset:
                    break
                last_offset = new_offset

        if not args.follow:
            return
        while True:
            last_offset = print_logs(last_offset)
            if last_state in constants.TERMINAL_STATES:
                break
            time.sleep(0.2)
    except KeyboardInterrupt:
        pass
    finally:
        print(
            colored(
                "Trial is in the {} state. To reopen log stream, run: "
                "det trial logs -f {}".format(last_state, args.trial_id),
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

    # Call `logs --follow` on the new trial.
    logs_args = Namespace(
        trial_id=first_trial_id, master=master_url, head=None, tail=None, follow=True
    )
    logs(logs_args)


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
        r = api.get(master_url, "experiments/{}".format(exp_id)).json()

        # Wait for experiment to start and initialize a trial and step.
        if len(r["trials"]) < 1 or len(r["trials"][0]["steps"]) < 1:
            step = {}  # type: Dict
        else:
            step = r["trials"][0]["steps"][0]

        # Update the active_stage by examining the result from master
        # /experiments/<experiment-id> endpoint.
        if r["state"] == constants.COMPLETED:
            active_stage = 4
        elif step.get("checkpoint"):
            active_stage = 3
        elif step.get("validation"):
            active_stage = 2
        elif step:
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
            logs_args = Namespace(
                trial_id=trial_id, master=master_url, head=None, tail=None, follow=False
            )
            logs(logs_args)
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


def generate_random_hparam_values(hparam_def: Dict[str, Any]) -> Dict[str, Any]:
    def generate_random_value(hparam: Any) -> Any:
        if isinstance(hparam, Dict):
            if hparam["type"] == "const":
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
        elif isinstance(hparam, (int, float, str)):
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
