import datetime
import enum
import itertools
import json
import warnings
from typing import Any, Dict, Iterable, List, Optional

from determined.common import api, util
from determined.common.api import bindings
from determined.common.experimental import checkpoint, metrics

# TODO (MLG-1087): move OrderBy to experimental.client namespace
from determined.common.experimental._util import OrderBy  # noqa: I2041


class ModelVersion:
    """A class representing a combination of Model and Checkpoint.

    This class can be fetched using the
    :meth:`model.get_version() <determined.experimental.client.Model.get_version>` method.
    Once a model has been added to the registry, checkpoints can be added to it.
    These registered checkpoints are ModelVersions.

    Attributes:
        session: HTTP request session.
        model_version: (int) Version number assigned by the registry, starting from 1 and
            incrementing each time a new model version is registered.
        model_name: (str) Name of the parent model.
        checkpoint: (Mutable, Optional[checkpoint.Checkpoint]) Checkpoint associated with this
            model version.
        model_id: (Mutable, Optional[int]) ID of the parent model.
        metadata: (Mutable, Optional[Dict]) Metadata of this model version.
        name: (Mutable, Optional[str]) Human-friendly name of this model version.

    Note:
        All attributes are cached by default.

        Mutable properties may be changed by methods that update these values either automatically
        (eg. :meth:`set_name`, :meth:`set_notes`) or explicitly with :meth:`reload`.
    """

    def __init__(
        self,
        session: api.Session,
        model_version: int,
        model_name: str,
    ):
        self._session = session
        self.model_name = model_name
        self.model_version = model_version

        self.model_id: Optional[int] = None
        self.checkpoint: Optional[checkpoint.Checkpoint] = None
        self.metadata: Optional[Dict[str, Any]] = None
        self.name: Optional[str] = None
        self.comment: Optional[str] = None
        self.notes: Optional[str] = None

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
        warnings.warn(
            "Model.get_metrics() has been deprecated and will be removed in a future version."
            "Please call Model.iter_metrics() instead.",
            FutureWarning,
            stacklevel=2,
        )
        return self.iter_metrics(group=group)

    def iter_metrics(self, group: Optional[str] = None) -> Iterable["metrics.TrialMetrics"]:
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

    def _hydrate(self, model_version: bindings.v1ModelVersion) -> None:
        self.checkpoint = checkpoint.Checkpoint._from_bindings(
            model_version.checkpoint, self._session
        )
        self.metadata = model_version.metadata or {}
        self.comment = model_version.comment or ""
        self.notes = model_version.notes or ""
        self.model_version = model_version.version
        self.model_id = model_version.model.id
        self.name = model_version.name

    def reload(self) -> None:
        resp = bindings.get_GetModelVersion(
            session=self._session, modelName=self.model_name, modelVersionNum=self.model_version
        ).modelVersion
        self._hydrate(resp)

    @classmethod
    def _from_bindings(
        cls, version_bindings: bindings.v1ModelVersion, session: api.Session
    ) -> "ModelVersion":
        version = cls(
            session,
            model_version=version_bindings.version,
            model_name=version_bindings.model.name,
        )
        version._hydrate(version_bindings)
        return version


class ModelSortBy(enum.Enum):
    """
    Specifies the field to sort a list of models on.

    Attributes:
        NAME
        DESCRIPTION
        CREATION_TIME
        LAST_UPDATED_TIME
        WORKSPACE
    """

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

    This class is deprecated in favor of ``OrderBy`` and will be removed in a future
    release.
    """

    def __getattribute__(self, name: str) -> Any:
        warnings.warn(
            "'ModelOrderBy' is deprecated and will be removed in a future "
            "release. Please use 'experimental.OrderBy' instead.",
            FutureWarning,
            stacklevel=1,
        )
        return super().__getattribute__(name)

    ASCENDING = bindings.v1OrderBy.ASC.value
    ASC = bindings.v1OrderBy.ASC.value
    DESCENDING = bindings.v1OrderBy.DESC.value
    DESC = bindings.v1OrderBy.DESC.value

    def _to_bindings(self) -> bindings.v1OrderBy:
        return bindings.v1OrderBy(self.value)


class Model:
    """
    Class representing a model in the model registry.

    A Model object is usually obtained from :func:`determined.experimental.client.create_model`
    or :func:`determined.experimental.client.get_model`. It contains methods for model
    versions and metadata.
    """

    def __init__(
        self,
        session: api.Session,
        name: str,
    ):
        """Construct a new Model object.

        Args:
            session (api.Session): The session to use for API calls.
            name (string): The name of the model.
        """
        self._session = session
        self.name = name

        self.model_id: Optional[int] = None
        self.description: Optional[str] = None
        self.creation_time: Optional[datetime.datetime] = None
        self.last_updated_time: Optional[datetime.datetime] = None
        self.metadata: Optional[Dict[str, Any]] = None
        self.labels: Optional[List[str]] = None
        self.username: Optional[str] = None
        self.workspace_id: Optional[int] = None
        self.archived: Optional[bool] = None

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

    def get_versions(self, order_by: OrderBy = OrderBy.DESC) -> List[ModelVersion]:
        warnings.warn(
            "Model.get_versions() has been deprecated and will be removed in a future version."
            "Please call Model.list_versions() instead.",
            FutureWarning,
            stacklevel=2,
        )
        return list(self.list_versions(order_by=order_by))

    def list_versions(self, order_by: OrderBy = OrderBy.DESC) -> List[ModelVersion]:
        """
        Get a list of ModelVersions with checkpoints of this model.

        The model versions are sorted by model version ID and are returned in descending
        order by default.

        Arguments:
            order_by (enum): A member of the :class:`OrderBy` enum.
        """

        def get_with_offset(offset: int) -> bindings.v1GetModelVersionsResponse:
            return bindings.get_GetModelVersions(
                self._session,
                limit=None,
                modelName=self.name,
                offset=offset,
                orderBy=order_by._to_bindings(),
            )

        bindings_models: Iterable[bindings.v1ModelVersion] = itertools.chain.from_iterable(
            r.modelVersions for r in api.read_paginated(get_with_offset)
        )
        return [ModelVersion._from_bindings(m, self._session) for m in bindings_models]

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
        updated_metadata = dict(self.metadata, **metadata) if self.metadata else metadata

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

        updated_metadata = dict(self.metadata) if self.metadata else {}
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

    def __repr__(self) -> str:
        return "Model(id={}, name={}, metadata={})".format(
            self.model_id, self.name, json.dumps(self.metadata)
        )

    def _hydrate(self, model: bindings.v1Model) -> None:
        self.model_id = model.id
        self.description = model.description or ""
        self.creation_time = util.parse_protobuf_timestamp(model.creationTime)
        self.last_updated_time = util.parse_protobuf_timestamp(model.lastUpdatedTime)
        self.metadata = model.metadata or {}
        self.labels = list(model.labels or [])
        self.username = model.username or ""
        self.workspace_id = model.workspaceId
        self.archived = model.archived or False

    def reload(self) -> None:
        """
        Explicit refresh of cached properties.
        """
        resp = bindings.get_GetModel(session=self._session, modelName=self.name).model
        self._hydrate(resp)

    @classmethod
    def _from_bindings(cls, model_bindings: bindings.v1Model, session: api.Session) -> "Model":
        model = cls(
            session,
            name=model_bindings.name,
        )
        model._hydrate(model_bindings)
        return model
