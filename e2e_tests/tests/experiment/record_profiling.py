from statistics import mean
from typing import Callable, Dict, List, Optional, Tuple
from urllib.parse import urlencode

import simplejson

from determined.common import api
from determined.profiler import SysMetricName
from tests import config as conf

summary_methods: Dict[str, Callable] = {"avg": mean, "max": max, "min": min}

default_metrics: Dict[str, List[str]] = {
    SysMetricName.GPU_UTIL_METRIC: ["avg", "max"],
    SysMetricName.SIMPLE_CPU_UTIL_METRIC: ["avg", "max"],
    SysMetricName.DISK_IOPS_METRIC: ["avg", "max"],
    SysMetricName.DISK_THRU_READ_METRIC: ["avg", "max"],
    SysMetricName.DISK_THRU_WRITE_METRIC: ["avg", "max"],
    SysMetricName.NET_THRU_SENT_METRIC: ["avg"],
    SysMetricName.NET_THRU_RECV_METRIC: ["avg"],
}


def profile_test(
    record_property: Callable[[str, object], None],
    profiled_metrics: Optional[Dict[str, List[str]]] = None,
) -> Callable[[int], None]:
    if not profiled_metrics:
        profiled_metrics = default_metrics

    def record(trial_id: int) -> None:
        assert profiled_metrics is not None
        for metric in profiled_metrics:
            metrics = get_profiling_metrics(trial_id, metric)
            if not metrics:
                print(f"No {metric} metrics collected")
                continue

            for method in profiled_metrics[metric]:
                metric_key, metric_value = format_xml_property(
                    metric, method, summary_methods[method](metrics)
                )
                record_property(metric_key, metric_value)

    return record


def format_xml_property(
    metric_type: str, summary_method: str, metric_value: float
) -> Tuple[str, float]:
    """
    Formats metric summary into XML name-value tuple in the form of
    (metric_type[summary_method], metric_value)
    ex: (cpu_util[avg], 88.23)
    """
    return f"{metric_type}[{summary_method}]", metric_value


def get_profiling_metrics(trial_id: int, metric_type: str) -> List[float]:
    """
    Calls profiler API to return a list of metric values given trial ID and metric type
    """
    with api.get(
        conf.make_master_url(),
        "api/v1/trials/{}/profiler/metrics?{}".format(
            trial_id,
            urlencode(
                {
                    "labels.name": metric_type,
                    "labels.metricType": "PROFILER_METRIC_TYPE_SYSTEM",
                    "follow": "true",
                }
            ),
        ),
        stream=True,
    ) as r:
        return [
            batch
            for batches in [
                simplejson.loads(line)["result"]["batch"]["values"] for line in r.iter_lines()
            ]
            for batch in batches
        ]
