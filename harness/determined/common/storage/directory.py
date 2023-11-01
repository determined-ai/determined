import contextlib
import logging
import os
import pathlib
from typing import Any, Dict, Iterator, Optional

from determined import errors
from determined.common import check, storage

logger = logging.getLogger("determined.common.storage.directory")


class DirectoryStorageManager(storage.SharedFSStorageManager):
    """
    Storage and load checkpoints from a predefined path available to the task runtime.
    """

    @classmethod
    def from_config(
        cls,
        config: Dict[str, Any],
        container_path: Optional[str] = None,
    ) -> "DirectoryStorageManager":
        # `container_path`` argument here is ignored: it's specific to the way `shared_fs`
        # discovers the storage location.
        # `directory` storage will always use the directory provided in the config.
        allowed_keys = {"container_path"}
        for key in config.keys():
            check.is_in(key, allowed_keys, "extra key in shared_fs config")

        check.is_in(
            "container_path", config, "directory checkpoint config is missing container_path"
        )

        base_path = config["container_path"]

        return cls(base_path)

    # This method needs an override for the better error and warning messaging.
    @contextlib.contextmanager
    def restore_path(
        self, src: str, selector: Optional[storage.Selector] = None
    ) -> Iterator[pathlib.Path]:
        """
        Prepare a local directory exposing the checkpoint. Do some simple checks to make sure the
        configuration seems reasonable.
        """
        if selector is not None:
            logger.warning(
                "Ignoring partial checkpoint download from 'directory' checkpoint storage; "
                "all files will be directly accessible."
            )
        check.true(
            os.path.exists(self._base_path),
            f"Storage directory does not exist: {self._base_path}. Please verify that you are "
            "using the correct configuration value for checkpoint_storage.container_path",
        )
        storage_dir = os.path.join(self._base_path, src)
        if not os.path.exists(storage_dir):
            raise errors.CheckpointNotFound(f"Did not find checkpoint {src} in 'directory' storage")
        yield pathlib.Path(storage_dir)
