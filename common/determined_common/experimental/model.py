import datetime
import enum
import json
from typing import Any, Dict, List, Optional

import determined_client
from determined_client.rest import ApiException

from determined_common import api
from determined_common.experimental.checkpoint import Checkpoint


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
    Class representing a model in the model registry. It contains methods for model
    versions and metadata.

    Arguments:
        name (string): The name of the model.
        description (string, optional): The description of the model.
        creation_time (datetime): The time the model was created.
        last_updated_time (datetime): The time the model was most recently updated.
        metadata (dict, optional): User-defined metadata associated with the checkpoint.
        master (string, optional): The address of the Determined master instance.
    """

    def __init__(
        self,
        name: str,
        description: str = "",
        creation_time: Optional[datetime.datetime] = None,
        last_updated_time: Optional[datetime.datetime] = None,
        metadata: Optional[Dict[str, Any]] = None,
        api_client=None,
        master: str = "",
    ):
        self._master = master
        self.api_client = api_client
        self.model_api = determined_client.api.models_api.ModelsApi(self.api_client)
        self.name = name
        self.description = description
        self.creation_time = creation_time
        self.last_updated_time = last_updated_time
        self.metadata = metadata or {}

    def get_version(self, version: int = 0) -> Optional[Checkpoint]:
        """
        Retrieve the checkpoint corresponding to the specified version of the
        model. If the specified version of the model does not exist, an exception
        is raised.

        If no version is specified, the latest version of the model is
        returned. In this case, if there are no registered versions of the
        model, ``None`` is returned.

        Arguments:
            version (int, optional): The model version number requested.
        """
        if version == 0:
            # resp = api.get(
            #     self._master,
            #     "/api/v1/models/{}/versions/".format(self.name),
            #     {"limit": 1, "order_by": 2},
            # )

            model_versions_response = self.model_api.determined_get_model_versions(
                model_name=self.name, limit=1, order_by=2
            )

            if not model_versions_response.model_versions:
                return None

            latest_version = model_versions_response.model_versions[0]
            return Checkpoint.from_spec(
                api_client=self.api_client,
                checkpoint_object=latest_version.checkpoint,
                model_name=self.name,
                model_version=latest_version.version,
            )

        model_version_response = self.model_api.determined_get_model_version(self.name, version)
        return Checkpoint.from_spec(
            api_client=self.api_client,
            checkpoint_object=model_version_response.model_version.checkpoint,
            model_name=self.name,
            model_version=model_version_response.model_version.version,
        )

    def get_versions(self, order_by: ModelOrderBy = ModelOrderBy.DESC) -> List[Checkpoint]:
        """
        Get a list of checkpoints corresponding to versions of this model. The
        models are sorted by version number and are returned in descending
        order by default.

        Arguments:
            order_by (enum): A member of the :class:`ModelOrderBy` enum.
        """

        model_versions_response = self.model_api.determined_get_model_versions(model_name=self.name)

        # resp = api.get(
        #     self._master,
        #     "/api/v1/models/{}/versions/".format(self.name),
        #     params={"order_by": order_by.value},
        # )
        # data = resp.json()

        return [
            Checkpoint.from_spec(
                api_client=self.api_client,
                checkpoint_object=model_version.checkpoint,
                model_name=self.name,
                model_version=model_version.version,
            )
            for model_version in model_versions_response.model_versions
        ]

    def register_version(self, checkpoint_uuid: str) -> Checkpoint:
        """
        Creates a new model version and returns the
        :class:`~determined.experimental.Checkpoint` corresponding to the
        version.

        Arguments:
            checkpoint_uuid: The UUID of the checkpoint to register.
        """
        model_version_request = (
            determined_client.models.v1_post_model_version_request.V1PostModelVersionRequest(
                model_name=self.name, checkpoint_uuid=checkpoint_uuid
            )
        )
        model_version_response = self.model_api.determined_post_model_version(
            model_version_request, self.name
        )

        return Checkpoint.from_spec(
            api_client=self.api_client,
            checkpoint_object=model_version_response.model_version.checkpoint,
            model_name=model_version_response.model_version.model.name,
            model_version=model_version_response.model_version.version,
        )

    def add_metadata(self, metadata: Dict[str, Any]):
        """
        Adds user-defined metadata to the model. The ``metadata`` argument must be a
        JSON-serializable dictionary. If any keys from this dictionary already appear in
        the model's metadata, the previous dictionary entries are replaced.

        Arguments:
            metadata (dict): Dictionary of metadata to add to the model.
        """
        for key, val in metadata.items():
            self.metadata[key] = val

        # api.patch(
        #     self._master,
        #     "/api/v1/models/{}".format(self.name),
        #     body={"model": {"metadata": self.metadata, "description": self.description}},
        # )
        model = determined_client.models.v1_model.V1Model(
            name=self.name,
            description=self.description,
            metadata=self.metadata,
            creation_time=self.creation_time,
            last_updated_time=self.last_updated_time,
        )
        body = determined_client.models.v1_patch_model_request.V1PatchModelRequest(model=model)
        patch_model_response = self.model_api.determined_patch_model(
            body=body, model_name=self.name
        )

        return Model.from_spec(patch_model_response.model, self.api_client)

    def remove_metadata(self, keys: List[str]):
        """
        Removes user-defined metadata from the model. Any top-level keys that
        appear in the ``keys`` list are removed from the model.

        Arguments:
            keys (List[string]): Top-level keys to remove from the model metadata.
        """
        for key in keys:
            if key in self.metadata:
                del self.metadata[key]

        model = determined_client.models.v1_model.V1Model(
            name=self.name,
            description=self.description,
            metadata=self.metadata,
            creation_time=self.creation_time,
            last_updated_time=self.last_updated_time,
        )
        body = determined_client.models.v1_patch_model_request.V1PatchModelRequest(model=model)
        patch_model_response = self.model_api.determined_patch_model(
            body=body, model_name=self.name
        )

        return Model.from_spec(patch_model_response.model, self.api_client)

    def to_json(self) -> Dict[str, Any]:
        return {
            "name": self.name,
            "description": self.description,
            "creation_time": self.creation_time,
            "last_updated_time": self.last_updated_time,
            "metadata": self.metadata,
        }

    def __repr__(self) -> str:
        return "Model(name={}, metadata={})".format(self.name, json.dumps(self.metadata))

    @staticmethod
    def from_json(data: Dict[str, Any], master: str) -> "Model":
        return Model(
            data["name"],
            data.get("description", ""),
            data.get("creationTime"),
            data.get("lastUpdatedTime"),
            data.get("metadata", {}),
            master,
        )

    @classmethod
    def from_spec(cls, model_object, api_client):
        return cls(
            model_object.name,
            model_object.description,
            model_object.creation_time,
            model_object.last_updated_time,
            model_object.metadata,
            api_client,
            master=api_client.configuration.host,
        )
