import pytest
import responses

from determined.common import api
from determined.common.experimental import model
from tests.fixtures import api_responses

_MASTER = "http://localhost:8080"


@pytest.fixture
def standard_session() -> api.Session:
    return api.Session(master=_MASTER, user=None, auth=None, cert=None)


@pytest.fixture
def sample_model(standard_session: api.Session) -> model.Model:
    bindings_model = api_responses.sample_get_model().model
    return model.Model._from_bindings(bindings_model, standard_session)


@responses.activate
def test_get_versions_gets_all_pages(sample_model: model.Model) -> None:
    model_versions_resp = api_responses.sample_get_model_versions()
    model_versions_resp.model.name = sample_model.name = "test_model"

    responses.add_callback(
        responses.GET,
        f"{_MASTER}/api/v1/models/{sample_model.name}/versions",
        callback=api_responses.serve_by_page(model_versions_resp, "modelVersions", max_page_size=2),
    )

    mvs = sample_model.get_versions()
    assert len(mvs) == len(model_versions_resp.modelVersions)
