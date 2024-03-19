import dataclasses
import unittest.mock

import pytest
import requests_mock as mock

import determined
import determined.cli
from determined.common import api
from determined.common.api import bindings
from tests.fixtures import api_responses


@dataclasses.dataclass
class CliArgs:
    master: str
    experiment_id: int
    user: str = "test"
    password: str = "test"
    polling_interval: float = 0.01  # Short polling interval so we can run tests quickly


@unittest.mock.patch("determined.common.api.authentication.login_with_cache")
def test_wait_returns_error_code_when_experiment_errors(
    login_with_cache_mock: unittest.mock.MagicMock,
    requests_mock: mock.Mocker,
) -> None:
    master = "http://localhost:8888"
    login_with_cache_mock.return_value = api.Session(master, "test", "test", None)
    exp = api_responses.sample_get_experiment(id=1, state=bindings.experimentv1State.COMPLETED)
    args = CliArgs(master=master, experiment_id=1)
    exp.experiment.state = bindings.experimentv1State.ERROR
    requests_mock.get(
        f"/api/v1/experiments/{args.experiment_id}",
        status_code=200,
        json=exp.to_json(),
    )
    with pytest.raises(SystemExit) as e:
        determined.cli.experiment.wait(args)  # type: ignore
    assert e.value.code == 1
