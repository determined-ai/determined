import datetime
import enum
import json
import warnings
from typing import Any, Dict, Iterable, List, Optional

from determined.common import api, util
from determined.common.api import bindings
from determined.common.experimental import checkpoint


class ModelVersion:
    """
    A ModelVersion object includes a Checkpoint,
    and can be fetched using ``model.get_version()``.
    """

    def __init__(
        self,
        session: api.Session,
        model_version_id: int,  # unique DB id
        checkpoint: checkpoint.Checkpoint,
        metadata: Dict[str, Any],
        name: str,
        comment: str,
        notes: str,
        model_id: int,
        model_name: str,
        model_version: int,  # sequential
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
        req = bindings.v1PatchModelVersion(name=name)
        bindings.patch_PatchModelVersion(
            self._session, body=req, modelName=self.model_name, modelVersionNum=self.model_version
        )

    def set_notes(self, notes: str) -> None:
        """
        Sets the human-friendly notes / readme for this model version

        Arguments:
            notes (string): Replaces notes for model version in registry
        """

        self.notes = notes
        req = bindings.v1PatchModelVersion(notes=notes)
        bindings.patch_PatchModelVersion(
            self._session, body=req, modelName=self.model_name, modelVersionNum=self.model_version
        )

    def delete(self) -> None:
        """
        Deletes the model version in the registry
        """
        bindings.delete_DeleteModelVersion(
            self._session, modelName=self.model_name, modelVersionNum=self.model_version
        )

    @classmethod
    def _from_json(cls, data: Dict[str, Any], session: api.Session) -> "ModelVersion":
        return cls(
            session,
            model_version_id=data.get("id", 1),
            checkpoint=checkpoint.Checkpoint._from_json(data["checkpoint"], session),
            metadata=data.get("metadata", {}),
            name=data.get("name", ""),
            comment=data.get("comment", ""),
            notes=data.get("notes", ""),
            model_id=data["model"]["id"],
            model_name=data["model"]["name"],
            model_version=data["version"],
        )

    @classmethod
    def from_json(cls, data: Dict[str, Any], session: api.Session) -> "ModelVersion":
        warnings.warn(
            "ModelVersion.from_json() is deprecated and will be removed from the public API "
            "in a future version",
            FutureWarning,
        )
        return cls._from_json(data, session)

    @classmethod
    def _from_bindings(cls, m: bindings.v1ModelVersion, session: api.Session) -> "ModelVersion":
        return cls(
            session,
            model_version_id=m.id,
            checkpoint=checkpoint.Checkpoint._from_bindings(m.checkpoint, session),
            metadata=m.metadata or {},
            name=m.name or "",
            comment=m.comment or "",
            notes=m.notes or "",
            model_id=m.model.id,
            model_name=m.model.name,
            model_version=m.version,
        )


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

    UNSPECIFIED = bindings.v1GetModelsRequestSortBy.SORT_BY_UNSPECIFIED.value
    NAME = bindings.v1GetModelsRequestSortBy.SORT_BY_NAME.value
    DESCRIPTION = bindings.v1GetModelsRequestSortBy.SORT_BY_DESCRIPTION.value
    CREATION_TIME = bindings.v1GetModelsRequestSortBy.SORT_BY_CREATION_TIME.value
    LAST_UPDATED_TIME = bindings.v1GetModelsRequestSortBy.SORT_BY_LAST_UPDATED_TIME.value
    NUM_VERSIONS = bindings.v1GetModelsRequestSortBy.SORT_BY_NUM_VERSIONS.value

    def _to_bindings(self) -> bindings.v1GetModelsRequestSortBy:
        return bindings.v1GetModelsRequestSortBy(self.value)


class ModelOrderBy(enum.Enum):
    """
    Specifies whether a sorted list of models should be in ascending or
    descending order.
    """

    ASCENDING = bindings.v1OrderBy.ORDER_BY_ASC.value
    ASC = bindings.v1OrderBy.ORDER_BY_ASC.value
    DESCENDING = bindings.v1OrderBy.ORDER_BY_DESC.value
    DESC = bindings.v1OrderBy.ORDER_BY_DESC.value

    def _to_bindings(self) -> bindings.v1OrderBy:
        return bindings.v1OrderBy(self.value)


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
        session: api.Session,
        model_id: int,
        name: str,
        description: str,
        creation_time: datetime.datetime,
        last_updated_time: datetime.datetime,
        metadata: Dict[str, Any],
        labels: List[str],
        username: str,
        archived: bool,
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
            resp = bindings.get_GetModelVersions(
                self._session,
                modelName=self.name,
                limit=1,
                sortBy=bindings.v1GetModelVersionsRequestSortBy.SORT_BY_VERSION,
                orderBy=bindings.v1OrderBy.ORDER_BY_DESC,
            )
            if not resp.modelVersions:
                return None
            return ModelVersion._from_bindings(resp.modelVersions[0], self._session)

        r = bindings.get_GetModelVersion(
            self._session, modelName=self.name, modelVersionNum=version
        )

        return ModelVersion._from_bindings(r.modelVersion, self._session)

    def get_versions(self, order_by: ModelOrderBy = ModelOrderBy.DESC) -> List[ModelVersion]:
        """
        Get a list of ModelVersions with checkpoints of this model. The
        model versions are sorted by model version ID and are returned in descending
        order by default.

        Arguments:
            order_by (enum): A member of the :class:`ModelOrderBy` enum.
        """

        def get_with_offset(offset: int) -> bindings.v1GetModelVersionsResponse:
            return bindings.get_GetModelVersions(
                self._session,
                modelName=self.name,
                orderBy=order_by._to_bindings(),
            )

        resps = api.read_paginated(get_with_offset)

        return [
            ModelVersion._from_bindings(m, self._session) for r in resps for m in r.modelVersions
        ]

    def register_version(self, checkpoint_uuid: str) -> ModelVersion:
        """
        Creates a new model version and returns the
        :class:`~determined.experimental.ModelVersion` corresponding to the
        version.

        Arguments:
            checkpoint_uuid: The UUID of the checkpoint to register.
        """

        req = bindings.v1PostModelVersionRequest(
            checkpointUuid=checkpoint_uuid, modelName=self.name
        )

        resp = bindings.post_PostModelVersion(self._session, body=req, modelName=self.name)

        return ModelVersion._from_bindings(resp.modelVersion, self._session)

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

        req = bindings.v1PatchModel(metadata=self.metadata)
        bindings.patch_PatchModel(self._session, body=req, modelName=self.name)

    def remove_metadata(self, keys: List[str]) -> None:
        """
        Removes user-defined metadata from the model. Any top-level keys that
        appear in the ``keys`` list are removed from the model.

        Arguments:
            keys (List[str]): Top-level keys to remove from the model metadata.
        """
        if not isinstance(keys, Iterable) or not all(isinstance(k, str) for k in keys):
            raise ValueError(
                f"remove_metadata() requires a list of strings as input but got: {keys}"
            )

        for key in keys:
            if key in self.metadata:
                del self.metadata[key]

        req = bindings.v1PatchModel(metadata=self.metadata)
        bindings.patch_PatchModel(self._session, body=req, modelName=self.name)

    def set_labels(self, labels: List[str]) -> None:
        """
        Sets user-defined labels for the model. The ``labels`` argument must be an
        array of strings. If the model previously had labels, they are replaced.

        Arguments:
            labels (List[str]): All labels to set on the model.
        """
        if not isinstance(labels, Iterable):
            raise ValueError(f"set_labels() requires a list of strings as input but got: {labels}")

        self.labels = list(labels)

        req = bindings.v1PatchModel(labels=self.labels)
        bindings.patch_PatchModel(self._session, body=req, modelName=self.name)

    def set_description(self, description: str) -> None:
        self.description = description
        req = bindings.v1PatchModel(description=self.description)
        bindings.patch_PatchModel(self._session, body=req, modelName=self.name)

    def archive(self) -> None:
        """
        Sets the model's state to archived
        """
        self.archived = True
        bindings.post_ArchiveModel(self._session, modelName=self.name)

    def unarchive(self) -> None:
        """
        Removes the model's archived state
        """
        self.archived = False
        bindings.post_UnarchiveModel(self._session, modelName=self.name)

    def delete(self) -> None:
        """
        Deletes the model in the registry
        """
        bindings.delete_DeleteModel(self._session, modelName=self.name)

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
    def _from_json(cls, data: Dict[str, Any], session: api.Session) -> "Model":
        return cls(
            session,
            model_id=data["id"],
            name=data["name"],
            description=data.get("description", ""),
            creation_time=util.parse_protobuf_timestamp(data["creationTime"]),
            last_updated_time=util.parse_protobuf_timestamp(data["lastUpdatedTime"]),
            metadata=data.get("metadata", {}),
            labels=data.get("labels", []),
            username=data.get("username", ""),
            archived=data.get("archived", False),
        )

    @classmethod
    def from_json(cls, data: Dict[str, Any], session: api.Session) -> "Model":
        warnings.warn(
            "Model.from_json() is deprecated and will be removed from the public API "
            "in a future version",
            FutureWarning,
        )
        return cls._from_json(data, session)

    @classmethod
    def _from_bindings(cls, m: bindings.v1Model, session: api.Session) -> "Model":
        return cls(
            session,
            model_id=m.id,
            name=m.name,
            description=m.description or "",
            creation_time=util.parse_protobuf_timestamp(m.creationTime),
            last_updated_time=util.parse_protobuf_timestamp(m.lastUpdatedTime),
            metadata=m.metadata,
            labels=list(m.labels or []),
            username=m.username or "",
            archived=m.archived or False,
        )
