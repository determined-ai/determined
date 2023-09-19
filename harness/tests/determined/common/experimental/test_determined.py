from typing import Callable, List
from unittest import mock

import pytest
import responses

from determined.common.api import authentication, errors
from determined.common.experimental import determined, experiment
from tests.fixtures import api_responses

_MASTER = "http://localhost:8080"

Determined = determined.Determined


@pytest.fixture
@mock.patch("determined.common.api.authentication.Authentication")
def mock_default_auth(auth_mock: mock.MagicMock) -> None:
    responses.get(f"{_MASTER}/api/v1/me", status=200, json={"username": api_responses.USERNAME})
    responses.post(
        f"{_MASTER}/api/v1/auth/login",
        status=200,
        json=api_responses.sample_login(username=api_responses.USERNAME).to_json(),
    )
    auth_mock.return_value = authentication.Authentication(
        master_address=_MASTER,
        requested_user=api_responses.USERNAME,
        password=api_responses.PASSWORD,
    )


@pytest.fixture
def make_client(mock_default_auth: Callable) -> Callable[[], Determined]:
    def _make_client() -> Determined:
        return Determined(master=_MASTER)

    return _make_client


@responses.activate
def test_default_retry_retries_transient_failures(make_client: Callable[[], Determined]) -> None:
    client = make_client()

    model_resp = api_responses.sample_get_model()
    get_model_url = f"{_MASTER}/api/v1/models/{model_resp.model.name}"

    # Mock retry requests within default max retries (5).
    responses.get(get_model_url, status=504)
    responses.get(get_model_url, status=504)
    responses.get(get_model_url, status=200, json=model_resp.to_json())

    client.get_model(model_resp.model.name)
    responses.assert_call_count(get_model_url, 3)


@responses.activate
def test_default_retry_retries_until_max(make_client: Callable[[], Determined]) -> None:
    client = make_client()
    model_resp = api_responses.sample_get_model()
    get_model_url = f"{_MASTER}/api/v1/models/{model_resp.model.name}"

    responses.get(get_model_url, status=504)
    with pytest.raises(errors.BadRequestException):
        client.get_model(model_resp.model.name)

    # Expect max retries (5) + 1 calls.
    responses.assert_call_count(get_model_url, 6)


@responses.activate
def test_default_retry_fails_after_max_retries(make_client: Callable[[], Determined]) -> None:
    client = make_client()
    model_resp = api_responses.sample_get_model()
    get_model_url = f"{_MASTER}/api/v1/models/{model_resp.model.name}"

    responses.get(get_model_url, status=504)
    with pytest.raises(errors.BadRequestException):
        client.get_model(model_resp.model.name)


@responses.activate
def test_default_retry_doesnt_retry_post(make_client: Callable[[], Determined]) -> None:
    client = make_client()
    model_resp = api_responses.sample_get_model()
    create_model_url = f"{_MASTER}/api/v1/models"
    responses.post(create_model_url, status=504)
    with pytest.raises(errors.BadRequestException):
        client.create_model(model_resp.model.name)
    responses.assert_call_count(create_model_url, 1)


@pytest.mark.parametrize(
    "status",
    [502, 503, 504],
)
@responses.activate
def test_default_retry_retries_status_forcelist(
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
def test_default_retry_doesnt_retry_allowed_status(
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


@pytest.mark.parametrize("attribute", ["summary_metrics", "state"])
@responses.activate
def test_get_trial_populates_attribute(
    make_client: Callable[[], Determined], attribute: str
) -> None:
    client = make_client()
    trial_id = 1
    tr_resp = api_responses.sample_get_trial(id=trial_id)
    trial_url = f"{_MASTER}/api/v1/trials/{trial_id}"
    responses.get(trial_url, json=tr_resp.to_json())

    resp = client.get_trial(trial_id=trial_id)

    assert attribute in resp.__dict__


@responses.activate
@mock.patch("determined.common.api.bindings.get_GetExperiments")
def test_list_experiments_calls_bindings_with_params(
    mock_bindings: mock.MagicMock,
    make_client: Callable[[], Determined],
) -> None:
    client = make_client()
    exps_resp = api_responses.sample_get_experiments()

    params = {
        "sort_by": experiment.ExperimentSortBy.ID,
        "order_by": experiment.ExperimentOrderBy.ASCENDING,
        "experiment_ids": list(range(10)),
        "labels": ["label1", "label2"],
        "users": ["user1", "user2"],
        "states": [experiment.ExperimentState.COMPLETED, experiment.ExperimentState.ACTIVE],
        "name": "exp name",
        "project_id": 1,
    }

    mock_bindings.side_effect = api_responses.iter_pages(
        pageable_resp=exps_resp,
        pageable_attribute="experiments",
    )

    list(client.list_experiments(**params))  # type: ignore

    _, call_kwargs = mock_bindings.call_args_list[0]

    assert call_kwargs["sortBy"] == params["sort_by"]._to_bindings()  # type: ignore
    assert call_kwargs["orderBy"] == params["order_by"]._to_bindings()  # type: ignore
    assert call_kwargs["experimentIdFilter_incl"] == params["experiment_ids"]
    assert call_kwargs["labels"] == params["labels"]
    assert call_kwargs["users"] == params["users"]
    assert call_kwargs["states"] == [
        state._to_bindings() for state in params["states"]  # type: ignore
    ]
    assert call_kwargs["name"] == params["name"]
    assert call_kwargs["projectId"] == params["project_id"]
