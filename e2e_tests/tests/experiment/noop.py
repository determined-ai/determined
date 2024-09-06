"""
A python API for creating tests that do no training, but only take the actions they are told to.
"""

import base64
import json
import logging
import pathlib
from typing import Any, Dict, Iterable, List, Optional, Sequence, Union

from determined.common import api
from determined.common.api import bindings
from determined.experimental import client
from tests import config as conf


class Exit:
    def __init__(self, code: int = 0) -> None:
        self.code = code

    def to_dict(self) -> Dict[str, Any]:
        return {"action": "exit", "code": self.code}


class Sleep:
    def __init__(self, time: float) -> None:
        self.time = time

    def to_dict(self) -> Dict[str, Any]:
        return {"action": "sleep", "time": self.time}


class Report:
    def __init__(self, metrics: Dict[str, Any], group: str = "training") -> None:
        self.metrics = metrics
        self.group = group

    def to_dict(self) -> Dict[str, Any]:
        return {"action": "report", "group": self.group, "metrics": self.metrics}


class Checkpoint:
    def to_dict(self) -> Dict[str, Any]:
        return {"action": "checkpoint"}


class Log:
    def __init__(self, msg: str, level: int = logging.INFO) -> None:
        # Transfer message as base64 to allow weird characters, and also so that when searching logs
        # you don't see the msg string anywhere in "metadata", like expconf.
        self.base64 = base64.b64encode(msg.encode("utf8")).decode("utf8")
        self.level = level

    def to_dict(self) -> Dict[str, Any]:
        return {"action": "log", "base64": self.base64, "level": self.level}


class CompleteSearcherOperation:
    def __init__(self, metric: float) -> None:
        self.metric = metric

    def to_dict(self) -> Dict[str, Any]:
        return {"action": "complete_searcher_operation", "metric": self.metric}


Action = Union[Exit, Sleep, Report, Checkpoint, Log, CompleteSearcherOperation]


def merge_config(old: Any, new: Any) -> Any:
    if isinstance(old, dict):
        out = dict(old)
        for key, new_value in new.items():
            if key in out:
                out[key] = merge_config(out[key], new_value)
            else:
                out[key] = new_value
        return out
    return new


def generate_config(
    actions: Sequence[Action],
    config: Optional[Dict[str, Any]] = None,
) -> Dict[str, Any]:
    base_config = {
        "name": "noop",
        "entrypoint": "python3 train.py",
        "hyperparameters": {
            "actions": {str(i): a.to_dict() for i, a in enumerate(actions, start=1)},
        },
        "max_restarts": 0,
        "searcher": {
            "name": "single",
            "metric": "x",
        },
    }

    if config is None:
        return base_config
    else:
        return merge_config(base_config, config)  # type: ignore


def create_experiment(
    sess: api.Session,
    actions: Sequence[Action] = (),
    config: Optional[Dict[str, Any]] = None,
    project_id: Optional[int] = None,
    includes: Optional[Iterable[Union[str, pathlib.Path]]] = None,
    template: Optional[str] = None,
) -> client.Experiment:
    config = generate_config(actions, config=config)
    model_dir = conf.fixtures_path("noop")
    return client.Determined._from_session(sess).create_experiment(
        config, model_dir, project_id=project_id, includes=includes, template=template
    )


def cli_config_overrides(actions: Sequence[Action]) -> List[str]:
    """
    Return a list of `--config=...` to override the actions hyperparameter for the CLI.

    Note that you must provide the full list of actions (which makes your test more readable anyway)
    and also note that you can't delete actions through this mechanism, because that isn't supported
    by our --config override mechanism.
    """

    return [
        f"--config=hyperparameters.actions.{action_id}={json.dumps(action.to_dict())}"
        for action_id, action in enumerate(actions, start=1)
    ]


def traininglike_steps(max_length: int, metric_scale: float = 1.0) -> List[Action]:
    metric = 1.0
    out: List[Action] = []
    # Make the noop experiment look like a train-validate-checkpoint training loop.
    for _ in range(max_length):
        out.append(Report({"x": metric}, group="training"))
        out.append(Report({"x": metric}, group="validation"))
        out.append(Checkpoint())
        metric *= metric_scale
    return out


# This is separate from create_experiment because the SDK doesn't currently support creating paused
# experiments, so we have to go the bindings layer.  It does not support the same configurability as
# noop.create_experiment() because we don't want to have to replicate the inside of
# Determined.create_experiment() in order to support pausing, and presently no caller of
# create_paused_experiment actually cares about ever activating the experiment.
def create_paused_experiment(
    sess: api.Session,
    project_id: Optional[int] = None,
    template: Optional[str] = None,
) -> client.Experiment:
    req = bindings.v1CreateExperimentRequest(
        activate=False,
        config=json.dumps(
            {
                "searcher": {
                    "name": "single",
                    "metric": "x",
                },
                "entrypoint": "echo yo",
            }
        ),
        projectId=project_id,
        template=template,
    )
    resp = bindings.post_CreateExperiment(sess, body=req)
    return client.Experiment._from_bindings(resp.experiment, sess)
