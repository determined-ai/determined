from typing import Callable, List
from unittest import mock

import pytest
import responses

from determined.common.api import _util, authentication, errors
from determined.common.experimental import Determined
from tests.fixtures import api_responses

_MASTER = "http://localhost:8080"


@pytest.fixture
@mock.patch("determined.common.api.authentication.Authentication")
def mock_default_auth(auth_mock: mock.MagicMock) -> None:
    responses.get(f"{_MASTER}/api/v1/me", status=200, json={"username": "determined"})
    responses.post(
        f"{_MASTER}/api/v1/auth/login", status=200, json=api_responses.sample_login().to_json()
    )
    auth_mock.return_value = authentication.Authentication(
        master_address=_MASTER,
        requested_user="determined",
        password="password",
    )


@pytest.fixture
def make_client(mock_default_auth: Callable) -> Callable[[], Determined]:
    def _make_client() -> Determined:
        return Determined(master=_MASTER)

    return _make_client


@responses.activate
def test_default_retry_max_retries(make_client: Callable[[], Determined]) -> None:
    client = make_client()

    model_resp = api_responses.sample_get_model()
    get_model_url = f"{_MASTER}/api/v1/models/{model_resp.model.name}"

    # Mock retry requests within default max retries (5).
    responses.get(get_model_url, status=504)
    responses.get(get_model_url, status=504)
    responses.get(get_model_url, status=200, json=model_resp.to_json())

    client.get_model(model_resp.model.name)
    responses.assert_call_count(get_model_url, 3)
    responses.calls.reset()

    # Always return a 504 error.
    responses.get(get_model_url, status=504)

    with pytest.raises(errors.BadRequestException):
        client.get_model(model_resp.model.name)

    responses.assert_call_count(get_model_url, 6)


@responses.activate
def test_default_retry_allowed_methods(make_client: Callable[[], Determined]) -> None:
    client = make_client()
    # GET requests should retry.
    model_resp = api_responses.sample_get_model()
    get_model_url = f"{_MASTER}/api/v1/models/{model_resp.model.name}"
    responses.get(get_model_url, status=504)
    with pytest.raises(errors.BadRequestException):
        client.get_model(model_resp.model.name)
    responses.assert_call_count(get_model_url, 6)


@pytest.mark.parametrize(
    "status",
    _util.RETRY_STATUSES,
)
@responses.activate
def test_default_retry_status_forcelist(
    make_client: Callable[[], Determined],
    status: List[int],
) -> None:
    client = make_client()
    model_resp = api_responses.sample_get_model()
    get_model_url = f"{_MASTER}/api/v1/models/{model_resp.model.name}"

    responses.get(get_model_url, status=status)
    with pytest.raises(errors.BadRequestException):
        client.get_model(model_resp.model.name)
    responses.assert_call_count(get_model_url, 6)


@pytest.mark.parametrize(
    "status",
    [400, 404, 500],
)
@responses.activate
def test_default_retry_no_retry_statuses(
    make_client: Callable[[], Determined],
    status: List[int],
) -> None:
    client = make_client()
    model_resp = api_responses.sample_get_model()
    get_model_url = f"{_MASTER}/api/v1/models/{model_resp.model.name}"

    responses.get(get_model_url, status=status)
    with pytest.raises(errors.BadRequestException):
        client.get_model(model_resp.model.name)
    responses.assert_call_count(get_model_url, 1)
