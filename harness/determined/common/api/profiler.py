import time
from typing import Any, Dict, List, Optional

from requests import exceptions

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
    backoff_interval = 1
    max_tries = 2
    tries = 0

    while tries < max_tries:
        try:
            api.post(
                master_url,
                "/api/v1/trials/profiler/metrics",
                json={"batches": [b.__dict__ for b in batches]},
            )
            return
        except exceptions.RequestException as e:
            if e.response is not None and e.response.status_code < 500:
                raise e

            tries += 1
            if tries == max_tries:
                raise e
            time.sleep(backoff_interval)
    return


class TrialProfilerSeriesLabels:
    def __init__(self, trial_id: int, name: str, agent_id: str, gpu_uuid: str, metric_type: str):
        self.trial_id = str(trial_id)
        self.name = name
        self.agent_id = agent_id
        self.gpu_uuid = gpu_uuid if gpu_uuid != "" else None  # type: Optional[str]
        self.metric_type = metric_type


def get_trial_profiler_available_series(
    master_url: str,
    trial_id: str,
) -> List[TrialProfilerSeriesLabels]:
    """
    Get available profiler series for a trial. This uses the non-streaming version of the API
    """
    follow = False
    backoff_interval = 1
    max_tries = 2
    tries = 0

    response = None
    while tries < max_tries:
        try:
            response = api.get(
                host=master_url,
                path=f"/api/v1/trials/{trial_id}/profiler/available_series",
                params={"follow": follow},
            )
            break
        except exceptions.RequestException as e:
            if e.response is not None and e.response.status_code < 500:
                raise e

            tries += 1
            if tries == max_tries:
                raise e
            time.sleep(backoff_interval)

    assert response
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
