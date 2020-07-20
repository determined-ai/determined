import datetime
import enum
from typing import Any, Dict, List, Optional

from determined_common import api
from determined_common.experimental.checkpoint import Checkpoint


class ModelSortBy(enum.Enum):
    UNSPECIFIED = 0
    NAME = 1
    DESCRIPTION = 2
    CREATION_TIME = 4
    LAST_UPDATED_TIME = 5


class ModelOrderBy(enum.Enum):
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

    def get_version(self, version: int = 0) -> Checkpoint:
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
            latest_version = data["versions"][0]
            return Checkpoint.from_json(
                {
                    **latest_version["checkpoint"],
                    "version": latest_version["version"],
                    "model_name": data["model"]["name"],
                }
            )
        else:
            resp = api.get(self._master, "/api/v1/models/{}/versions/{}".format(self.name, version))

        data = resp.json()
        return Checkpoint.from_json(data["version"]["checkpoint"], self._master)

    def get_versions(self, order_by: ModelOrderBy = ModelOrderBy.DESC) -> List[Checkpoint]:
        """
        Get a list of checkpoints corresponding to versions of this model. The
        models are sorted by version number and are returned in descending
        order by default.

        Arguments:
            order_by (enum): A member of the ModelOrderBy enum.
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
                    "version": version["version"],
                    "model_name": data["model"]["name"],
                },
                self._master,
            )
            for version in data["versions"]
        ]

    def register_version(self, checkpoint_uuid: str) -> Checkpoint:
        """
        Creats a new model version and returns the
        :class:`~determined.experimental.Checkpoint` corresponding to the
        version.

        Arguments:
            checkpoint_uuid: The uuid to associated with the new model version.
        """
        resp = api.post(
            self._master,
            "/api/v1/models/{}/versions".format(self.name),
            body={"checkpoint_uuid": checkpoint_uuid},
        )

        data = resp.json()

        return Checkpoint.from_json(
            {
                **data["version"]["checkpoint"],
                "version": data["version"]["version"],
                "model_name": data["version"]["model"]["name"],
            },
            self._master,
        )

    def add_metadata(self, metadata: Dict[str, Any]) -> None:
        """
        Adds user-defined metadata to the model. The ``metadata`` argument must be a
        JSON-serializable dictionary. If any keys from this dictionary already appear in
        the model metadata, the corresponding dictionary entries in the model are
        replaced by the passed-in dictionary values.

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
