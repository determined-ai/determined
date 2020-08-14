import datetime
import enum
import json
from typing import Any, Dict, List, Optional

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
    Class representing a model. Contains methods for managing metadata and
    model versions.

    Arguments:
        name (string): The name of the model.
        description (string, optional): The description of the model.
        creation_time (datetime): The time the model was created.
        last_updated_time (datetime): The time the model was most recently updated.
        metadata (dict, optional): User defined metadata associated with the checkpoint.
        master (string, optional): The address of the Determined master instance.
    """

    def __init__(
        self,
        name: str,
        description: str = "",
        creation_time: Optional[datetime.datetime] = None,
        last_updated_time: Optional[datetime.datetime] = None,
        metadata: Optional[Dict[str, Any]] = None,
        master: str = "",
    ):
        self._master = master
        self.name = name
        self.description = description
        self.creation_time = creation_time
        self.last_updated_time = last_updated_time
        self.metadata = metadata or {}

    def get_version(self, version: int = 0) -> Optional[Checkpoint]:
        """
        Retrieve the checkpoint corresponding to the specified version of the
        model. If no version is specified the latest model version is returned.

        Arguments:
            version (int, optional): The model version number requested.
        """
        if version == 0:
            resp = api.get(
                self._master,
                "/api/v1/models/{}/versions/".format(self.name),
                {"limit": 1, "order_by": 2},
            )

            data = resp.json()
            if data["modelVersions"] == []:
                return None

            latest_version = data["modelVersions"][0]
            return Checkpoint.from_json(
                {
                    **latest_version["checkpoint"],
                    "model_version": latest_version["version"],
                    "model_name": data["model"]["name"],
                }
            )
        else:
            resp = api.get(self._master, "/api/v1/models/{}/versions/{}".format(self.name, version))

        data = resp.json()
        return Checkpoint.from_json(data["model_version"]["checkpoint"], self._master)

    def get_versions(self, order_by: ModelOrderBy = ModelOrderBy.DESC) -> List[Checkpoint]:
        """
        Get a list of checkpoints corresponding to versions of this model. The
        models are sorted by version number and are returned in descending
        order by default.

        Arguments:
            order_by (enum): A member of the :class:`ModelOrderBy` enum.
        """
        resp = api.get(
            self._master,
            "/api/v1/models/{}/versions/".format(self.name),
            params={"order_by": order_by.value},
        )
        data = resp.json()

        return [
            Checkpoint.from_json(
                {
                    **version["checkpoint"],
                    "model_version": version["version"],
                    "model_name": data["model"]["name"],
                },
                self._master,
            )
            for version in data["modelVersions"]
        ]

    def register_version(self, checkpoint_uuid: str) -> Checkpoint:
        """
        Creates a new model version and returns the
        :class:`~determined.experimental.Checkpoint` corresponding to the
        version.

        Arguments:
            checkpoint_uuid: The uuid to associate with the new model version.
        """
        resp = api.post(
            self._master,
            "/api/v1/models/{}/versions".format(self.name),
            body={"checkpoint_uuid": checkpoint_uuid},
        )

        data = resp.json()

        return Checkpoint.from_json(
            {
                **data["modelVersion"]["checkpoint"],
                "model_version": data["modelVersion"]["version"],
                "model_name": data["modelVersion"]["model"]["name"],
            },
            self._master,
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

        api.patch(
            self._master,
            "/api/v1/models/{}".format(self.name),
            body={"model": {"metadata": self.metadata}},
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

        api.patch(
            self._master,
            "/api/v1/models/{}".format(self.name),
            body={"model": {"metadata": self.metadata}},
        )

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
