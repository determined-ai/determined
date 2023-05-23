import dataclasses
import unittest.mock

import pytest
import requests_mock as mock

import determined
import determined.cli
from determined.common import api, constants
from determined.common.api import bindings, certs
from determined.common.api.authentication import Authentication
from determined.experimental import client
from tests.common import api_server


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
def test_wait_doesnt_throw_exception_on_master_504(
    auth_mock: unittest.mock.MagicMock,
) -> None:
    auth_mock.return_value = mock_det_auth()
    user = "user1"
    experiment_id_flaky = 1
    with api_server.run_api_server(
        credentials=(user, "password1", "token1"), ssl_keys=None
    ) as master_url:
        args = CliArgs(master=master_url, experiment_id=experiment_id_flaky)
        determined.cli.experiment.wait(args)


@unittest.mock.patch("determined.common.api.authentication.Authentication")
def test_wait_waits_until_longrunning_experiment_is_complete(
    auth_mock: unittest.mock.MagicMock,
) -> None:
    auth_mock.return_value = mock_det_auth()
    user, password, token = "user", "password1", "token1"
    api_server_session = api.Session(
        master=f"{api_server.DEFAULT_HOST}:{api_server.DEFAULT_PORT}",
        user=user,
        auth=None,
        cert=None,
    )
    experiment_id_longrunning = 2

    with api_server.run_api_server(
        credentials=(user, password, token), ssl_keys=None
    ) as master_url:
        args = CliArgs(master=master_url, experiment_id=experiment_id_longrunning)
        determined.cli.experiment.wait(args)

        fetched_experiment = client.Experiment(experiment_id_longrunning, api_server_session)._get()

    assert fetched_experiment.state == bindings.experimentv1State.COMPLETED


@unittest.mock.patch("determined.common.api.authentication.Authentication")
def test_wait_returns_error_code_when_experiment_errors(
    auth_mock: unittest.mock.MagicMock,
    requests_mock: mock.Mocker,
) -> None:
    auth_mock.return_value = mock_det_auth()

    exp = api_server.sample_get_experiment().experiment
    args = CliArgs(master="http://localhost:8888", experiment_id=1)

    exp.state = bindings.experimentv1State.ERROR
    requests_mock.get(
        f"/api/v1/experiments/{args.experiment_id}",
        status_code=200,
        json={"experiment": exp.to_json()},
    )
    with pytest.raises(SystemExit) as e:
        determined.cli.experiment.wait(args)
    assert e.value.code == 1
