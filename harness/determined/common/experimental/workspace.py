from typing import Optional

from determined.common import api
from determined.common.api import bindings


class Workspace:
    """A class representing a Workspace object.

    Attributes:
        id: The ID of the workspace. Though a Workspace object can be created without an ID (and
            therefore the attribute is mutable), workspace IDs are immutable for the lifetime of
            the workspace.
        name: (Mutable, str) The name of the workspace.
    """

    def __init__(
        self,
        session: api.Session,
        workspace_id: Optional[int] = None,
        workspace_name: Optional[str] = None,
    ):
        if not (workspace_name or workspace_id):
            raise ValueError("Workspace must be constructed with either a name or id")
        self._session = session
        self.id = workspace_id
        self.name = workspace_name

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
        assert not (self.id is None and self.name is None)
        if self.id is None:  # We know the name but not the ID
            workspaces_resp = bindings.get_GetWorkspaces(session=self._session, name=self.name)
            if len(workspaces_resp.workspaces) == 0:
                raise ValueError(f"No workspace found with name {self.name}")
            workspace_bindings = workspaces_resp.workspaces[0]
        else:
            workspace_resp = bindings.get_GetWorkspace(session=self._session, id=self.id)
            workspace_bindings = workspace_resp.workspace

        self._hydrate(workspace_bindings)
