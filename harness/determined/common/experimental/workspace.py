from typing import List, Optional

from determined.common import api
from determined.common.api import bindings


class Workspace:
    """A class representing a Workspace object.

    Attributes:
        id: (int) The ID of the workspace.
        name: (Mutable, str) The name of the workspace.
    """

    def __init__(
        self,
        session: api.Session,
        workspace_id: Optional[int] = None,
        workspace_name: Optional[str] = None,
    ):
        """Create a Workspace object.

        Arguments:
            session: The session to use for API calls.
            workspace_id: (Optional) ID of the workspace. If None or not passed, the workspace ID
                will be looked up here at initialization.
            workspace_name: (Optional) Name of the workspace.

        Note: Only one of workspace_id or workspace_name should be passed.
        """
        if (workspace_id is None) == (workspace_name is None):
            raise ValueError("Workspace must be constructed with either a name or id (not both).")

        if workspace_id is None:
            assert workspace_name is not None
            self.id = _get_workspace_id_from_name(session, workspace_name)
        else:
            self.id = workspace_id
        self.name = workspace_name
        self._session = session

    @classmethod
    def _from_bindings(
        cls, workspace_bindings: bindings.v1Workspace, session: api.Session
    ) -> "Workspace":
        workspace = cls(session, workspace_id=workspace_bindings.id)
        workspace._hydrate(workspace_bindings)
        return workspace

    def _hydrate(self, workspace_bindings: bindings.v1Workspace) -> None:
        self.id = workspace_bindings.id
        self.name = workspace_bindings.name

    def reload(self) -> None:
        resp = bindings.get_GetWorkspace(session=self._session, id=self.id)
        workspace_bindings = resp.workspace

        self._hydrate(workspace_bindings)

    def list_pools(self) -> List[str]:
        """
        Lists the resources pools that the workspace has access to. Tasks submitted to this
        workspace can only use the resource pools listed here.
        """

        def get_with_offset(offset: int) -> bindings.v1ListRPsBoundToWorkspaceResponse:
            return bindings.get_ListRPsBoundToWorkspace(
                session=self._session,
                offset=offset,
                workspaceId=self.id,
            )

        resps = api.read_paginated(get_with_offset)
        resource_pools = [
            rp for r in resps if r.resourcePools is not None for rp in r.resourcePools
        ]

        return resource_pools


def _get_workspace_id_from_name(session: api.Session, name: str) -> int:
    """Workspace lookup from master that relies on a workspace name."""
    resp = bindings.get_GetWorkspaces(session=session, name=name)
    if len(resp.workspaces) == 0:
        raise ValueError(f"No workspace found with name {name}")
    assert len(resp.workspaces) < 2, f"Multiple workspaces found with name {name}"

    return resp.workspaces[0].id
