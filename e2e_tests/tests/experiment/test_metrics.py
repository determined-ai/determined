import json
import multiprocessing as mp

import pytest

import determined_common.api.authentication as auth
from determined_common import api
from tests import config as conf
from tests import experiment as exp


@pytest.mark.e2e_cpu  # type: ignore
@pytest.mark.timeout(150)  # type: ignore
def test_streaming_metrics_api() -> None:
    auth.initialize_session(conf.make_master_url(), try_reauth=True)

    pool = mp.pool.ThreadPool(processes=2)

    experiment_id = exp.create_experiment(
        conf.fixtures_path("mnist_pytorch/adaptive_short.yaml"),
        conf.tutorials_path("mnist_pytorch"),
    )
    # To fully test the streaming APIs, the requests need to start running immediately after the
    # experiment, and then stay open until the experiment is complete. To accomplish this with all
    # of the API calls on a single experiment, we spawn them all in threads.

    metric_names_thread = pool.apply_async(request_metric_names, (experiment_id,))
    metric_batches_thread = pool.apply_async(request_metric_batches, (experiment_id,))

    metric_names_results = metric_names_thread.get()
    metric_batches_results = metric_batches_thread.get()
    if metric_names_results is not None:
        pytest.fail("metric-names: %s. Results: %s" % metric_names_results)
    if metric_batches_results is not None:
        pytest.fail("metric-batches: %s. Results: %s" % metric_batches_results)


def request_metric_names(experiment_id):  # type: ignore
    response = api.get(
        conf.make_master_url(),
        "api/v1/experiments/{}/metrics-stream/metric-names".format(experiment_id),
    )
    results = [message["result"] for message in map(json.loads, response.text.splitlines())]

    # First let's verify an empty response was sent back before any real work was done
    if results[0]["searcherMetric"] != "validation_loss":
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


def request_metric_batches(experiment_id):  # type: ignore
    response = api.get(
        conf.make_master_url(),
        "api/v1/experiments/{}/metrics-stream/batches".format(experiment_id),
        params={"training_metric": "loss"},
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
