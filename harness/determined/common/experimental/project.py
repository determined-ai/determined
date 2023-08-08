import json
from typing import Any, Dict, List, Optional

from determined.common import api
from determined.common.api import bindings

# Names of attributes in bindings.v1Project and their corresponding names in Project. These are
# limited to attributes that will be hydrated from bindings without changing their type.
HYDRATION_BINDINGS_TO_CLASS_ATTR = {
    "archived": "archived",
    "description": "description",
    "name": "name",
    "numExperiments": "n_experiments",
}

# The attributes of Project that can be modified on master by bindings calls.
PATCHABLE_ATTRS = {"description", "name"}


class Project:
    """A class representing a Project object.

    Attributes:
        workspace_id: (int) The ID of the workspace this project belongs to.
        id: (int) The ID of the project.
        archived: (Mutable, bool) True if experiment is archived, else false.
        description: (Mutable, str) The description of the project.
        name: (Mutable, str) Human-friendly name of the project.
        notes: (Mutable, List[Dict[str,str]) Notes about the project. As determined upstream,
            each note is a dict with exactly the keys "name" and "contents".
    """

    def __init__(
        self,
        session: api.Session,
        workspace_id: int,
        project_id: int,
    ):
        """Create a Project object.

        Arguments:
            session: The session to use for API calls.
            workspace_id: ID of the workspace this project belongs to.
            project_id: ID of the project.
        """
        self._session = session
        self.workspace_id = workspace_id
        self.id = project_id

        # These properties may be mutable and will be set by _hydrate()
        self.archived: Optional[bool] = None
        self.description: Optional[str] = None
        self.n_experiments: Optional[int] = None
        self.name: Optional[str] = None
        self.notes: Optional[List[Dict[str, str]]] = None

    @classmethod
    def _from_bindings(
        cls, project_bindings: bindings.v1Project, session: api.Session
    ) -> "Project":
        project = cls(session, workspace_id=project_bindings.id, project_id=project_bindings.id)
        project._hydrate(project_bindings)
        return project

    def _hydrate(self, project_bindings: bindings.v1Project) -> None:
        for bindings_attr, class_attr in HYDRATION_BINDINGS_TO_CLASS_ATTR.items():
            setattr(self, class_attr, getattr(project_bindings, bindings_attr))

        self.notes = [note.to_json() for note in project_bindings.notes]

    def reload(self) -> None:
        resp = bindings.get_GetProject(session=self._session, id=self.id)
        project_bindings = resp.project

        self._hydrate(project_bindings)

    def set(self, key: str, value: str) -> None:
        """Set an attribute on the project.

        This method can be used to set any attribute on the project that is settable with a PATCH.
        The attribute whose name is passed in 'key' will be set to 'value' on the master, and then
        will be set accordingly on the local object with whatever value master now thinks 'key' has.

        Args:
            key: The name of the attribute to set.
            value: The value to set the attribute to.

        Raises:
            ValueError: If the attribute is not settable
                (no bindings support for the attribute exists).
        """
        if key not in PATCHABLE_ATTRS:
            raise ValueError(f"Cannot set attribute {key} on Project.")

        patch_body = bindings.v1PatchProject(**{key: value})
        resp = bindings.patch_PatchProject(session=self._session, id=self.id, body=patch_body)

        setattr(self, key, getattr(resp.project, key))

    def to_json(self) -> Dict[str, Any]:
        """Dump this item as a json-shaped string.

        Emulates the bindings to_json() method.
        """
        return {
            "archived": self.archived,
            "description": self.description,
            "id": self.id,
            "numExperiments": self.n_experiments,
            "name": self.name,
            "notes": json.dumps(self.notes),
            "workspace_id": self.workspace_id,
        }

    def archive(self) -> None:
        bindings.post_ArchiveProject(session=self._session, id=self.id)
        self.archived = True

    def unarchive(self) -> None:
        bindings.post_UnarchiveProject(session=self._session, id=self.id)
        self.archived = False
