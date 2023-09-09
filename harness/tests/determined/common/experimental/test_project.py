import functools
import re
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

    responses.patch(re.compile(f"{_MASTER}/api/v1/projects/{sample_project.id}"), status=400)

    try:
        sample_project.set_name("new_project_name")
        raise AssertionError("Server's 400 should raise an exception")
    except api.errors.BadRequestException:
        assert sample_project.name == "test_project_name"


def test_to_json_encapsulates_hydrated_attributes(sample_project: project.Project) -> None:
    # We're going to patch some __dunder__ methods of Project in order to patch them in
    # sample_project. This is safe so long as the overridden block doesn't use any other Projects
    with mock.patch("determined.common.experimental.project.Project.__setattr__") as mock_setattr:
        mock_setattr.side_effect = functools.partial(object.__setattr__, sample_project)
        sample_project._hydrate(api_responses.sample_get_project().project)
        hydrated_attrs = {call[0][0] for call in mock_setattr.call_args_list}

    with mock.patch(
        "determined.common.experimental.project.Project.__getattribute__"
    ) as mock_getattr:
        mock_getattr.side_effect = functools.partial(object.__getattribute__, sample_project)
        sample_project.to_json()
        accessed_attrs_to_json = {call[0][0] for call in mock_getattr.call_args_list}

    assert hydrated_attrs.issubset(accessed_attrs_to_json)


@mock.patch("determined.common.api.bindings.get_GetProject")
def test_remove_note_raises_error_when_name_not_found(
    mock_get_project: mock.MagicMock, sample_project: project.Project
) -> None:
    bindings_project = api_responses.sample_get_project()
    bindings_project.project.notes = [bindings.v1Note(name="sample_name", contents="")]
    mock_get_project.return_value = bindings_project

    with pytest.raises(ValueError):
        sample_project.remove_note("nonexistent_note_name")


@mock.patch("determined.common.api.bindings.get_GetProject")
def test_remove_note_raises_exception_when_multiple_names_found(
    mock_get_project: mock.MagicMock, sample_project: project.Project
) -> None:
    bindings_project = api_responses.sample_get_project()
    bindings_project.project.notes = [
        bindings.v1Note(name="repeated_note_name", contents=""),
        bindings.v1Note(name="repeated_note_name", contents=""),
        bindings.v1Note(name="repeated_note_name", contents=""),
    ]
    mock_get_project.return_value = bindings_project

    with pytest.raises(NotImplementedError):
        sample_project.remove_note("repeated_note_name")
