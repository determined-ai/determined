import os

import analytics


def send_analytics(tracking_key: str) -> None:
    analytics.write_key = os.environ.get("DET_SEGMENT_API_KEY")
    if os.environ.get("DET_SEGMENT_ENABLED"):
        analytics.track(os.environ.get("DET_CLUSTER_ID"), tracking_key)
