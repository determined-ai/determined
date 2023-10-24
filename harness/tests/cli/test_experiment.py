import dataclasses
import unittest.mock

import pytest
import requests_mock as mock

import determined
import determined.cli
from determined.common import constants
from determined.common.api import bindings, certs
from determined.common.api.authentication import Authentication
from tests.fixtures import api_responses


@dataclasses.dataclass
class CliArgs:
    master: str
    experiment_id: int
    user: str = "test"
    password: str = "test"
    polling_interval: float = 0.01  # Short polling interval so we can run tests quickly


def mock_det_auth(user: str = "test", master_url: str = "http://localhost:8888") -> Authentication:
    with mock.Mocker() as mocker:
        mocker.get(master_url + "/api/v1/me", status_code=200, json={"username": user})
        fake_user = {"username": user, "admin": True, "active": True}
        mocker.post(
            master_url + "/api/v1/auth/login",
            status_code=200,
            json={"token": "fake-token", "user": fake_user},
        )
        mocker.get("/info", status_code=200, json={"version": "1.0"})
        mocker.get(
            "/users/me", status_code=200, json={"username": constants.DEFAULT_DETERMINED_USER}
        )
        auth = Authentication(
            master_address=master_url,
            requested_user=user,
            password="password1",
            cert=certs.Cert(noverify=True),
        )
        return auth


@unittest.mock.patch("determined.common.api.authentication.Authentication")
def test_wait_returns_error_code_when_experiment_errors(
    auth_mock: unittest.mock.MagicMock,
    requests_mock: mock.Mocker,
) -> None:
    auth_mock.return_value = mock_det_auth()
    exp = api_responses.sample_get_experiment(id=1, state=bindings.experimentv1State.COMPLETED)
    args = CliArgs(master="http://localhost:8888", experiment_id=1)
    exp.experiment.state = bindings.experimentv1State.ERROR
    requests_mock.get(
        f"/api/v1/experiments/{args.experiment_id}",
        status_code=200,
        json=exp.to_json(),
    )
    with pytest.raises(SystemExit) as e:
        determined.cli.experiment.wait(args)
    assert e.value.code == 1
