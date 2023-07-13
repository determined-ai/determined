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
        if (workspace_id is None) ^ (workspace_name is None):
            raise ValueError("Workspace must be constructed with either a name or id (not both).")

        if workspace_id is None:
            assert workspace_name is not None
            self.id = _get_from_name(session, workspace_name).id
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


def _get_from_name(session: api.Session, name: str) -> bindings.v1Workspace:
    """Workspace lookup from master that relies on a workspace name."""
    resp = bindings.get_GetWorkspaces(session=session, name=name)
    if len(resp.workspaces) == 0:
        raise ValueError(f"No workspace found with name {name}")
    return resp.workspaces[0]
