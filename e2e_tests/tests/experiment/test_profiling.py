import json
import tempfile
from typing import Any, Dict, Sequence

import pytest

from determined.common import api, util
from determined.common.api import bindings
from tests import api_utils
from tests import config as conf
from tests import experiment as exp


@pytest.mark.e2e_gpu
@pytest.mark.timeout(30 * 60)
def test_streaming_observability_metrics_apis() -> None:
    sess = api_utils.user_session()
    config_path = conf.fixtures_path("mnist_pytorch/const-profiling.yaml")

    config_obj = conf.load_config(config_path)
    with tempfile.NamedTemporaryFile() as tf:
        with open(tf.name, "w") as f:
            util.yaml_safe_dump(config_obj, f)
        experiment_id = exp.create_experiment(sess, tf.name, conf.fixtures_path("mnist_pytorch"))

    exp.wait_for_experiment_state(sess, experiment_id, bindings.experimentv1State.COMPLETED)
    trials = exp.experiment_trials(sess, experiment_id)
    trial_id = trials[0].trial.id

    gpu_enabled = conf.GPU_ENABLED

    labels = request_profiling_metric_labels(sess, trial_id, gpu_enabled)
    if gpu_enabled:
        gpu_labels = [label for label in labels if label["name"] == "gpu_util"]
        assert len(gpu_labels) >= 1, "expected at least 1 GPU label"
        request_profiling_system_metrics(sess, trial_id, gpu_labels[0])


def request_profiling_metric_labels(
    sess: api.Session, trial_id: int, gpu_enabled: bool
) -> Sequence[Dict[str, Any]]:
    def validate_labels(labels: Sequence[Dict[str, Any]]) -> None:
        # Check some labels against the expected labels. Return the missing labels.
        expected = {
            "cpu_util_simple": PROFILER_METRIC_TYPE_SYSTEM,
            "disk_iops": PROFILER_METRIC_TYPE_SYSTEM,
            "disk_throughput_read": PROFILER_METRIC_TYPE_SYSTEM,
            "disk_throughput_write": PROFILER_METRIC_TYPE_SYSTEM,
            "memory_free": PROFILER_METRIC_TYPE_SYSTEM,
            "net_throughput_recv": PROFILER_METRIC_TYPE_SYSTEM,
            "net_throughput_sent": PROFILER_METRIC_TYPE_SYSTEM,
        }

        if gpu_enabled:
            expected.update(
                {
                    "gpu_free_memory": PROFILER_METRIC_TYPE_SYSTEM,
                    "gpu_util": PROFILER_METRIC_TYPE_SYSTEM,
                }
            )

        for label in labels:
            metric_name = label["name"]
            metric_type = label["metricType"]
            if expected.get(metric_name, None) == metric_type:
                del expected[metric_name]

        if len(expected) > 0:
            pytest.fail(
                f"expected completed experiment to have all labels but some are missing: {expected}"
            )

    with sess.get(
        f"api/v1/trials/{trial_id}/profiler/available_series",
        stream=True,
    ) as r:
        for line in r.iter_lines():
            labels: Sequence[Dict[str, Any]] = json.loads(line)["result"]["labels"]
            validate_labels(labels)
            # Just check 1 iter.
            return labels
    return []


def request_profiling_system_metrics(
    sess: api.Session, trial_id: int, labels: Dict[str, Any]
) -> None:
    def validate_gpu_metric_batch(batch: Dict[str, Any]) -> None:
        num_values = len(batch["values"])
        num_timestamps = len(batch["timestamps"])
        if not (num_values == num_timestamps):
            pytest.fail(f"mismatched lists: not ({num_values} == {num_timestamps})")

        if num_values == 0:
            pytest.fail(f"received batch of size 0, something went wrong: {batch}")

    with sess.get(
        f"api/v1/trials/{trial_id}/profiler/metrics",
        params={
            "labels.name": labels["name"],
            "labels.agentId": labels["agentId"],
            "labels.metricType": PROFILER_METRIC_TYPE_SYSTEM,
        },
        stream=True,
    ) as r:
        have_batch = False
        for line in r.iter_lines():
            batch = json.loads(line)["result"]["batch"]
            validate_gpu_metric_batch(batch)
            have_batch = True
        if not have_batch:
            pytest.fail("no batch metrics at all")


PROFILER_METRIC_TYPE_SYSTEM = "PROFILER_METRIC_TYPE_SYSTEM"
