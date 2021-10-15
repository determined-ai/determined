import contextlib
import logging
import pathlib
from typing import Iterator

import determined as det
from determined.common import constants, storage


class Context:
    """
    generic.Context will someday evolve into a core part of the Generic API.
    """

    def __init__(
        self,
        env: det.EnvContext,
        dist: det.DistributedContext,
    ) -> None:
        self._env = env
        self._dist = dist

        self._storage_mgr = storage.build(
            env.experiment_config["checkpoint_storage"],
            container_path=None if not env.on_cluster else constants.SHARED_FS_CONTAINER_PATH,
        )

    @contextlib.contextmanager
    def _download_initial_checkpoint(self, storage_id: str) -> Iterator[pathlib.Path]:
        """
        Wrap a storage_mgr.restore_path() context manager, but only download/cleanup on the
        local chief.
        """

        logging.info("Restoring trial from checkpoint {}".format(storage_id))

        restore_path = self._dist._local_chief_contextmanager(self._storage_mgr.restore_path)
        with restore_path(storage_id) as path:
            yield path
