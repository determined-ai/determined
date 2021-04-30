from typing import Any, Dict, List

import backoff
from requests.exceptions import RequestException

from determined.common import api


class TrialProfilerMetricsBatch:
    """
    TrialProfilerMetricsBatch is the representation of a batch of trial
    profiler metrics as accepted by POST /api/v1/trials/:trial_id/profiler/metrics
    """

    def __init__(
        self,
        values: List[float],
        batches: List[int],
        timestamps: List[str],
        labels: Dict[str, Any],
    ):
        self.values = values
        self.batches = batches
        self.timestamps = timestamps
        self.labels = labels


@backoff.on_exception(  # type: ignore
    backoff.constant,
    RequestException,
    max_tries=2,
    giveup=lambda e: e.response is not None and e.response.status_code < 500,
)
def post_trial_profiler_metrics_batches(
    master_url: str,
    batches: List[TrialProfilerMetricsBatch],
) -> None:
    """
    Post the given metrics to the master to be persisted. Labels
    must contain only a subset of the keys: trial_id,  name,
    gpu_uuid, agent_id and metric_type, where metric_type is one
    of PROFILER_METRIC_TYPE_SYSTEM or PROFILER_METRIC_TYPE_TIMING.
    """
    api.post(
        master_url,
        "/api/v1/trials/profiler/metrics",
        body={"batches": [b.__dict__ for b in batches]},
    )
