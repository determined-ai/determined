import logging
import os
import sys
from typing import Any, Dict, Type

import analytics

enabled = os.environ.get("DET_SEGMENT_ENABLED") == "true"
analytics.write_key = os.environ.get("DET_SEGMENT_API_KEY")
_cluster_id = os.environ.get("DET_CLUSTER_ID")


def get_library_version_analytics() -> Dict[str, Any]:
    modules = [
        "determined",
        "model_hub",
        "torch",
        "pytorch_lightning",
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
        logging.debug(f"Sending analytics event {event}: {properties}.")
        analytics.track(_cluster_id, event, properties)
