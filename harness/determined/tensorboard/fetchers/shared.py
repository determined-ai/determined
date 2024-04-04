import datetime
import logging
import os
import posixpath
import shutil
from typing import Any, Callable, Dict, Generator, List

from determined.tensorboard.fetchers import base

logger = logging.getLogger("determined.tensorboard.shared")


class SharedFSFetcher(base.Fetcher):
    def __init__(self, storage_config: Dict[str, Any], storage_paths: List[str], local_dir: str):
        """Fetch tensorboard events files from storage and save to local directory"""
        _ = storage_config
        self.local_dir = local_dir
        self.storage_paths = storage_paths
        self._file_records = {}  # type: Dict[str, datetime.datetime]

    def _list(self, storage_path: str) -> Generator[str, None, None]:
        logger.debug(f"Finding files in storage_path: '{storage_path}'")

        for root, _, files in os.walk(storage_path):
            for file in files:
                filepath = posixpath.join(root, file)
                mtime = os.path.getmtime(filepath)
                prev_mdatetime = self._file_records.get(filepath)
                mdatetime = datetime.datetime.fromtimestamp(mtime)
                if prev_mdatetime is not None and prev_mdatetime >= mdatetime:
                    continue
                self._file_records[filepath] = mdatetime
                yield filepath

    def _fetch(self, filepath: str, new_file_callback: Callable) -> None:
        local_path = posixpath.join(self.local_dir, filepath.lstrip("/"))

        dir_path = os.path.dirname(local_path)
        os.makedirs(dir_path, exist_ok=True)

        shutil.copyfile(filepath, local_path)

        logger.debug(f"Transfered '{filepath}' to '{local_path}'")
        new_file_callback()
