import datetime
import enum
import json
import warnings
from typing import Any, Dict, List, Optional

from determined.common.experimental import checkpoint, session


class ModelVersion:
    """
    A ModelVersion object includes a Checkpoint,
    and can be fetched using ``model.get_version()``.
    """

    def __init__(
        self,
        session: session.Session,
        model_version_id: int,  # unique DB id
        checkpoint: checkpoint.Checkpoint,
        metadata: Dict[str, Any],
        name: Optional[str] = "",
        comment: Optional[str] = "",
        notes: Optional[str] = "",
        model_id: Optional[int] = 0,
        model_name: Optional[str] = "",
        model_version: Optional[int] = None,  # sequential
    ):
        self._session = session
        self.checkpoint = checkpoint
        self.metadata = metadata
        self.name = name
        self.comment = comment
        self.notes = notes
        self.model_id = model_id
        self.model_name = model_name
        self.model_version_id = model_version_id
        self.model_version = model_version

    def set_name(self, name: str) -> None:
        """
        Sets the human-friendly name for this model version

        Arguments:
            name (string): New name for model version
        """

        self.name = name
        self._session.patch(
            "/api/v1/models/{}/versions/{}".format(self.model_name, self.model_version_id),
            json={"name": self.name},
        )

    def set_notes(self, notes: str) -> None:
        """
        Sets the human-friendly notes / readme for this model version

        Arguments:
            notes (string): Replaces notes for model version in registry
        """

        self.notes = notes
        self._session.patch(
            "/api/v1/models/{}/versions/{}".format(self.model_name, self.model_version_id),
            json={"notes": self.notes},
        )

    def delete(self) -> None:
        """
        Deletes the model version in the registry
        """
        self._session.delete(
            "/api/v1/models/{}/versions/{}".format(self.model_name, self.model_version_id),
        )

    @classmethod
    def _from_json(cls, data: Dict[str, Any], session: session.Session) -> "ModelVersion":
        ckpt_data = data.get("checkpoint", {})
        ckpt = checkpoint.Checkpoint._from_json(ckpt_data, session)

        return cls(
            session,
            model_version_id=data.get("id", 1),
            checkpoint=ckpt,
            metadata=data.get("metadata", {}),
            name=data.get("name"),
            comment=data.get("comment"),
            notes=data.get("notes"),
            model_id=data.get("model", {}).get("id"),
            model_name=data.get("model", {}).get("name"),
            model_version=data.get("version"),
        )

    @classmethod
    def from_json(cls, data: Dict[str, Any], session: session.Session) -> "ModelVersion":
        warnings.warn(
            "ModelVersion.from_json() is deprecated and will be removed from the public API "
            "in a future version",
            FutureWarning,
        )
        return cls._from_json(data, session)


class ModelSortBy(enum.Enum):
    """
    Specifies the field to sort a list of models on.

    Attributes:
        UNSPECIFIED
        NAME
        DESCRIPTION
        CREATION_TIME
        LAST_UPDATED_TIME
    """

    UNSPECIFIED = 0
    NAME = 1
    DESCRIPTION = 2
    CREATION_TIME = 4
    LAST_UPDATED_TIME = 5


class ModelOrderBy(enum.Enum):
    """
    Specifies whether a sorted list of models should be in ascending or
    descending order.

    Attributes:
        ASCENDING
        ASC
        DESCENDING
        DESC
    """

    ASCENDING = 1
    ASC = 1
    DESCENDING = 2
    DESC = 2


class Model:
    """
    A Model object is usually obtained from
    ``determined.experimental.client.create_model()``
    or ``determined.experimental.client.get_model()``.

    Class representing a model in the model registry. It contains methods for model
    versions and metadata.

    Arguments:
        model_id (int): The unique id of this model.
        name (string): The name of the model.
        description (string, optional): The description of the model.
        creation_time (datetime): The time the model was created.
        last_updated_time (datetime): The time the model was most recently updated.
        metadata (dict, optional): User-defined metadata associated with the checkpoint.
        labels ([string]): User-defined text labels associated with the checkpoint.
        username (string): The user who initially created this model.
        archived (boolean): The status (archived or not) for this model.
    """

    def __init__(
        self,
        session: session.Session,
        model_id: int,
        name: str,
        description: str = "",
        creation_time: Optional[datetime.datetime] = None,
        last_updated_time: Optional[datetime.datetime] = None,
        metadata: Optional[Dict[str, Any]] = None,
        labels: Optional[List[str]] = None,
        username: str = "",
        archived: bool = False,
    ):
        self._session = session
        self.model_id = model_id
        self.name = name
        self.description = description
        self.creation_time = creation_time
        self.last_updated_time = last_updated_time
        self.metadata = metadata or {}
        self.labels = labels
        self.username = username
        self.archived = archived

    def get_version(self, version: int = -1) -> Optional[ModelVersion]:
        """
        Retrieve the checkpoint corresponding to the specified id of the
        model version. If the specified version of the model does not exist,
        an exception is raised.

        If no version is specified, the latest version of the model is
        returned. In this case, if there are no registered versions of the
        model, ``None`` is returned.

        Arguments:
            version (int, optional): The model version ID requested.
        """
        if version == -1:
            resp = self._session.get(
                "/api/v1/models/{}/versions/".format(self.name),
                {"limit": 1, "order_by": ModelOrderBy.DESC.value},
            )

            data = resp.json()
            if data["modelVersions"] == []:
                return None

            latest_version = data["modelVersions"][0]
            return ModelVersion._from_json(
                latest_version,
                self._session,
            )
        else:
            resp = self._session.get("/api/v1/models/{}/versions/{}".format(self.name, version))

        data = resp.json()
        return ModelVersion._from_json(data["modelVersion"], self._session)

    def get_versions(self, order_by: ModelOrderBy = ModelOrderBy.DESC) -> List[ModelVersion]:
        """
        Get a list of ModelVersions with checkpoints of this model. The
        model versions are sorted by model version ID and are returned in descending
        order by default.

        Arguments:
            order_by (enum): A member of the :class:`ModelOrderBy` enum.
        """
        resp = self._session.get(
            "/api/v1/models/{}/versions/".format(self.name),
            params={"order_by": order_by.value},
        )
        data = resp.json()

        return [
            ModelVersion._from_json(
                version,
                self._session,
            )
            for version in data["modelVersions"]
        ]

    def register_version(self, checkpoint_uuid: str) -> ModelVersion:
        """
        Creates a new model version and returns the
        :class:`~determined.experimental.ModelVersion` corresponding to the
        version.

        Arguments:
            checkpoint_uuid: The UUID of the checkpoint to register.
        """
        resp = self._session.post(
            "/api/v1/models/{}/versions".format(self.name),
            json={"checkpointUuid": checkpoint_uuid},
        )

        data = resp.json()
        return ModelVersion._from_json(
            data["modelVersion"],
            self._session,
        )

    def add_metadata(self, metadata: Dict[str, Any]) -> None:
        """
        Adds user-defined metadata to the model. The ``metadata`` argument must be a
        JSON-serializable dictionary. If any keys from this dictionary already appear in
        the model's metadata, the previous dictionary entries are replaced.

        Arguments:
            metadata (dict): Dictionary of metadata to add to the model.
        """
        for key, val in metadata.items():
            self.metadata[key] = val

        self._session.patch(
            "/api/v1/models/{}".format(self.name),
            json={"metadata": self.metadata, "description": self.description},
        )

    def remove_metadata(self, keys: List[str]) -> None:
        """
        Removes user-defined metadata from the model. Any top-level keys that
        appear in the ``keys`` list are removed from the model.

        Arguments:
            keys (List[string]): Top-level keys to remove from the model metadata.
        """
        for key in keys:
            if key in self.metadata:
                del self.metadata[key]

        self._session.patch(
            "/api/v1/models/{}".format(self.name),
            json={"metadata": self.metadata, "description": self.description},
        )

    def set_labels(self, labels: List[str]) -> None:
        """
        Sets user-defined labels for the model. The ``labels`` argument must be an
        array of strings. If the model previously had labels, they are replaced.

        Arguments:
            labels (List[string]): All labels to set on the model.
        """
        self.labels = labels
        self._session.patch(
            "/api/v1/models/{}".format(self.name),
            json={"labels": self.labels},
        )

    def set_description(self, description: str) -> None:
        self.description = description
        self._session.patch(
            "/api/v1/models/{}".format(self.name),
            json={"description": description},
        )

    def archive(self) -> None:
        """
        Sets the model's state to archived
        """
        self.archived = True
        self._session.post(
            "/api/v1/models/{}/archive".format(self.name),
        )

    def unarchive(self) -> None:
        """
        Removes the model's archived state
        """
        self.archived = False
        self._session.post(
            "/api/v1/models/{}/unarchive".format(self.name),
        )

    def delete(self) -> None:
        """
        Deletes the model in the registry
        """
        self._session.delete(
            "/api/v1/models/{}".format(self.name),
        )

    def to_json(self) -> Dict[str, Any]:
        return {
            "name": self.name,
            "id": self.model_id,
            "description": self.description,
            "creation_time": self.creation_time,
            "last_updated_time": self.last_updated_time,
            "metadata": self.metadata,
            "archived": self.archived,
        }

    def __repr__(self) -> str:
        return "Model(id={}, name={}, metadata={})".format(
            self.model_id, self.name, json.dumps(self.metadata)
        )

    @classmethod
    def _from_json(cls, data: Dict[str, Any], session: session.Session) -> "Model":
        return cls(
            session,
            data["id"],
            data["name"],
            data.get("description", ""),
            data.get("creationTime"),
            data.get("lastUpdatedTime"),
            data.get("metadata", {}),
            data.get("labels", []),
            data.get("username", ""),
            data.get("archived", False),
        )

    @classmethod
    def from_json(cls, data: Dict[str, Any], session: session.Session) -> "Model":
        warnings.warn(
            "Model.from_json() is deprecated and will be removed from the public API "
            "in a future version",
            FutureWarning,
        )
        return cls._from_json(data, session)
