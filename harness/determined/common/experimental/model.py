import datetime
import enum
import json
from typing import Any, Dict, Iterable, List, Optional

from determined.common import api, util
from determined.common.api import bindings
from determined.common.experimental import checkpoint, metrics


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

        req = bindings.v1PatchModelVersion(name=name)
        bindings.patch_PatchModelVersion(
            self._session, body=req, modelName=self.model_name, modelVersionNum=self.model_version
        )
        self.name = name

    def set_notes(self, notes: str) -> None:
        """
        Sets the human-friendly notes / readme for this model version

        Arguments:
            notes (string): Replaces notes for model version in registry
        """

        req = bindings.v1PatchModelVersion(notes=notes)
        bindings.patch_PatchModelVersion(
            self._session, body=req, modelName=self.model_name, modelVersionNum=self.model_version
        )
        self.notes = notes

    def delete(self) -> None:
        """
        Deletes the model version in the registry
        """
        bindings.delete_DeleteModelVersion(
            self._session, modelName=self.model_name, modelVersionNum=self.model_version
        )

    def get_metrics(self, group: Optional[str] = None) -> Iterable["metrics.TrialMetrics"]:
        """
        Gets all metrics for a given metric group associated with this model version.
        The checkpoint can be originally associated by calling
        ``core_context.experimental.report_task_using_model_version(<MODEL_VERSION>)``
        from within a task.

        Arguments:
            group (str, optional): Group name for the metrics (example: "training", "validation").
                All metrics will be returned when querying by None.
        """
        resp = bindings.get_GetTrialMetricsByModelVersion(
            session=self._session,
            modelName=self.model_name,
            modelVersionNum=self.model_version,
            trialSourceInfoType=bindings.v1TrialSourceInfoType.INFERENCE,
            metricGroup=group,
        )
        for d in resp.metrics:
            yield metrics.TrialMetrics._from_bindings(d, group)

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
        WORKSPACE
    """

    UNSPECIFIED = bindings.v1GetModelsRequestSortBy.UNSPECIFIED.value
    NAME = bindings.v1GetModelsRequestSortBy.NAME.value
    DESCRIPTION = bindings.v1GetModelsRequestSortBy.DESCRIPTION.value
    CREATION_TIME = bindings.v1GetModelsRequestSortBy.CREATION_TIME.value
    LAST_UPDATED_TIME = bindings.v1GetModelsRequestSortBy.LAST_UPDATED_TIME.value
    NUM_VERSIONS = bindings.v1GetModelsRequestSortBy.NUM_VERSIONS.value
    WORKSPACE = bindings.v1GetModelsRequestSortBy.WORKSPACE.value

    def _to_bindings(self) -> bindings.v1GetModelsRequestSortBy:
        return bindings.v1GetModelsRequestSortBy(self.value)


class ModelOrderBy(enum.Enum):
    """
    Specifies whether a sorted list of models should be in ascending or
    descending order.
    """

    ASCENDING = bindings.v1OrderBy.ASC.value
    ASC = bindings.v1OrderBy.ASC.value
    DESCENDING = bindings.v1OrderBy.DESC.value
    DESC = bindings.v1OrderBy.DESC.value

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
        workspace_id: Optional[int] = None,
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
        self.workspace_id = workspace_id
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
                sortBy=bindings.v1GetModelVersionsRequestSortBy.VERSION,
                orderBy=bindings.v1OrderBy.DESC,
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
                limit=None,
                modelName=self.name,
                offset=offset,
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
        updated_metadata = dict(self.metadata, **metadata)

        req = bindings.v1PatchModel(metadata=updated_metadata)
        bindings.patch_PatchModel(self._session, body=req, modelName=self.name)

        self.metadata = updated_metadata

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

        updated_metadata = dict(self.metadata)
        for key in keys:
            if key in updated_metadata:
                del updated_metadata[key]

        req = bindings.v1PatchModel(metadata=updated_metadata)
        bindings.patch_PatchModel(self._session, body=req, modelName=self.name)

        self.metadata = updated_metadata

    def move_to_workspace(self, workspace_name: str) -> None:
        req = bindings.v1PatchModel(workspaceName=workspace_name)
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

        labels = list(labels)

        req = bindings.v1PatchModel(labels=labels)
        bindings.patch_PatchModel(self._session, body=req, modelName=self.name)

        self.labels = labels

    def set_description(self, description: str) -> None:
        req = bindings.v1PatchModel(description=description)
        bindings.patch_PatchModel(self._session, body=req, modelName=self.name)

        self.description = description

    def archive(self) -> None:
        """
        Sets the model's state to archived
        """
        bindings.post_ArchiveModel(self._session, modelName=self.name)
        self.archived = True

    def unarchive(self) -> None:
        """
        Removes the model's archived state
        """
        bindings.post_UnarchiveModel(self._session, modelName=self.name)
        self.archived = False

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
            workspace_id=m.workspaceId,
        )
