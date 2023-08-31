import itertools
import json
from typing import Any, Dict, List, Optional

from determined.common import api
from determined.common.api import bindings
from determined.common.experimental import experiment


class Project:
    """A class representing a Project object.

    Attributes:
        id: (int) The ID of the project.
        archived: (Mutable, bool) True if experiment is archived, else false.
        description: (Mutable, str) The description of the project.
        name: (Mutable, str) Human-friendly name of the project.
        notes: (Mutable, List[Dict[str,str]) Notes about the project. As determined upstream,
            each note is a dict with exactly the keys "name" and "contents".
        workspace_id: (int) The ID of the workspace this project belongs to.
    """

    def __init__(
        self,
        session: api.Session,
        id: int,
    ):
        """Create a Project object.

        Arguments:
            session: The session to use for API calls.
            project_id: ID of the project.
        """
        self._session = session
        self.id = id

        # These properties may be mutable and will be set by _hydrate()
        self.archived: Optional[bool] = None
        self.description: Optional[str] = None
        self.name: Optional[str] = None
        self.notes: Optional[List[Dict[str, str]]] = None
        self.workspace_id: Optional[int] = None

    @classmethod
    def _from_bindings(
        cls, project_bindings: bindings.v1Project, session: api.Session
    ) -> "Project":
        project = cls(session, id=project_bindings.id)
        project._hydrate(project_bindings)
        return project

    def _hydrate(self, project_bindings: bindings.v1Project) -> None:
        """Set this object's mutable attributes from those in a bindings object."""
        self.archived = project_bindings.archived
        self.description = project_bindings.description
        self.name = project_bindings.name
        self.notes = [note.to_json() for note in project_bindings.notes]
        self.workspace_id = project_bindings.workspaceId

    def reload(self) -> None:
        resp = bindings.get_GetProject(session=self._session, id=self.id)
        project_bindings = resp.project

        self._hydrate(project_bindings)

    def list_experiments(self) -> List["experiment.Experiment"]:
        def get_with_offset(offset: int) -> bindings.v1GetExperimentsResponse:
            return bindings.get_GetExperiments(
                session=self._session, projectId=self.id, offset=offset
            )

        exp_bindings = itertools.chain.from_iterable(
            r.experiments for r in api.read_paginated(get_with_offset)
        )
        return [experiment.Experiment._from_bindings(exp, self._session) for exp in exp_bindings]

    def set_description(self, description: str) -> None:
        """Set the description of the project.

        The attribute will be changed both on master and this local object.

        Args:
            description: The description to set.
        """
        patch_body = bindings.v1PatchProject(description=description)
        resp = bindings.patch_PatchProject(session=self._session, id=self.id, body=patch_body)

        self.description = resp.project.description

    def set_name(self, name: str) -> None:
        """Set the name of the project.

        The attribute will be changed both on master and this local object.

        Args:
            name: The name to set.
        """
        patch_body = bindings.v1PatchProject(name=name)
        resp = bindings.patch_PatchProject(session=self._session, id=self.id, body=patch_body)

        self.name = resp.project.name

    def archive(self) -> None:
        """Set the project to archived (archived = True).

        As with other setters, this will change the attribute both on master and this local object.
        """
        bindings.post_ArchiveProject(session=self._session, id=self.id)
        self.archived = True

    def unarchive(self) -> None:
        """Set the project to unarchived (archived = False).

        As with other setters, this will change the attribute both on master and this local object.
        """
        bindings.post_UnarchiveProject(session=self._session, id=self.id)
        self.archived = False

    def to_json(self) -> Dict[str, Any]:
        """Dump this item as a json-shaped string.

        Emulates the bindings to_json() method.
        """
        return {
            "archived": self.archived,
            "description": self.description,
            "id": self.id,
            "name": self.name,
            "notes": json.dumps(self.notes),
            "workspace_id": self.workspace_id,
        }
