import os
import tempfile
from typing import Optional
import uuid
from pathlib import Path

import pytest
import requests
import requests_mock as mock
import dataclasses
import determined
from determined import common

import determined.cli
import determined.cli.cli as cli
import determined.cli.command as command
from determined.common import constants, context
from determined.common.api import bindings
from tests.filetree import FileTree
from types import SimpleNamespace
import json


@dataclasses.dataclass
class CliArgs:
    master: str
    experiment_id: int
    user: str = "test"
    password: str = "test"
    polling_interval: int = 1


fixtures = Path(__file__).parent.parent / "fixtures"


@pytest.fixture
def experiment_json():
    with open(fixtures / "experiment.json") as f:
        return json.load(f)["experiment"]


def test_transient_network(requests_mock: mock.Mocker, experiment_json) -> None:
    requests_mock.get("/info", status_code=200, json={"version": "1.0"})
    requests_mock.get(
        "/users/me", status_code=200, json={"username": constants.DEFAULT_DETERMINED_USER}
    )
    requests_mock.post("/login", status_code=200, json={"token": "fake-token"})
    args = CliArgs(master="http://localhost:8080", experiment_id=1)
    exp = bindings.v1Experiment.from_json(experiment_json)
    exp.state = bindings.determinedexperimentv1State.STATE_COMPLETED

    def make_callback(fail_count: int):
        """
        Make a callback that will fail the first `fail_count` times it is called, and then succeed.
        """
        calls = 0

        def callback(request, context):
            nonlocal calls
            if calls < fail_count:
                calls += 1
                context.status_code = 504
                return ""
            else:
                context.status_code = 200
                return json.dumps({"experiment": exp.to_json()})

        return callback

    requests_mock.register_uri(
        "GET",
        f"http://localhost:8080/api/v1/experiments/{args.experiment_id}",
        text=make_callback(0),
    )
    with pytest.raises(SystemExit) as e:
        determined.cli.experiment.wait(args)
    assert e.value.code == 0

    requests_mock.register_uri(
        "GET",
        f"http://localhost:8080/api/v1/experiments/{args.experiment_id}",
        text=make_callback(5),
    )
    with pytest.raises(SystemExit) as e:
        determined.cli.experiment.wait(args)
    assert e.value.code == 0


def test_stable_network(requests_mock: mock.Mocker, experiment_json) -> None:
    requests_mock.get("/info", status_code=200, json={"version": "1.0"})
    requests_mock.get(
        "/users/me", status_code=200, json={"username": constants.DEFAULT_DETERMINED_USER}
    )
    requests_mock.post("/login", status_code=200, json={"token": "fake-token"})
    args = CliArgs(master="http://localhost:8080", experiment_id=1)
    exp = bindings.v1Experiment.from_json(experiment_json)

    exp.state = bindings.determinedexperimentv1State.STATE_COMPLETED
    requests_mock.get(
        f"http://localhost:8080/api/v1/experiments/{args.experiment_id}",
        status_code=200,
        json={"experiment": exp.to_json()},
    )
    with pytest.raises(SystemExit) as e:
        determined.cli.experiment.wait(args)
    assert e.value.code == 0

    exp.state = bindings.determinedexperimentv1State.STATE_ERROR
    requests_mock.get(
        f"http://localhost:8080/api/v1/experiments/{args.experiment_id}",
        status_code=200,
        json={"experiment": exp.to_json()},
    )
    with pytest.raises(SystemExit) as e:
        determined.cli.experiment.wait(args)
    assert e.value.code == 1
