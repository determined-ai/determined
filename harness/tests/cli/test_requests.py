import dataclasses

import pytest
import requests_mock as mock

import determined
import determined.cli
from determined.common import constants
from determined.common.api import Session, bindings, certs
from determined.common.api.authentication import Authentication
from tests.common import api_server


@dataclasses.dataclass
class CliArgs:
    master: str
    experiment_id: int
    user: str = "test"
    password: str = "test"
    polling_interval: int = 1


# https://docs.pytest.org/en/latest/example/parametrize.html#apply-indirect-on-particular-arguments
def det_session(user: str = "test", master_url: str = "http://localhost:8888") -> Session:
    with mock.Mocker() as mocker:
        mocker.post(master_url + "/login", status_code=200, json={"token": "fake-token"})
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
        session = Session(master_url, user, cert=certs.Cert(noverify=True), auth=auth)
        assert session._user
        return session


def test_wait_transient_network() -> None:
    user = "user1"
    with api_server.run_api_server(
        credentials=(user, "password1", "token1"),
    ) as master_url:
        session = det_session(user=user, master_url=master_url)
        with pytest.raises(SystemExit) as e:
            determined.cli.experiment._wait(session, 1, 100)
        assert e.value.code == 0


def test_wait_stable_network(requests_mock: mock.Mocker) -> None:
    session = det_session()
    experiment_id = 1
    exp = api_server.sample_get_experiment().experiment

    exp.state = bindings.determinedexperimentv1State.STATE_COMPLETED
    requests_mock.get(
        f"/api/v1/experiments/{experiment_id}",
        status_code=200,
        json={"experiment": exp.to_json()},
    )
    with pytest.raises(SystemExit) as e:
        determined.cli.experiment._wait(session, experiment_id)
    assert e.value.code == 0

    exp.state = bindings.determinedexperimentv1State.STATE_ERROR
    requests_mock.get(
        f"/api/v1/experiments/{experiment_id}",
        status_code=200,
        json={"experiment": exp.to_json()},
    )
    with pytest.raises(SystemExit) as e:
        determined.cli.experiment._wait(session, experiment_id)
    assert e.value.code == 1
