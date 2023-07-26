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
        name: str = "",
    ):
        """Create a resource pool object.

        Arguments:
            session: The session to use for API calls.
            name: (Optional) Name of the resource pool.
        """
        self.name = name
        self._session = session

    @classmethod
    def _from_bindings(
        cls, resource_pool_bindings: bindings.v1ResourcePool, session: api.Session
    ) -> "ResourcePool":
        resource_pool = cls(session, name=resource_pool_bindings.name)
        return resource_pool

    def add_bindings(
        self,
        workspace_names: List[str],
    ) -> None:
        """Binds a resource pool to one or more workspaces.

        A resource pool with bindings can only be used by workspaces bound to it. Attempting to add
        a binding that already exists results or binding workspaces or resource pools
        that do not exist will result in errors.
        """
        req = bindings.v1BindRPToWorkspaceRequest(
            resourcePoolName=self.name,
            workspaceNames=workspace_names,
        )

        bindings.post_BindRPToWorkspace(self._session, body=req, resourcePoolName=self.name)

    def remove_bindings(
        self,
        workspace_names: List[str],
    ) -> None:
        """Unbinds a resource pool from one or more workspaces.

        A resource pool with bindings can only be used by workspaces bound to it. Attempting to
        remove a binding that does not exist results in a no-op.
        """
        req = bindings.v1UnbindRPFromWorkspaceRequest(
            resourcePoolName=self.name,
            workspaceNames=workspace_names,
        )

        bindings.delete_UnbindRPFromWorkspace(self._session, body=req, resourcePoolName=self.name)

    def list_workspaces(self) -> List[Optional[str]]:
        """Lists the workspaces bound to a specified resource pool.

        A resource pool with bindings can only be used by workspaces bound to it.
        """

        def get_with_offset(offset: int) -> bindings.v1ListWorkspacesBoundToRPResponse:
            return bindings.get_ListWorkspacesBoundToRP(
                session=self._session,
                offset=offset,
                resourcePoolName=self.name,
            )

        resps = api.read_paginated(get_with_offset)
        workspace_names = [
            workspace.Workspace(session=self._session, workspace_id=w).name
            for r in resps
            if r.workspaceIds is not None
            for w in r.workspaceIds
        ]

        return workspace_names

    def replace_bindings(
        self,
        workspace_names: List[str],
    ) -> None:
        """Replaces all the workspaces bound to a resource pool with those specified.

        If no bindings exist, new bindings will be added. Binding the same workspace more than once
        results in an SQL error. Binding workspaces or resource pools that do not exist result in
        Not Found errors.
        """
        req = bindings.v1OverwriteRPWorkspaceBindingsRequest(
            resourcePoolName=self.name,
            workspaceNames=workspace_names,
        )

        bindings.put_OverwriteRPWorkspaceBindings(
            self._session, body=req, resourcePoolName=self.name
        )
