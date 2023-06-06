import json
import pathlib
from typing import Any

from determined.common.api import bindings

FIXTURES_DIR = pathlib.Path(__file__).resolve().parent


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


def sample_get_model() -> bindings.v1GetModelResponse:
    """Get a sample model from a fixture."""
    with open(FIXTURES_DIR / "model.json") as f:
        resp = bindings.v1GetModelResponse.from_json(json.load(f))
        return resp


def sample_get_model_versions() -> bindings.v1GetModelVersionsResponse:
    with open(FIXTURES_DIR / "model_versions.json") as f:
        resp = bindings.v1GetModelVersionsResponse.from_json(json.load(f))
        return resp


def sample_get_pagination() -> bindings.v1Pagination:
    with open(FIXTURES_DIR / "pagination.json") as f:
        resp = bindings.v1Pagination.from_json(json.load(f))
        return resp
