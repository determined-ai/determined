import logging
from typing import Any, Dict, Optional

import determined as det
from determined import tensorboard
from determined.common.experimental.session import Session

logger = logging.getLogger("determined.generic")


class Checkpointing:
    """
    Some checkpoint-related REST API wrappers.
    """

    def __init__(
        self,
        session: Session,
        api_path: str,
        static_metadata: Optional[Dict[str, Any]] = None,
        tbd_mgr: Optional[tensorboard.TensorboardManager] = None,
    ) -> None:
        self._session = session
        self._static_metadata = static_metadata or {}
        self._static_metadata["determined_version"] = det.__version__
        self._api_path = api_path
        self._tbd_mgr = tbd_mgr

    def _report_checkpoint(
        self,
        uuid: str,
        resources: Optional[Dict[str, int]] = None,
        metadata: Optional[Dict[str, Any]] = None,
    ) -> None:
        """
        After having uploaded a checkpoint, report its existence to the master.

        This is still a private function because it might never be offered directly to the user;
        they might always go through a StorageManager.
        """
        resources = resources or {}
        metadata = metadata or {}
        required = {"framework", "format", "latest_batch"}
        allowed = required.union({"total_records", "total_epochs"})
        missing = [k for k in required if k not in metadata]
        extra = [k for k in metadata.keys() if k not in allowed]
        if missing:
            raise ValueError(
                "metadata for reported checkpoints, in the current implementation, requires all of "
                f"the following items that have not been provided: {missing}"
            )
        if extra:
            raise ValueError(
                "metadata for reported checkpoints, in the current implementation, cannot support "
                f"the following items that were provided: {extra}"
            )

        body = {
            "uuid": uuid,
            "resources": resources,
            **self._static_metadata,
            **metadata,
        }
        logger.debug(f"_report_checkpoint({uuid})")
        self._session.post(self._api_path, data=det.util.json_encode(body))

        # Also sync tensorboard.
        if self._tbd_mgr:
            self._tbd_mgr.sync()
