import re

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


@pytest.fixture
def sample_model_version(standard_session: api.Session) -> model.ModelVersion:
    bindings_model_versions = api_responses.sample_get_model_versions().modelVersions
    return model.ModelVersion._from_bindings(bindings_model_versions[0], standard_session)


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


@responses.activate
def test_set_name_doesnt_update_local_on_rest_failure(
    sample_model_version: model.ModelVersion,
) -> None:
    sample_model_version.name = "test_version_name"

    responses.patch(
        re.compile(f"{_MASTER}/api/v1/models/{sample_model_version.model_name}.*"), status=400
    )

    try:
        sample_model_version.set_name("new_version_name")
        raise AssertionError("Server's 400 should raise an exception")
    except api.errors.APIException:
        assert sample_model_version.name == "test_version_name"


@responses.activate
def test_set_notes_doesnt_update_local_on_rest_failure(
    sample_model_version: model.ModelVersion,
) -> None:
    sample_model_version.notes = "test notes"

    responses.patch(
        re.compile(f"{_MASTER}/api/v1/models/{sample_model_version.model_name}.*"), status=400
    )

    try:
        sample_model_version.set_notes("new notes")
        raise AssertionError("Server's 400 should raise an exception")
    except api.errors.APIException:
        assert sample_model_version.notes == "test notes"


@responses.activate
def test_add_metadata_doesnt_update_local_on_rest_failure(sample_model: model.Model) -> None:
    sample_model.metadata = {}

    responses.patch(re.compile(f"{_MASTER}/api/v1/models/{sample_model.name}.*"), status=400)

    try:
        sample_model.add_metadata({"test": "test"})
        raise AssertionError("Server's 400 should raise an exception")
    except api.errors.APIException:
        assert "test" not in sample_model.metadata


@responses.activate
def test_remove_metadata_doesnt_update_local_on_rest_failure(sample_model: model.Model) -> None:
    sample_model.metadata = {"test": "test"}

    responses.patch(re.compile(f"{_MASTER}/api/v1/models/{sample_model.name}.*"), status=400)

    try:
        sample_model.remove_metadata(["test"])
        raise AssertionError("Server's 400 should raise an exception")
    except api.errors.APIException:
        assert "test" in sample_model.metadata


@responses.activate
def test_archive_doesnt_update_local_on_rest_failure(sample_model: model.Model) -> None:
    sample_model.archived = False

    responses.post(re.compile(f"{_MASTER}/api/v1/models/{sample_model.name}.*"), status=400)

    try:
        sample_model.archive()
        raise AssertionError("Server's 400 should raise an exception")
    except api.errors.APIException:
        assert sample_model.archived is False


@responses.activate
def test_unarchive_doesnt_update_local_on_rest_failure(sample_model: model.Model) -> None:
    sample_model.archived = True

    responses.post(re.compile(f"{_MASTER}/api/v1/models/{sample_model.name}.*"), status=400)

    try:
        sample_model.unarchive()
        raise AssertionError("Server's 400 should raise an exception")
    except api.errors.APIException:
        assert sample_model.archived is True
