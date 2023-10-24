import json
import os
import tempfile
from typing import Any, Dict, Optional, Sequence
from urllib.parse import urlencode

import pytest

from determined.common import api, yaml
from determined.common.api import authentication, bindings, certs
from tests import config as conf
from tests import experiment as exp


@pytest.mark.e2e_gpu
@pytest.mark.timeout(30 * 60)
@pytest.mark.parametrize(
    "model_def,timings_enabled",
    [
        (conf.tutorials_path("mnist_pytorch"), True),
    ],
)
def test_streaming_observability_metrics_apis(model_def: str, timings_enabled: bool) -> None:
    # TODO: refactor tests to not use cli singleton auth.
    certs.cli_cert = certs.default_load(conf.make_master_url())
    authentication.cli_auth = authentication.Authentication(conf.make_master_url())

    config_path = os.path.join(model_def, "const.yaml")

    config_obj = conf.load_config(config_path)
    config_obj = conf.set_profiling_enabled(config_obj)
    with tempfile.NamedTemporaryFile() as tf:
        with open(tf.name, "w") as f:
            yaml.dump(config_obj, f)
        experiment_id = exp.create_experiment(tf.name, model_def)

    exp.wait_for_experiment_state(experiment_id, bindings.experimentv1State.COMPLETED)
    trials = exp.experiment_trials(experiment_id)
    trial_id = trials[0].trial.id

    gpu_enabled = conf.GPU_ENABLED

    request_profiling_metric_labels(trial_id, timings_enabled, gpu_enabled)
    if gpu_enabled:
        request_profiling_system_metrics(trial_id, "gpu_util")
    if timings_enabled:
        request_profiling_pytorch_timing_metrics(trial_id, "train_batch")
        request_profiling_pytorch_timing_metrics(trial_id, "train_batch.backward", accumulated=True)


def request_profiling_metric_labels(trial_id: int, timing_enabled: bool, gpu_enabled: bool) -> None:
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

    with api.get(
        conf.make_master_url(),
        "api/v1/trials/{}/profiler/available_series".format(trial_id),
        stream=True,
    ) as r:
        for line in r.iter_lines():
            labels = json.loads(line)["result"]["labels"]
            validate_labels(labels)
            # Just check 1 iter.
            return


def request_profiling_system_metrics(trial_id: int, metric_name: str) -> None:
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

    with api.get(
        conf.make_master_url(),
        "api/v1/trials/{}/profiler/metrics?{}".format(
            trial_id,
            to_query_params(PROFILER_METRIC_TYPE_SYSTEM, metric_name),
        ),
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
    trial_id: int, metric_name: str, accumulated: bool = False
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

    with api.get(
        conf.make_master_url(),
        "api/v1/trials/{}/profiler/metrics?{}".format(
            trial_id,
            to_query_params(PROFILER_METRIC_TYPE_TIMING, metric_name),
        ),
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


def to_query_params(metric_type: str, metric_name: Optional[str] = None) -> str:
    return urlencode(
        {
            "labels.name": metric_name,
            "labels.metricType": metric_type,
        }
    )
