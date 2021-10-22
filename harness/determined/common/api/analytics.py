import os

import analytics

analytics.write_key = os.environ.get("DET_SEGMENT_API_KEY")
_cluster_id = os.environ.get("DET_CLUSTER_ID")

if (
    os.environ.get("DET_SEGMENT_ENABLED") == "true"
    and _cluster_id is not None
    and analytics.write_key is not None
):

    def send_analytics(tracking_key: str) -> None:
        analytics.track(_cluster_id, tracking_key)


else:

    def send_analytics(tracking_key: str) -> None:
        pass
