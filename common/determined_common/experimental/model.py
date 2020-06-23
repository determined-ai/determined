import datetime
import enum
from typing import Any, Dict, List, Optional

from determined_common import api


class ModelSortBy(enum.Enum):
    UNSPECIFIED = 0
    NAME = 1
    DESCRIPTION = 2
    CREATION_TIME = 4
    LAST_UPDATED_TIME = 5


class ModelOrderBy(enum.Enum):
    ASCENDING = 1
    DESCENDING = 2


class Model:
    """
    Class representing a model. Contains methods for managing metadata.
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
            body={
                "model": {"metadata": self.metadata},
                "update_mask": {"paths": ["model.metadata"]},
            },
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
            body={
                "model": {"metadata": self.metadata},
                "update_mask": {"paths": ["model.metadata"]},
            },
        )

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
