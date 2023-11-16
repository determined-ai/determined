import itertools
from typing import Iterable, List, Optional

from determined.common import api
from determined.common.api import bindings, errors
from determined.common.experimental import project, resource_pool


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

    def list_pools(self) -> List[resource_pool.ResourcePool]:
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
        resource_pool_names = [
            rp for r in resps if r.resourcePools is not None for rp in r.resourcePools
        ]

        return [resource_pool.ResourcePool(self._session, name=rp) for rp in resource_pool_names]

    def create_project(self, name: str, description: Optional[str] = None) -> project.Project:
        """Creates a new project in this workspace with the provided name.

        Args:
            name: The name of the project to create.
            description: Optional description to give the new project.

        Returns:
            The newly-created :class:`~determined.experimental.Workspace`.

        Raises:
            errors.APIException: If the project with the passed name already exists.
        """
        req = bindings.v1PostProjectRequest(name=name, workspaceId=self.id, description=description)
        resp = bindings.post_PostProject(self._session, workspaceId=self.id, body=req)
        return project.Project._from_bindings(resp.project, self._session)

    def delete_project(self, name: str) -> None:
        """Deletes a project from this workspace.

        Args:
            name: The name of the project to delete.

        Raises:
            errors.NotFoundException: If the project with the passed name is not found.
        """
        project_id = self.get_project(name).id
        bindings.delete_DeleteProject(session=self._session, id=project_id)

    def get_project(self, project_name: str) -> project.Project:
        """Gets a project that is a part of this workspace.

        Args:
            project_name: The name of the project to get.

        Raises:
            errors.NotFoundException: If the project with the passed name is not found.
        """
        projects = bindings.get_GetWorkspaceProjects(
            session=self._session,
            id=self.id,
            name=project_name,
        ).projects
        if projects:
            return project.Project._from_bindings(projects[0], self._session)
        else:
            raise errors.NotFoundException(f"Project '{project_name}' not found in this workspace.")

    def list_projects(self) -> List[project.Project]:
        """Lists all projects that are a part of this workspace."""

        def get_with_offset(offset: int) -> bindings.v1GetWorkspaceProjectsResponse:
            return bindings.get_GetWorkspaceProjects(
                session=self._session,
                offset=offset,
                id=self.id,
            )

        bindings_projects: Iterable[bindings.v1Project] = itertools.chain.from_iterable(
            r.projects for r in api.read_paginated(get_with_offset)
        )

        return [project.Project._from_bindings(p, self._session) for p in bindings_projects]


def _get_workspace_id_from_name(session: api.Session, name: str) -> int:
    """Workspace lookup from master that relies on a workspace name."""
    resp = bindings.get_GetWorkspaces(session=session, name=name)
    if len(resp.workspaces) == 0:
        raise errors.NotFoundException(f"No workspace found with name {name}")
    assert len(resp.workspaces) < 2, f"Multiple workspaces found with name {name}"

    return resp.workspaces[0].id
