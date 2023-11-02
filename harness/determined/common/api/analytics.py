import logging
import os
import sys
from typing import Any, Dict, Type

import analytics

logger = logging.getLogger("determined.common")

enabled = os.environ.get("DET_SEGMENT_ENABLED") == "true"
_cluster_id = os.environ.get("DET_CLUSTER_ID")

# Segment.io Configuration
# https://github.com/segmentio/analytics-python/blob/5d87a9085c18ee25660c8196dd1233100f9b61b6/segment/analytics/__init__.py
analytics.write_key = os.environ.get("DET_SEGMENT_API_KEY")
analytics.max_retries = 5


def get_library_version_analytics() -> Dict[str, Any]:
    modules = [
        "determined",
        "model_hub",
        "torch",
        "tensorflow",
        "transformers",
        "mmcv",
        "mmdet",
    ]
    versions = {}
    for m in sys.modules:
        if m in modules:
            try:
                versions[m] = sys.modules[m].__version__  # type: ignore
            except Exception:
                pass
    return versions


def get_trial_analytics(obj: Type) -> Dict[str, Any]:
    ancestors = ",".join(each.__name__ for each in obj.mro())
    return {
        "library_version": get_library_version_analytics(),
        "trial_name": obj.__name__,
        "trial_ancestors": ancestors,
    }


def send_analytics(event: str, properties: Dict) -> None:
    if enabled and _cluster_id is not None and analytics.write_key is not None:
        logger.debug(f"Sending analytics event {event}: {properties}.")
        analytics.track(_cluster_id, event, properties)
