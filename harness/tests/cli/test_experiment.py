import dataclasses
import unittest.mock

import pytest
import requests_mock as mock

import determined
import determined.cli
from determined.common import constants
from determined.common.api import bindings, certs
from determined.common.api.authentication import Authentication
from tests.common import api_server


@dataclasses.dataclass
class CliArgs:
    master: str
    experiment_id: int
    user: str = "test"
    password: str = "test"
    polling_interval: int = 1


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
            try_reauth=True,
            cert=certs.Cert(noverify=True),
        )
        return auth


@unittest.mock.patch("determined.common.api.authentication.Authentication")
def test_wait_unstable_network(
    auth_mock: unittest.mock.MagicMock,
) -> None:
    auth_mock.return_value = mock_det_auth()
    user = "user1"
    with api_server.run_api_server(
        credentials=(user, "password1", "token1"), ssl_keys=None
    ) as master_url:
        args = CliArgs(master=master_url, experiment_id=1)
        determined.cli.experiment.wait(args)


@unittest.mock.patch("determined.common.api.authentication.Authentication")
def test_wait_stable_network(
    auth_mock: unittest.mock.MagicMock,
    requests_mock: mock.Mocker,
) -> None:
    auth_mock.return_value = mock_det_auth()

    exp = api_server.sample_get_experiment().experiment
    args = CliArgs(master="http://localhost:8888", experiment_id=1)

    exp.state = bindings.determinedexperimentv1State.STATE_COMPLETED
    requests_mock.get(
        f"/api/v1/experiments/{args.experiment_id}",
        status_code=200,
        json={"experiment": exp.to_json()},
    )
    determined.cli.experiment.wait(args)

    exp.state = bindings.determinedexperimentv1State.STATE_ERROR
    requests_mock.get(
        f"/api/v1/experiments/{args.experiment_id}",
        status_code=200,
        json={"experiment": exp.to_json()},
    )
    with pytest.raises(SystemExit) as e:
        determined.cli.experiment.wait(args)
    assert e.value.code == 1
