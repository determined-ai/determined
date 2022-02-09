import datetime
import logging
import os
import posixpath
import shutil
from typing import Any, Dict, Generator, List, Tuple

from .base import Fetcher

logger = logging.getLogger(__name__)


class SharedFSFetcher(Fetcher):
    def __init__(self, storage_config: Dict[str, Any], storage_paths: List[str], local_dir: str):
        """Fetch tensorboard events files from storage and save to local directory"""
        _ = storage_config
        self.local_dir = local_dir
        self.storage_paths = storage_paths
        self._file_records = {}  # type: Dict[str, datetime.datetime]

    def _list(self, log_dir: str) -> Generator[Tuple[str, datetime.datetime], None, None]:
        logger.debug(f"Finding files in log directory '{log_dir}'")

        for root, _, files in os.walk(log_dir):
            for file in files:
                filepath = posixpath.join(root, file)
                mtime = os.path.getmtime(filepath)
                yield (filepath, datetime.datetime.fromtimestamp(mtime))

    def fetch_new(self) -> int:
        new_files = []

        # Look at all files in our storage location.
        for storage_path in self.storage_paths:
            for filepath, mdatetime in self._list(storage_path):
                prev_mdatetime = self._file_records.get(filepath)

                if prev_mdatetime is not None and prev_mdatetime >= mdatetime:
                    continue

                new_files.append(filepath)
                self._file_records[filepath] = mdatetime

        # Download the new or updated files.
        for filepath in new_files:
            local_path = posixpath.join(self.local_dir, filepath.lstrip("/"))

            dir_path = os.path.dirname(local_path)
            os.makedirs(dir_path, exist_ok=True)

            shutil.copyfile(filepath, local_path)

            logger.debug(f"Transfered '{filepath}' to '{local_path}'")

        return len(new_files)
