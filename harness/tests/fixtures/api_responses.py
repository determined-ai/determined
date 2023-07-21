import copy
import json
import pathlib
from typing import Any, Callable, Dict, Iterable, Iterator, Optional, Tuple, TypeVar

import requests

from determined.common.api import bindings

FIXTURES_DIR = pathlib.Path(__file__).resolve().parent

# Generic for all types that can be paginated
P = TypeVar("P", bound=bindings.Paginated)

# Default constants.
USERNAME = "determined"
PASSWORD = "password"


def sample_get_experiment(**kwargs: Any) -> bindings.v1GetExperimentResponse:
    """Get an experiment from a fixture and optionally override some fields.

    Load a sample experiment from a fixture.  It's assumed that generally a caller cares only that
    the response is well-formed. If instead the caller cares about any particular fields, they can
    override them by passing them as keyword arguments.

    Args:
        **kwargs: Fields to override in the experiment.

    Returns:
        A bindings.v1GetExperimentResponse object with the experiment. NOTE: The returned object
        is a bindings type, *not* a ExperimentReference.
    """
    with open(FIXTURES_DIR / "experiment.json") as f:
        resp = bindings.v1GetExperimentResponse.from_json(json.load(f))
        for k, v in kwargs.items():
            setattr(resp.experiment, k, v)
        return resp


def sample_get_experiments() -> bindings.v1GetExperimentsResponse:
    with open(FIXTURES_DIR / "experiments.json") as f:
        resp = bindings.v1GetExperimentsResponse.from_json(json.load(f))
        return resp


def sample_get_experiment_trials() -> bindings.v1GetExperimentTrialsResponse:
    with open(FIXTURES_DIR / "experiment_trials.json") as f:
        resp = bindings.v1GetExperimentTrialsResponse.from_json(json.load(f))
        return resp


def sample_get_trial(**kwargs: Any) -> bindings.v1GetTrialResponse:
    with open(FIXTURES_DIR / "trial.json") as f:
        resp = bindings.v1GetTrialResponse.from_json(json.load(f))
        for k, v in kwargs.items():
            setattr(resp.trial, k, v)
        return resp


def sample_trial_logs() -> str:
    with open(FIXTURES_DIR / "trial_logs.json") as f:
        return f.read()


def sample_get_model() -> bindings.v1GetModelResponse:
    """Get a sample model from a fixture."""
    with open(FIXTURES_DIR / "model.json") as f:
        resp = bindings.v1GetModelResponse.from_json(json.load(f))
        return resp


def sample_get_model_versions() -> bindings.v1GetModelVersionsResponse:
    with open(FIXTURES_DIR / "model_versions.json") as f:
        resp = bindings.v1GetModelVersionsResponse.from_json(json.load(f))
        return resp


def sample_login(username: str = USERNAME) -> bindings.v1LoginResponse:
    resp = bindings.v1LoginResponse(
        token="fake-login-token", user=sample_get_user(username=username).user
    )
    return resp


def sample_get_user(username: str = USERNAME) -> bindings.v1GetUserResponse:
    user = bindings.v1User(
        active=True,
        admin=False,
        username=username,
    )

    return bindings.v1GetUserResponse(user=user)


def sample_get_checkpoint() -> bindings.v1GetCheckpointResponse:
    with open(FIXTURES_DIR / "checkpoint.json") as f:
        resp = bindings.v1GetCheckpointResponse.from_json(json.load(f))
        return resp


def sample_get_workspace() -> bindings.v1GetWorkspaceResponse:
    with open(FIXTURES_DIR / "workspace.json") as f:
        resp = bindings.v1GetWorkspaceResponse.from_json(json.load(f))
        return resp


def sample_get_pagination() -> bindings.v1Pagination:
    with open(FIXTURES_DIR / "pagination.json") as f:
        resp = bindings.v1Pagination.from_json(json.load(f))
        return resp


def empty_get_pagination() -> bindings.v1Pagination:
    """A pagination response for an object with no entries."""
    return bindings.v1Pagination(endIndex=0, limit=0, offset=0, startIndex=0, total=0)


def page_of(
    complete_resp: P, pageable_type: str, offset: int = 0, limit: Optional[int] = None
) -> P:
    """Return a paginated response from a complete response.

    This assumes that the passed `complete_resp` contains an attribute named `pageable_type` that
    can be broken up into pages.

    Args:
        complete_resp: A complete response that can be paginated
        pageable_type: The name of the attribute of the complete_resp that can be paginated
        offset: If positive, the number of attributes to start the page from. If negative, indexed
          from the end.
        limit: The maximum number of attributes to include in the page. If None, include all
          attributes.

    Returns:
        A copy of the complete_resp wherein:
            - the pageable_type attribute has been sliced into a single page
            - a new pagination attribute has been created from passed offset and limit
    """
    if not hasattr(complete_resp, pageable_type):
        raise ValueError(f"Response does not have a {pageable_type} attribute")
    if not isinstance(getattr(complete_resp, pageable_type), Iterable):
        raise ValueError(f"Attribute {pageable_type} is not pageable")

    paged_resp = copy.deepcopy(complete_resp)
    total = len(getattr(complete_resp, pageable_type))
    start_index = offset if offset >= 0 else total + offset  # Negative offset means from the end
    end_index = total if limit is None else min(start_index + limit, total)

    paged_resp.pagination = bindings.v1Pagination(
        endIndex=end_index, limit=limit, offset=offset, startIndex=start_index, total=total
    )

    page = getattr(paged_resp, pageable_type)[start_index:end_index]
    setattr(paged_resp, pageable_type, page)

    return paged_resp


def serve_by_page(
    pageable_resp: bindings.Paginated, pageable_type: str, max_page_size: int
) -> Callable[[requests.PreparedRequest], Tuple[int, Dict, str]]:
    """Create a callback for responses that serves a paginated response.

    Pages will be created from a complete response based on the request's param's offset and limit.

    Args:
        pageable_resp: A complete response that can be paginated
        pageable_type: The name of the field in the response that will be split up across pages when
          a response is paginated
        max_page_size: The maximum number of items to include in each page. If a request's params
          specify a limit that is larger than this (or no limit at all), the limit will be reduced
          to this value

    Returns:
        A function that returns a page of a response based on a request's params
        (and this function's args)

        The precise return value (required by responses) is a tuple of (status_code, headers, body).
    """

    def _serve_by_page(request: requests.PreparedRequest) -> Tuple[int, Dict, str]:
        # ignore type checking on request.params -- responses guarantees params is populated
        limit = min(int(request.params.get("limit", max_page_size)), max_page_size)  # type: ignore
        paged_response = page_of(
            pageable_resp,
            pageable_type,
            offset=int(request.params.get("offset", 0)),  # type: ignore
            limit=limit,
        )
        return (200, {}, json.dumps(paged_response.to_json()))

    return _serve_by_page


def iter_pages(
    pageable_resp: bindings.Paginated, pageable_type: str, max_page_size: int = None
) -> Iterator[P]:
    """Creates an infinite generator from a pageable response.

    If the pageable response is exhausted, this method will return an empty response.

    Args:
        pageable_resp: A complete response that can be paginated
        pageable_type: The name of the field in the response that will be split up across pages when
          a response is paginated
        max_page_size: The maximum number of items to include in each page. If a request's params
          specify a limit that is larger than this (or no limit at all), the limit will be reduced
          to this value
    """
    offset = 0
    while True:
        page = page_of(
            complete_resp=pageable_resp,
            pageable_type=pageable_type,
            offset=offset,
            limit=max_page_size,
        )
        yield page
        offset = page.pagination.endIndex
