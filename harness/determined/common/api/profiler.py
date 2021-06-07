from typing import Any, Dict, List, Optional

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


class TrialProfilerSeriesLabels:
    def __init__(self, trial_id: int, name: str, agent_id: str, gpu_uuid: str, metric_type: str):
        self.trial_id = str(trial_id)
        self.name = name
        self.agent_id = agent_id
        self.gpu_uuid = gpu_uuid if gpu_uuid != "" else None  # type: Optional[str]
        self.metric_type = metric_type


@backoff.on_exception(  # type: ignore
    backoff.constant,
    RequestException,
    max_tries=2,
    giveup=lambda e: e.response is not None and e.response.status_code < 500,
)
def get_trial_profiler_available_series(
    master_url: str,
    trial_id: str,
) -> List[TrialProfilerSeriesLabels]:
    """
    Get available profiler series for a trial. This uses the non-streaming version of the API
    """
    follow = False
    response = api.get(
        host=master_url,
        path=f"/api/v1/trials/{trial_id}/profiler/available_series",
        params={"follow": follow},
    )
    j = response.json()
    labels = [
        TrialProfilerSeriesLabels(
            trial_id=ld["trialId"],
            name=ld["name"],
            agent_id=ld["agentId"],
            gpu_uuid=ld["gpuUuid"],
            metric_type=ld["metricType"],
        )
        for ld in j["result"]["labels"]
    ]
    return labels
