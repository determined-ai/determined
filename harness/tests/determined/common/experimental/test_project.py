from unittest import mock

import pytest
import responses

from determined.common import api
from determined.common.api import bindings
from determined.common.experimental import project
from tests.fixtures import api_responses

_MASTER = "http://localhost:8080"


@pytest.fixture
def standard_session() -> api.Session:
    return api.Session(master=_MASTER, user=None, auth=None, cert=None)


@pytest.fixture
def sample_project(standard_session: api.Session) -> project.Project:
    bindings_project = api_responses.sample_get_project().project
    return project.Project._from_bindings(bindings_project, standard_session)


@responses.activate
def test_set_name_doesnt_update_local_on_rest_failure(
    sample_project: project.Project,
) -> None:
    sample_project.name = "test_project_name"

    responses.patch(f"{_MASTER}/api/v1/projects/{sample_project.id}", status=400)

    try:
        sample_project.set_name("new_project_name")
        raise AssertionError("Server's 400 should raise an exception")
    except api.errors.BadRequestException:
        assert sample_project.name == "test_project_name"


@mock.patch("determined.common.api.bindings.get_GetProject")
def test_remove_note_raises_exception_when_name_not_found(
    mock_get_project: mock.MagicMock, sample_project: project.Project
) -> None:
    bindings_project = api_responses.sample_get_project()
    bindings_project.project.notes = [bindings.v1Note(name="sample_name", contents="")]
    mock_get_project.return_value = bindings_project

    with pytest.raises(ValueError):
        sample_project.remove_note(name="nonexistent_note_name")


@mock.patch("determined.common.api.bindings.get_GetProject")
def test_remove_note_raises_exception_when_name_and_contents_not_found(
    mock_get_project: mock.MagicMock, sample_project: project.Project
) -> None:
    bindings_project = api_responses.sample_get_project()
    bindings_project.project.notes = [
        bindings.v1Note(name="sample_name", contents="sample_contents")
    ]
    mock_get_project.return_value = bindings_project

    with pytest.raises(ValueError):
        sample_project.remove_note(name="sample_name", contents="nonexistent_contents")


@mock.patch("determined.common.api.bindings.get_GetProject")
def test_remove_note_raises_exception_when_multiple_names_found_contents_absent(
    mock_get_project: mock.MagicMock, sample_project: project.Project
) -> None:
    bindings_project = api_responses.sample_get_project()
    bindings_project.project.notes = [
        bindings.v1Note(name="repeated_note_name", contents="1"),
        bindings.v1Note(name="repeated_note_name", contents="2"),
        bindings.v1Note(name="repeated_note_name", contents="3"),
    ]
    mock_get_project.return_value = bindings_project

    with pytest.raises(ValueError):
        sample_project.remove_note(name="repeated_note_name")


@mock.patch("determined.common.api.bindings.get_GetProject")
@responses.activate
def test_remove_note_handles_removal_when_multiple_names_found_contents_present(
    mock_get_project: mock.MagicMock, sample_project: project.Project
) -> None:
    bindings_project = api_responses.sample_get_project()
    bindings_project.project.notes = [
        bindings.v1Note(name="repeated_note_name", contents="1"),
        bindings.v1Note(name="repeated_note_name", contents="2"),
        bindings.v1Note(name="repeated_note_name", contents="3"),
    ]
    mock_get_project.return_value = bindings_project

    responses.put(f"{_MASTER}/api/v1/projects/{sample_project.id}/notes", json={"notes": []})
    sample_project.remove_note(name="repeated_note_name", contents="1")
