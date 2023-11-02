from typing import Any, Dict, List, Type

from .azure import AzureFetcher
from .base import Fetcher
from .gcs import GCSFetcher
from .s3 import S3Fetcher
from .shared import SharedFSFetcher
from .directory import DirectoryFetcher

__all__ = [
    "S3Fetcher",
    "GCSFetcher",
    "AzureFetcher",
    "SharedFSFetcher",
]

_FETCHERS = {
    "s3": S3Fetcher,
    "gcs": GCSFetcher,
    "azure": AzureFetcher,
    "shared_fs": SharedFSFetcher,
    "directory": DirectoryFetcher,
}  # type: Dict[str, Type[Fetcher]]


def build(storage_config: Dict[str, Any], paths: List[str], local_dir: str) -> Fetcher:
    storage_type = storage_config.get("type")
    if storage_type not in _FETCHERS:
        raise ValueError(f"checkpoint_storage type '{storage_type}' is not supported")

    return _FETCHERS[storage_type](storage_config, paths, local_dir)
