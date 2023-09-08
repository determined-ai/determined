import math
from typing import List
from unittest import mock

import pytest
import responses

from determined.common.api import authentication, errors
from determined.experimental import client as _client
from tests.fixtures import api_responses

_MASTER = "http://localhost:8080"


def make_client() -> _client.Determined:
    with mock.patch("determined.common.api.authentication.login_with_cache") as mock_login:
        mock_login.return_value = authentication.UsernameTokenPair("username", "token")
        return _client.Determined(_MASTER)


@responses.activate
def test_default_retry_retries_transient_failures() -> None:
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
def test_default_retry_retries_until_max() -> None:
    client = make_client()
    model_resp = api_responses.sample_get_model()
    get_model_url = f"{_MASTER}/api/v1/models/{model_resp.model.name}"

    responses.get(get_model_url, status=504)
    with pytest.raises(errors.BadRequestException):
        client.get_model(model_resp.model.name)

    # Expect max retries (5) + 1 calls.
    responses.assert_call_count(get_model_url, 6)


@responses.activate
def test_default_retry_fails_after_max_retries() -> None:
    client = make_client()
    model_resp = api_responses.sample_get_model()
    get_model_url = f"{_MASTER}/api/v1/models/{model_resp.model.name}"

    responses.get(get_model_url, status=504)
    with pytest.raises(errors.BadRequestException):
        client.get_model(model_resp.model.name)


@responses.activate
def test_default_retry_doesnt_retry_post() -> None:
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
def test_default_retry_retries_status_forcelist(status: List[int]) -> None:
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
def test_default_retry_doesnt_retry_allowed_status(status: List[int]) -> None:
    client = make_client()
    model_resp = api_responses.sample_get_model()
    get_model_url = f"{_MASTER}/api/v1/models/{model_resp.model.name}"

    responses.get(get_model_url, status=status)
    with pytest.raises(errors.BadRequestException):
        client.get_model(model_resp.model.name)
    responses.assert_call_count(get_model_url, 1)


@pytest.mark.parametrize("attribute", ["summary_metrics", "state"])
@responses.activate
def test_get_trial_populates_attribute(attribute: str) -> None:
    client = make_client()
    trial_id = 1
    tr_resp = api_responses.sample_get_trial(id=trial_id)
    trial_url = f"{_MASTER}/api/v1/trials/{trial_id}"
    responses.get(trial_url, json=tr_resp.to_json())

    resp = client.get_trial(trial_id=trial_id)

    assert attribute in resp.__dict__


@responses.activate
@mock.patch("determined.common.api.bindings.get_GetExperiments")
def test_list_experiments_calls_bindings_with_params(mock_bindings: mock.MagicMock) -> None:
    client = make_client()
    exps_resp = api_responses.sample_get_experiments()

    params = {
        "sort_by": _client.ExperimentSortBy.ID,
        "order_by": _client.OrderBy.ASCENDING,
        "experiment_ids": list(range(10)),
        "labels": ["label1", "label2"],
        "users": ["user1", "user2"],
        "states": [_client.ExperimentState.COMPLETED, _client.ExperimentState.ACTIVE],
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


@responses.activate
@mock.patch("determined.common.api.bindings.get_GetExperiments")
def test_list_experiments_returns_all_response_pages(mock_bindings: mock.MagicMock) -> None:
    client = make_client()
    exps_resp = api_responses.sample_get_experiments()
    total_exps = len(exps_resp.experiments)
    page_size = 2
    total_pages = math.ceil(total_exps / page_size)
    if total_pages == 1:
        raise ValueError(f"Test expects response to contain > {page_size} objects.")
    mock_bindings.side_effect = api_responses.iter_pages(
        pageable_resp=exps_resp,
        pageable_attribute="experiments",
        max_page_size=page_size,
    )

    exps = client.list_experiments()

    exp_ids = [exp.id for exp in exps]
    expected_exp_ids = [exp_b.id for exp_b in exps_resp.experiments]
    assert exp_ids == expected_exp_ids
