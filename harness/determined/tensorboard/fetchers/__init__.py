from typing import Any, Dict, List, Optional, Type

from .azure import AzureFetcher
from .base import Fetcher
from .gcs import GCSFetcher
from .s3 import S3Fetcher
from .shared import SharedFSFetcher

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
}  # type: Dict[str, Type[Fetcher]]


def build(config: Dict[str, Any], paths: List[str], local_dir: str) -> Fetcher:
    storage_config = config.get("checkpoint_storage")
    if storage_config is None:
        raise ValueError("config does not contain a 'checkpoint_storage' key")

    storage_type = storage_config.get("type")
    if storage_type not in _FETCHERS:
        raise ValueError(f"checkpoint_storage type '{storage_type}' is not supported")

    return _FETCHERS[storage_type](storage_config, paths, local_dir)
