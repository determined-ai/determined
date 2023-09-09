import itertools
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
        n_active_experiments: (int) The number of active experiments in the project.
        n_experiments: (Mutable, int) The number of experiments in the project.
        name: (Mutable, str) Human-friendly name of the project.
        notes: (Mutable, List[Dict[str,str]) Notes about the project. As determined upstream,
            each note is a dict with exactly the keys "name" and "contents".
        username: (Mutable, str) The username of the project owner.
        workspace_id: (int) The ID of the workspace this project belongs to.
    """

    def __init__(
        self,
        session: api.Session,
        project_id: int,
    ):
        """Create a Project object.

        Arguments:
            session: The session to use for API calls.
            project_id: ID of the project.
        """
        self._session = session
        self.id = project_id

        # These properties may be mutable and will be set by _hydrate()
        self.archived: Optional[bool] = None
        self.description: Optional[str] = None
        self.n_active_experiments: Optional[int] = None
        self.n_experiments: Optional[int] = None
        self.name: Optional[str] = None
        self.notes: Optional[List[Dict[str, str]]] = None
        self.workspace_id: Optional[int] = None
        self.username: Optional[str] = None

    @classmethod
    def _from_bindings(
        cls, project_bindings: bindings.v1Project, session: api.Session
    ) -> "Project":
        project = cls(session, project_id=project_bindings.id)
        project._hydrate(project_bindings)
        return project

    def _hydrate(self, project_bindings: bindings.v1Project) -> None:
        """Set this object's mutable attributes from those in a bindings object."""
        self.archived = project_bindings.archived
        self.description = project_bindings.description
        self.n_active_experiments = project_bindings.numActiveExperiments
        self.n_experiments = project_bindings.numExperiments
        self.name = project_bindings.name
        self.notes = [note.to_json() for note in project_bindings.notes]
        self.workspace_id = project_bindings.workspaceId
        self.username = project_bindings.username

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

    def add_note(self, name: str, contents: str) -> None:
        """Add a note to the project.

        Because there is not yet functionality on the backend to add a single note, this method:
        1. fetches current notes for this project from the master.
        2. adds the new note to the list of notes.
        3. sends the updated list of notes to the master.

        WARNING:
        When it exits, the notes attribute of this object will be the same notes updated on master.
        This set of notes might reflect changes to the project that have happened since the project
        was last hydrated from master.

        Args:
            name: The name of the note.
            contents: The contents of the note.
        """
        master_notes = bindings.get_GetProject(session=self._session, id=self.id).project.notes
        combined_notes = list(master_notes) + [bindings.v1Note(name=name, contents=contents)]

        request_body = bindings.v1PutProjectNotesRequest(notes=combined_notes, projectId=self.id)
        resp = bindings.put_PutProjectNotes(
            session=self._session, body=request_body, projectId=self.id
        )

        self.notes = [note.to_json() for note in resp.notes]

    def remove_note(self, name: str) -> None:
        """Remove a note from the project.

        Because there is not yet functionality on the backend to remove a single note, this method:
        1. fetches current notes for this project from the master.
        2. removes the note with the passed name from the list of notes.
        3. sends the updated list of notes to the master.

        WARNING:
        When it exits, the notes attribute of this object will be the same notes updated on master.
        This set of notes might reflect changes to the project that have happened since the project
        was last hydrated from master.

        Args:
            name: The name of the note to remove.

        Raises:
            ValueError: If no note with the passed name is found
            NotImplementedError: If multiple notes with the passed name are found

        It is possible to for a project to have multiple notes with the same name. This method's
        interface doesn't make it possible to choose which of them to remove.
        """
        master_notes = bindings.get_GetProject(session=self._session, id=self.id).project.notes
        n_matching_notes = len([i for i, note in enumerate(master_notes) if note.name == name])
        if n_matching_notes == 0:
            raise ValueError(f"No note with name {name} found to remove.")

        if n_matching_notes > 1:
            raise NotImplementedError(
                f"Multiple notes with name {name} found. "
                "Please use the web UI to remove a specific one of them."
            )

        filtered_notes = [note for note in master_notes if note.name != name]

        request_body = bindings.v1PutProjectNotesRequest(notes=filtered_notes, projectId=self.id)
        resp = bindings.put_PutProjectNotes(
            session=self._session, body=request_body, projectId=self.id
        )

        self.notes = [note.to_json() for note in resp.notes]

    def to_json(self) -> Dict[str, Any]:
        """Return this object as a dict.

        Attributes may be renamed to accord with the database they're stored in.

        This methods is called `to_json` for historical consistency.
        """
        return {
            "archived": self.archived,
            "description": self.description,
            "id": self.id,
            "name": self.name,
            "notes": self.notes,
            "numActiveExperiments": self.n_active_experiments,
            "numExperiments": self.n_experiments,
            "workspaceId": self.workspace_id,
            "username": self.username,
        }
