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
@pytest.mark.e2e_slurm_gpu
@pytest.mark.timeout(30 * 60)
@pytest.mark.parametrize(
    "model_def,timings_enabled",
    [
        (conf.fixtures_path("mnist_pytorch"), True),
    ],
)
def test_streaming_observability_metrics_apis(model_def: str, timings_enabled: bool) -> None:
    sess = api_utils.user_session()
    config_path = conf.fixtures_path("mnist_pytorch/const-profiling.yaml")

    config_obj = conf.load_config(config_path)
    with tempfile.NamedTemporaryFile() as tf:
        with open(tf.name, "w") as f:
            util.yaml_safe_dump(config_obj, f)
        experiment_id = exp.create_experiment(sess, tf.name, model_def)

    exp.wait_for_experiment_state(sess, experiment_id, bindings.experimentv1State.COMPLETED)
    trials = exp.experiment_trials(sess, experiment_id)
    trial_id = trials[0].trial.id

    gpu_enabled = conf.GPU_ENABLED

    request_profiling_metric_labels(sess, trial_id, timings_enabled, gpu_enabled)
    if gpu_enabled:
        request_profiling_system_metrics(sess, trial_id, "gpu_util")
    if timings_enabled:
        request_profiling_pytorch_timing_metrics(sess, trial_id, "train_batch")
        request_profiling_pytorch_timing_metrics(
            sess, trial_id, "train_batch.backward", accumulated=True
        )


def request_profiling_metric_labels(
    sess: api.Session, trial_id: int, timing_enabled: bool, gpu_enabled: bool
) -> None:
    def validate_labels(labels: Sequence[Dict[str, Any]]) -> None:
        # Check some labels against the expected labels. Return the missing labels.
        expected = {
            "cpu_util_simple": PROFILER_METRIC_TYPE_SYSTEM,
            "dataloader_next": PROFILER_METRIC_TYPE_TIMING,
            "disk_iops": PROFILER_METRIC_TYPE_SYSTEM,
            "disk_throughput_read": PROFILER_METRIC_TYPE_SYSTEM,
            "disk_throughput_write": PROFILER_METRIC_TYPE_SYSTEM,
            "free_memory": PROFILER_METRIC_TYPE_SYSTEM,
            "from_device": PROFILER_METRIC_TYPE_TIMING,
            "net_throughput_recv": PROFILER_METRIC_TYPE_SYSTEM,
            "net_throughput_sent": PROFILER_METRIC_TYPE_SYSTEM,
            "reduce_metrics": PROFILER_METRIC_TYPE_TIMING,
            "step_lr_schedulers": PROFILER_METRIC_TYPE_TIMING,
            "to_device": PROFILER_METRIC_TYPE_TIMING,
            "train_batch": PROFILER_METRIC_TYPE_TIMING,
        }

        if gpu_enabled:
            expected.update(
                {
                    "gpu_free_memory": PROFILER_METRIC_TYPE_SYSTEM,
                    "gpu_util": PROFILER_METRIC_TYPE_SYSTEM,
                }
            )
        if not timing_enabled:
            expected = {k: v for k, v in expected.items() if v != PROFILER_METRIC_TYPE_TIMING}
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
            labels = json.loads(line)["result"]["labels"]
            validate_labels(labels)
            # Just check 1 iter.
            return


def request_profiling_system_metrics(sess: api.Session, trial_id: int, metric_name: str) -> None:
    def validate_gpu_metric_batch(batch: Dict[str, Any]) -> None:
        num_values = len(batch["values"])
        num_batch_indexes = len(batch["batches"])
        num_timestamps = len(batch["timestamps"])
        if not (num_values == num_batch_indexes == num_timestamps):
            pytest.fail(
                f"mismatched lists: not ({num_values} == {num_batch_indexes} == {num_timestamps})"
            )

        if num_values == 0:
            pytest.fail(f"received batch of size 0, something went wrong: {batch}")

    with sess.get(
        f"api/v1/trials/{trial_id}/profiler/metrics",
        params={
            "labels.name": metric_name,
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


def request_profiling_pytorch_timing_metrics(
    sess: api.Session, trial_id: int, metric_name: str, accumulated: bool = False
) -> None:
    def validate_timing_batch(batch: Dict[str, Any], batch_idx: int) -> int:
        values = batch["values"]
        batches = batch["batches"]
        num_values = len(values)
        num_batch_indexes = len(batches)
        num_timestamps = len(batch["timestamps"])
        if num_values != num_batch_indexes or num_batch_indexes != num_timestamps:
            pytest.fail(
                f"mismatched slices: not ({num_values} == {num_batch_indexes} == {num_timestamps})"
            )

        if not any(values):
            pytest.fail(f"received bad batch, something went wrong: {batch}")

        if batches[0] != batch_idx:
            pytest.fail(
                f"batch did not start at correct batch, {batches[0]} != {batch_idx}: {batch}"
            )

        # Check batches are monotonic with no gaps.
        if not all(x + 1 == y for x, y in zip(batches, batches[1:])):
            pytest.fail(f"skips in batches sampled: {batch}")

        # 10 is just a threshold at which it would be really strange for a batch to be monotonic.
        if accumulated and len(values) > 10 and all(x < y for x, y in zip(values, values[1:])):
            pytest.fail(
                f"per batch accumulated metric was monotonic, which is really fishy: {batch}"
            )

        return int(batches[-1]) + 1

    with sess.get(
        f"api/v1/trials/{trial_id}/profiler/metrics",
        params={
            "labels.name": metric_name,
            "labels.metricType": PROFILER_METRIC_TYPE_TIMING,
        },
        stream=True,
    ) as r:
        batch_idx = 0
        have_batch = False
        for line in r.iter_lines():
            batch = json.loads(line)["result"]["batch"]
            batch_idx = validate_timing_batch(batch, batch_idx)
            have_batch = True
        if not have_batch:
            pytest.fail("no batch metrics at all")


PROFILER_METRIC_TYPE_SYSTEM = "PROFILER_METRIC_TYPE_SYSTEM"
PROFILER_METRIC_TYPE_TIMING = "PROFILER_METRIC_TYPE_TIMING"
