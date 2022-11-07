import os
import tempfile
import uuid
from pathlib import Path

import pytest
import requests
import requests_mock
import dataclasses
import determined

from determined.cli import experiment
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


fixtures = Path(__file__).parent.parent / "fixtures"


@pytest.fixture
def experiment_details():
    with open(fixtures / "experiment.json") as f:
        return json.load(f)


def test_transient_network(requests_mock: requests_mock.Mocker, experiment_details) -> None:
    requests_mock.get("/info", status_code=200, json={"version": "1.0"})
    requests_mock.get(
        "/users/me", status_code=200, json={"username": constants.DEFAULT_DETERMINED_USER}
    )
    requests_mock.post("/login", status_code=200, json={"token": "fake-token"})
    args = CliArgs(master="http://localhost:8080", experiment_id=1)
    requests_mock.get(
        f"http://localhost:8080/api/v1/experiments/{args.experiment_id}",
        status_code=200,
        json=experiment_details,
    )
    experiment.wait(args)
