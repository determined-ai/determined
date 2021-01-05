import logging
import pathlib
from typing import Optional

from determined import workload
from determined_common import storage


class StorageLayer(workload.Source):
    """StorageLayer coordinates preparing and uploading checkpoints."""

    def __init__(
        self,
        workloads: workload.Stream,
        storage_mgr: storage.StorageManager,
        is_chief: bool,
    ) -> None:
        self.workloads = workloads
        self.storage_mgr = storage_mgr
        self.is_chief = is_chief

    def __iter__(self) -> workload.Stream:
        for wkld, args, response_func in self.workloads:
            # This layer only cares about checkpoints.
            if wkld.kind != workload.Workload.Kind.CHECKPOINT_MODEL:
                yield wkld, args, response_func
                continue

            # Only the chief container should checkpoint.
            if not self.is_chief:
                response_func(workload.Skipped())
                continue

            # Save the workload completed message for after checkpoint upload completes.
            message = None  # type: Optional[workload.Response]

            def _respond(checkpoint_info: workload.Response) -> None:
                assert isinstance(checkpoint_info, dict)
                metadata = storage.StorageMetadata(
                    storage_id,
                    storage.StorageManager._list_directory(path),
                    checkpoint_info.get("framework", ""),
                    checkpoint_info.get("format", ""),
                )

                logging.info("Saved trial to checkpoint {}".format(metadata.storage_id))

                nonlocal message
                message = {
                    "type": "WORKLOAD_COMPLETED",
                    "workload": wkld,
                    "metrics": metadata,
                }

            with self.storage_mgr.store_path() as (storage_id, path):
                assert not args, "CHECKPOINT_MODEL args should be empty!"
                yield wkld, [pathlib.Path(path)], _respond

            # Because the messaging is synchronous, the layer below us must have called _respond.
            if message is None:
                raise AssertionError("response function did not get called")

            response_func(message)
