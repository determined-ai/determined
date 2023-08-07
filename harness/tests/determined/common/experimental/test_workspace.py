import pytest
import responses

from determined.common import api
from determined.common.api import bindings
from determined.common.experimental import workspace
from tests.fixtures import api_responses

_MASTER = "http://localhost:8080"


@pytest.fixture
def standard_session() -> api.Session:
    return api.Session(master=_MASTER, user=None, auth=None, cert=None)


@pytest.fixture
def single_item_workspaces() -> bindings.v1GetWorkspacesResponse:
    sample_workspace = api_responses.sample_get_workspace().workspace
    single_item_pagination = bindings.v1Pagination(endIndex=1, startIndex=0, total=1)
    return bindings.v1GetWorkspacesResponse(
        workspaces=[sample_workspace], pagination=single_item_pagination
    )


@pytest.fixture
def single_item_rps_bound_to_workspace() -> bindings.v1ListRPsBoundToWorkspaceResponse:
    """Create a dummy ListRPsBoundToWorkspaceResponse containing 1 resource pool."""
    single_item_pagination = bindings.v1Pagination(endIndex=1, startIndex=0, total=1)
    return bindings.v1ListRPsBoundToWorkspaceResponse(
        resourcePools=["foo"], pagination=single_item_pagination
    )


@pytest.fixture
def multi_item_rps_bound_to_workspace() -> bindings.v1ListRPsBoundToWorkspaceResponse:
    """Create a dummy ListRPsBoundToWorkspaceResponse containing 2 resource pools."""
    multi_items_pagination = bindings.v1Pagination(endIndex=2, startIndex=0, total=2)
    return bindings.v1ListRPsBoundToWorkspaceResponse(
        resourcePools=["foo", "bar"], pagination=multi_items_pagination
    )


@responses.activate
def test_workspace_constructor_requires_exactly_one_of_id_or_name(
    standard_session: api.Session,
    single_item_workspaces: bindings.v1GetWorkspacesResponse,
) -> None:
    responses.get(f"{_MASTER}/api/v1/workspaces", json=single_item_workspaces.to_json())

    with pytest.raises(ValueError):
        workspace.Workspace(session=standard_session, workspace_id=1, workspace_name="foo")

    with pytest.raises(ValueError):
        workspace.Workspace(session=standard_session)

    workspace.Workspace(session=standard_session, workspace_id=1)
    workspace.Workspace(session=standard_session, workspace_name="foo")


@responses.activate
def test_workspace_constructor_errors_when_no_workspace_found(
    standard_session: api.Session,
) -> None:
    resp = bindings.v1GetWorkspacesResponse(
        workspaces=[], pagination=api_responses.empty_get_pagination()
    )

    responses.get(f"{_MASTER}/api/v1/workspaces", json=resp.to_json())
    with pytest.raises(ValueError):
        workspace.Workspace(session=standard_session, workspace_name="not_found")


@responses.activate
def test_workspace_constructor_populates_id_from_name(
    standard_session: api.Session,
    single_item_workspaces: bindings.v1GetWorkspacesResponse,
) -> None:
    workspace_id = 1
    workspace_name = "foo"
    single_item_workspaces.workspaces[0].id = workspace_id
    single_item_workspaces.workspaces[0].name = workspace_name
    responses.get(f"{_MASTER}/api/v1/workspaces", json=single_item_workspaces.to_json())

    ws = workspace.Workspace(session=standard_session, workspace_name=workspace_name)
    assert ws.id == workspace_id


def test_workspace_constructor_doesnt_populate_name_from_id(standard_session: api.Session) -> None:
    ws = workspace.Workspace(session=standard_session, workspace_id=1)
    assert ws.name is None


@responses.activate
def test_workspace_get_available_resource_pools_reads_single_binding_no_pagination(
    standard_session: api.Session,
    single_item_workspaces: bindings.v1GetWorkspacesResponse,
    single_item_rps_bound_to_workspace: bindings.v1ListRPsBoundToWorkspaceResponse,
) -> None:
    workspace_id = single_item_workspaces.workspaces[0].id
    responses.get(f"{_MASTER}/api/v1/workspaces", json=single_item_workspaces.to_json())
    responses.get(
        f"{_MASTER}/api/v1/workspaces/{workspace_id}/available-resource-pools",
        json=single_item_rps_bound_to_workspace.to_json(),
    )

    ws = workspace.Workspace(session=standard_session, workspace_id=workspace_id)
    rps = ws.list_pools()
    assert rps == ["foo"]
