import logging
import os
from typing import Any

import analytics


def send_analytics(tracking_key: str) -> None:
    if os.environ.get("DET_SEGMENT_ENABLED"):
        analytics.write_key = os.environ.get("DET_SEGMENT_API_KEY")
        analytics.track(os.environ.get("DET_CLUSTER_ID"), tracking_key)


def on_error(error: Any, items: Any) -> None:
    logging.warning(f"Analytics tracking received error: {error}")
