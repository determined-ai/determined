from typing import Dict, List

from determined.common import api


def post_trial_profiler_metrics(
    master_url: str,
    values: List[float],
    batches: List[int],
    timestamps: List[str],
    labels: Dict[str, str],
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
        body={
            "batch": {
                "values": values,
                "batches": batches,
                "timestamps": timestamps,
                "labels": labels,
            }
        },
    )