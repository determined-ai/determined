import json
import multiprocessing as mp
import subprocess
from typing import Dict, List, Set, Union

import pytest

from determined.common import api
from determined.common.api import authentication, bindings, certs
from tests import api_utils
from tests import config as conf
from tests import experiment as exp


@pytest.mark.e2e_cpu
@pytest.mark.timeout(600)
def test_streaming_metrics_api() -> None:
    # TODO: refactor tests to not use cli singleton auth.
    certs.cli_cert = certs.default_load(conf.make_master_url())
    authentication.cli_auth = authentication.Authentication(conf.make_master_url())

    pool = mp.pool.ThreadPool(processes=7)

    experiment_id = exp.create_experiment(
        conf.fixtures_path("mnist_pytorch/adaptive_short.yaml"),
        conf.tutorials_path("mnist_pytorch"),
    )
    # To fully test the streaming APIs, the requests need to start running immediately after the
    # experiment, and then stay open until the experiment is complete. To accomplish this with all
    # of the API calls on a single experiment, we spawn them all in threads.

    # The HP importance portion of this test is commented out until the feature is enabled by
    # default

    metric_names_thread = pool.apply_async(request_metric_names, (experiment_id,))
    train_metric_batches_thread = pool.apply_async(request_train_metric_batches, (experiment_id,))
    valid_metric_batches_thread = pool.apply_async(request_valid_metric_batches, (experiment_id,))
    train_trials_snapshot_thread = pool.apply_async(request_train_trials_snapshot, (experiment_id,))
    valid_trials_snapshot_thread = pool.apply_async(request_valid_trials_snapshot, (experiment_id,))
    train_trials_sample_thread = pool.apply_async(request_train_trials_sample, (experiment_id,))
    valid_trials_sample_thread = pool.apply_async(request_valid_trials_sample, (experiment_id,))

    metric_names_results = metric_names_thread.get()
    train_metric_batches_results = train_metric_batches_thread.get()
    valid_metric_batches_results = valid_metric_batches_thread.get()
    train_trials_snapshot_results = train_trials_snapshot_thread.get()
    valid_trials_snapshot_results = valid_trials_snapshot_thread.get()
    train_trials_sample_results = train_trials_sample_thread.get()
    valid_trials_sample_results = valid_trials_sample_thread.get()

    if metric_names_results is not None:
        pytest.fail("metric-names: %s. Results: %s" % metric_names_results)
    if train_metric_batches_results is not None:
        pytest.fail("metric-batches (training): %s. Results: %s" % train_metric_batches_results)
    if valid_metric_batches_results is not None:
        pytest.fail("metric-batches (validation): %s. Results: %s" % valid_metric_batches_results)
    if train_trials_snapshot_results is not None:
        pytest.fail("trials-snapshot (training): %s. Results: %s" % train_trials_snapshot_results)
    if valid_trials_snapshot_results is not None:
        pytest.fail("trials-snapshot (validation): %s. Results: %s" % valid_trials_snapshot_results)
    if train_trials_sample_results is not None:
        pytest.fail("trials-sample (training): %s. Results: %s" % train_trials_sample_results)
    if valid_trials_sample_results is not None:
        pytest.fail("trials-sample (validation): %s. Results: %s" % valid_trials_sample_results)


def request_metric_names(experiment_id):  # type: ignore
    response = api.get(
        conf.make_master_url(),
        "api/v1/experiments/metrics-stream/metric-names?ids={}".format(experiment_id),
        params={"period_seconds": 1},
    )
    results = [message["result"] for message in map(json.loads, response.text.splitlines())]

    # First let's verify an empty response was sent back before any real work was done
    if results[0]["searcherMetrics"][0] != "validation_loss":
        return ("unexpected searcher metric in first response", results)
    if results[0]["trainingMetrics"] != []:
        return ("unexpected training metric in first response", results)
    if results[0]["validationMetrics"] != []:
        return ("unexpected validation metric in first response", results)

    # Then we verify that all expected responses are eventually received exactly once
    accumulated_training = set()
    accumulated_validation = set()
    for i in range(1, len(results)):
        for training in results[i]["trainingMetrics"]:
            if training in accumulated_training:
                return ("training metric appeared twice", results)
            accumulated_training.add(training)
        for validation in results[i]["validationMetrics"]:
            if validation in accumulated_validation:
                return ("training metric appeared twice", results)
            accumulated_validation.add(validation)

    if accumulated_training != {"loss"}:
        return ("unexpected set of training metrics", results)
    if accumulated_validation != {"validation_loss", "accuracy"}:
        return ("unexpected set of validation metrics", results)
    return None


def request_train_metric_batches(experiment_id):  # type: ignore
    response = api.get(
        conf.make_master_url(),
        "api/v1/experiments/{}/metrics-stream/batches".format(experiment_id),
        params={"metric_name": "loss", "metric_type": "METRIC_TYPE_TRAINING", "period_seconds": 1},
    )
    results = [message["result"] for message in map(json.loads, response.text.splitlines())]

    # First let's verify an empty response was sent back before any real work was done
    if results[0]["batches"] != []:
        return ("unexpected batches in first response", results)

    # Then we verify that all expected responses are eventually received exactly once
    accumulated = set()
    for i in range(1, len(results)):
        for batch in results[i]["batches"]:
            if batch in accumulated:
                return ("batch appears twice", results)
            accumulated.add(batch)
    if accumulated != {100, 200, 300, 400}:
        return ("unexpected set of batches", results)
    return None


def request_valid_metric_batches(experiment_id):  # type: ignore
    response = api.get(
        conf.make_master_url(),
        "api/v1/experiments/{}/metrics-stream/batches".format(experiment_id),
        params={
            "metric_name": "accuracy",
            "metric_type": "METRIC_TYPE_VALIDATION",
            "period_seconds": 1,
        },
    )
    results = [message["result"] for message in map(json.loads, response.text.splitlines())]

    # First let's verify an empty response was sent back before any real work was done
    if results[0]["batches"] != []:
        return ("unexpected batches in first response", results)

    # Then we verify that all expected responses are eventually received exactly once
    accumulated = set()
    for i in range(1, len(results)):
        for batch in results[i]["batches"]:
            if batch in accumulated:
                return ("batch appears twice", results)
            accumulated.add(batch)
    if accumulated != {200, 400}:
        return ("unexpected set of batches", results)
    return None


def validate_hparam_types(hparams: dict) -> Union[None, str]:
    for hparam in ["dropout1", "dropout2", "learning_rate"]:
        if type(hparams[hparam]) != float:
            return "hparam %s of unexpected type" % hparam
    for hparam in ["global_batch_size", "n_filters1", "n_filters2"]:
        if type(hparams[hparam]) != int:
            return "hparam %s of unexpected type" % hparam
    return None


def request_train_trials_snapshot(experiment_id):  # type: ignore
    response = api.get(
        conf.make_master_url(),
        "api/v1/experiments/{}/metrics-stream/trials-snapshot".format(experiment_id),
        params={
            "metric_name": "loss",
            "metric_type": "METRIC_TYPE_TRAINING",
            "batches_processed": 100,
            "period_seconds": 1,
        },
    )
    results = [message["result"] for message in map(json.loads, response.text.splitlines())]

    # First let's verify an empty response was sent back before any real work was done
    if results[0]["trials"] != []:
        return ("unexpected trials in first response", results)

    # Then we verify that we receive the expected number of trials and the right types
    trials = set()
    for i in range(1, len(results)):
        for trial in results[i]["trials"]:
            trials.add(trial["trialId"])
            validate_hparam_types(trial["hparams"])
            if type(trial["metric"]) != float:
                return ("metric of unexpected type", results)
    if len(trials) != 5:
        return ("unexpected number of trials received", results)
    return None


def request_valid_trials_snapshot(experiment_id):  # type: ignore
    response = api.get(
        conf.make_master_url(),
        "api/v1/experiments/{}/metrics-stream/trials-snapshot".format(experiment_id),
        params={
            "metric_name": "accuracy",
            "metric_type": "METRIC_TYPE_VALIDATION",
            "batches_processed": 200,
            "period_seconds": 1,
        },
    )
    results = [message["result"] for message in map(json.loads, response.text.splitlines())]

    # First let's verify an empty response was sent back before any real work was done
    if results[0]["trials"] != []:
        return ("unexpected trials in first response", results)

    # Then we verify that we receive the expected number of trials and the right types
    trials = set()
    for i in range(1, len(results)):
        for trial in results[i]["trials"]:
            trials.add(trial["trialId"])
            hparam_error = validate_hparam_types(trial["hparams"])
            if hparam_error is not None:
                return (hparam_error, results)
            if type(trial["metric"]) != float:
                return ("metric of unexpected type", results)
    if len(trials) != 5:
        return ("unexpected number of trials received", results)
    return None


def check_trials_sample_result(results: list) -> Union[None, tuple]:
    # First let's verify an empty response was sent back before any real work was done
    if (
        results[0]["trials"] != []
        or results[0]["promotedTrials"] != []
        or results[0]["demotedTrials"] != []
    ):
        return ("unexpected trials in first response", results)

    # Then we verify that we receive the expected number of trials and the right types
    trials: Set[int] = set()
    datapoints = {}
    for i in range(1, len(results)):
        newTrials = set()
        for trial in results[i]["promotedTrials"]:
            if trial in trials:
                return ("trial lists as promoted twice", results)
            newTrials.add(trial)
            datapoints[trial] = 0
        for trial in results[i]["trials"]:
            if trial["trialId"] in newTrials:
                hparam_error = validate_hparam_types(trial["hparams"])
                if hparam_error is not None:
                    return (hparam_error, results)
            else:
                if trial["hparams"] is not None:
                    return ("hparams repeated for trial", results)
            for point in trial["data"]:
                if point["batches"] > datapoints[trial["trialId"]]:
                    datapoints[trial["trialId"]] = point["batches"]
                else:
                    return ("data received out of order: " + str(trial["trialId"]), results)
    return None


def request_train_trials_sample(experiment_id):  # type: ignore
    response = api.get(
        conf.make_master_url(),
        "api/v1/experiments/{}/metrics-stream/trials-sample".format(experiment_id),
        params={
            "metric_name": "loss",
            "metric_type": "METRIC_TYPE_TRAINING",
            "period_seconds": 1,
        },
    )
    results = [message["result"] for message in map(json.loads, response.text.splitlines())]
    return check_trials_sample_result(results)


def request_valid_trials_sample(experiment_id):  # type: ignore
    response = api.get(
        conf.make_master_url(),
        "api/v1/experiments/{}/metrics-stream/trials-sample".format(experiment_id),
        params={
            "metric_name": "accuracy",
            "metric_type": "METRIC_TYPE_VALIDATION",
            "period_seconds": 1,
        },
    )
    results = [message["result"] for message in map(json.loads, response.text.splitlines())]
    return check_trials_sample_result(results)


@pytest.mark.e2e_cpu
@pytest.mark.parametrize("group", ["validation", "training", "abc"])
def test_trial_time_series(group: str) -> None:
    exp_id = exp.create_experiment(
        conf.fixtures_path("no_op/single-one-short-step.yaml"),
        conf.fixtures_path("no_op"),
        ["--project_id", str(1), ("--paused")],
    )
    trials = exp.experiment_trials(exp_id)
    trial_id = trials[0].trial.id
    sess = api_utils.determined_test_session(admin=False)
    metric_names = ["lossx"]

    trial_metrics = bindings.v1TrialMetrics(
        metrics=bindings.v1Metrics(avgMetrics={name: 3.3 for name in metric_names}),
        stepsCompleted=10,
        trialId=trial_id,
        trialRunId=0,
    )
    bindings.post_ReportTrialMetrics(
        sess,
        body=bindings.v1ReportTrialMetricsRequest(group=group, metrics=trial_metrics),
        metrics_trialId=trial_id,
    )
    trial_resp = bindings.get_CompareTrials(
        sess, trialIds=[trial_id], metricIds=[f"{group}.{name}" for name in metric_names]
    ).trials[0]

    assert trial_resp.metrics[0].data[0].values is not None
    print(trial_resp.metrics[0].data[0].values)
    for name in metric_names:
        val = trial_resp.metrics[0].data[0].values[name]
        assert val == 3.3, f"unexpected value for metric {name}, type: {type(val)}"


@pytest.mark.e2e_cpu
def test_trial_describe_metrics() -> None:
    exp_id = exp.run_basic_test(
        conf.fixtures_path("no_op/single-one-short-step.yaml"), conf.fixtures_path("no_op"), 1
    )
    trials = exp.experiment_trials(exp_id)
    trial_id = trials[0].trial.id

    cmd = [
        "det",
        "-m",
        conf.make_master_url(),
        "trial",
        "describe",
        "--json",
        "--metrics",
        str(trial_id),
    ]

    output = json.loads(subprocess.check_output(cmd))

    workloads = output["workloads"]
    assert len(workloads) == 102
    flattened_batch_metrics: List[Dict[str, float]] = sum(
        (w["training"]["metrics"]["batchMetrics"] for w in workloads if w["training"]), []
    )
    losses = [m["loss"] for m in flattened_batch_metrics]

    assert len(losses) == 100

    # assert summary metrics in trial
    sess = api_utils.determined_test_session(admin=True)
    resp = bindings.get_GetTrial(session=sess, trialId=trial_id)
    summaryMetrics = resp.trial.summaryMetrics
    assert summaryMetrics is not None
    assert summaryMetrics["avg_metrics"]["loss"]["count"] == 100
    assert summaryMetrics["avg_metrics"]["loss"]["max"] is not None
    assert summaryMetrics["avg_metrics"]["loss"]["min"] is not None
    assert summaryMetrics["avg_metrics"]["loss"]["sum"] is not None
    assert summaryMetrics["avg_metrics"]["loss"]["type"] == "number"
