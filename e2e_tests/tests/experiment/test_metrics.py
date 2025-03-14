import json
from multiprocessing import pool
from typing import Set, Union

import pytest

from determined.common.api import bindings
from determined.experimental import client
from tests import api_utils
from tests import config as conf
from tests import detproc
from tests import experiment as exp
from tests.experiment import noop


@pytest.mark.e2e_cpu
@pytest.mark.timeout(600)
def test_streaming_metrics_api() -> None:
    sess = api_utils.user_session()
    thread_pool = pool.ThreadPool(processes=7)

    experiment_id = exp.create_experiment(
        sess,
        conf.fixtures_path("mnist_pytorch/adaptive_short.yaml"),
        conf.fixtures_path("mnist_pytorch"),
    )
    # To fully test the streaming APIs, the requests need to start running immediately after the
    # experiment, and then stay open until the experiment is complete. To accomplish this with all
    # of the API calls on a single experiment, we spawn them all in threads.

    # The HP importance portion of this test is commented out until the feature is enabled by
    # default

    metric_names_thread = thread_pool.apply_async(request_metric_names, (experiment_id,))
    train_metric_batches_thread = thread_pool.apply_async(
        request_train_metric_batches, (experiment_id,)
    )
    valid_metric_batches_thread = thread_pool.apply_async(
        request_valid_metric_batches, (experiment_id,)
    )
    train_trials_snapshot_thread = thread_pool.apply_async(
        request_train_trials_snapshot, (experiment_id,)
    )
    valid_trials_snapshot_thread = thread_pool.apply_async(
        request_valid_trials_snapshot, (experiment_id,)
    )
    train_trials_sample_thread = thread_pool.apply_async(
        request_train_trials_sample, (experiment_id,)
    )
    valid_trials_sample_thread = thread_pool.apply_async(
        request_valid_trials_sample, (experiment_id,)
    )

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
    sess = api_utils.user_session()
    response = sess.get(
        f"api/v1/experiments/metrics-stream/metric-names?ids={experiment_id}",
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

    if accumulated_training != {"loss", "batches", "epochs"}:
        return (f"unexpected set of training metrics {accumulated_training}", results)
    if accumulated_validation != {"validation_loss", "accuracy", "batches", "epochs"}:
        return (f"unexpected set of validation metrics {accumulated_validation}", results)
    return None


def request_train_metric_batches(experiment_id):  # type: ignore
    sess = api_utils.user_session()
    response = sess.get(
        f"api/v1/experiments/{experiment_id}/metrics-stream/batches",
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
    sess = api_utils.user_session()
    response = sess.get(
        f"api/v1/experiments/{experiment_id}/metrics-stream/batches",
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
    if accumulated != {100, 200, 300, 400}:
        return (f"unexpected set of batches: {accumulated}", results)
    return None


def validate_hparam_types(hparams: dict) -> Union[None, str]:
    for hparam in ["dropout1", "dropout2", "learning_rate"]:
        if type(hparams[hparam]) != float:
            return "hparam %s of unexpected type" % hparam
    for hparam in ["n_filters1", "n_filters2"]:
        if type(hparams[hparam]) != int:
            return "hparam %s of unexpected type" % hparam
    return None


def request_train_trials_snapshot(experiment_id):  # type: ignore
    sess = api_utils.user_session()
    response = sess.get(
        f"api/v1/experiments/{experiment_id}/metrics-stream/trials-snapshot",
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
    sess = api_utils.user_session()
    response = sess.get(
        f"api/v1/experiments/{experiment_id}/metrics-stream/trials-snapshot",
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
    sess = api_utils.user_session()
    response = sess.get(
        f"api/v1/experiments/{experiment_id}/metrics-stream/trials-sample",
        params={
            "metric_name": "loss",
            "metric_type": "METRIC_TYPE_TRAINING",
            "period_seconds": 1,
        },
    )
    results = [message["result"] for message in map(json.loads, response.text.splitlines())]
    return check_trials_sample_result(results)


def request_valid_trials_sample(experiment_id):  # type: ignore
    sess = api_utils.user_session()
    response = sess.get(
        f"api/v1/experiments/{experiment_id}/metrics-stream/trials-sample",
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
    sess = api_utils.user_session()
    exp_ref = noop.create_paused_experiment(sess)
    trials = exp.experiment_trials(sess, exp_ref.id)
    trial_id = trials[0].trial.id
    metric_names = ["lossx"]

    trial_metrics = bindings.v1TrialMetrics(
        metrics=bindings.v1Metrics(avgMetrics=dict.fromkeys(metric_names, 3.3)),
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
    exp_ref.kill()


@pytest.mark.e2e_cpu
def test_trial_describe_metrics() -> None:
    sess = api_utils.user_session()
    exp_ref = noop.create_experiment(sess, noop.traininglike_steps(10, metric_scale=1.1))
    assert exp_ref.wait(interval=0.01) == client.ExperimentState.COMPLETED
    trial_id = exp_ref.get_trials()[0].id

    cmd = [
        "det",
        "trial",
        "describe",
        "--json",
        "--metrics",
        str(trial_id),
    ]

    output = detproc.check_json(sess, cmd)

    workloads = output["workloads"]
    assert len(workloads) == 30

    # assert summary metrics in trial
    resp = bindings.get_GetTrial(session=sess, trialId=trial_id)
    summaryMetrics = resp.trial.summaryMetrics
    mean = sum(1.1**i for i in range(10)) / 10
    assert summaryMetrics is not None
    assert summaryMetrics["avg_metrics"]["x"]["count"] == 10
    assert abs(summaryMetrics["avg_metrics"]["x"]["max"] - 1.1**9) < 0.00001
    assert summaryMetrics["avg_metrics"]["x"]["min"] == 1
    assert abs(summaryMetrics["avg_metrics"]["x"]["mean"] - mean) < 0.0001
    assert summaryMetrics["avg_metrics"]["x"]["type"] == "number"
