from typing import List, Optional

from determined.common import api
from determined.common.api import bindings
from determined.common.experimental import workspace


class ResourcePool:
    """A class representing a resource pool object.

    Attributes:
        name: (str) The name of the resource pool.
    """

    def __init__(
        self,
        session: api.Session,
        name: Optional[str] = None,
    ):
        """Create a resource pool object.

        Arguments:
            session: The session to use for API calls.
            workspace_name: (Optional) Name of the resource pool.
        """
        self.name = name
        self._session = session

    @classmethod
    def _from_bindings(
        cls, resource_pool_bindings: bindings.v1ResourcePool, session: api.Session
    ) -> "ResourcePool":
        resource_pool = cls(session, name=resource_pool_bindings.name)
        return resource_pool

    def bind(
        self,
        workspace_names: List[str],
    ) -> None:
        """
        Bind a resource pool to workspaces.

        Arguments:
            workspace_names (list(str)): The names of the workspaces to be bound.
        """
        req = bindings.v1BindRPToWorkspaceRequest(
            resourcePoolName=self.name,
            workspaceNames=workspace_names,
        )

        bindings.post_BindRPToWorkspace(self._session, body=req, resourcePoolName=self.name)

    def unbind(
        self,
        workspace_names: List[str],
    ) -> None:
        """
        Unbind a resource pool from workspaces.

        Arguments:
            workspace_names (list(str)): The names of the workspaces to be unbound.
        """
        req = bindings.v1UnbindRPFromWorkspaceRequest(
            resourcePoolName=self.name,
            workspaceIds=workspace_names,
        )

        bindings.delete_UnbindRPFromWorkspace(self._session, body=req, resourcePoolName=self.name)

    def workspaces(self) -> List[str]:
        """
        List workspaces bound to a resource pool.

        Returns:
            (List(str)) The names of workspaces bound to the resource pool.
        """

        def get_with_offset(offset: int) -> bindings.v1ListWorkspacesBoundToRPResponse:
            return bindings.get_ListWorkspacesBoundToRP(
                session=self._session,
                offset=offset,
                resourcePoolName=self.name,
            )

        resps = api.read_paginated(get_with_offset)
        workspace_names = [
            workspace.Workspace(session=self._session, workspace_id=w).workspace_name
            for r in resps
            if r.workspaceIds is not None
            for w in r.workspaceIds
        ]

        return workspace_names

    def set_bindings(
        self,
        workspace_names: List[str],
    ) -> None:
        """
        Overwrite the workspaces bound to a resource pool.

        Arguments:
            workspace_names (list(str)): The names of the workspaces to overwrite
                existing workspaces bound to the resource pool.
        """
        req = bindings.v1OverwriteRPWorkspaceBindingsRequest(
            resourcePoolName=self.name,
            workspaceNames=workspace_names,
        )

        bindings.put_OverwriteRPWorkspaceBindings(
            self._session, body=req, resourcePoolName=self.name
        )
