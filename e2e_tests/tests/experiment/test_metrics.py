import json

import pytest

import determined_common.api.authentication as auth
from determined_common import api
from tests import config as conf
from tests import experiment as exp


@pytest.mark.e2e_cpu  # type: ignore
@pytest.mark.timeout(180)  # type: ignore
def test_streaming_metric_names() -> None:
    auth.initialize_session(conf.make_master_url(), try_reauth=True)

    experiment_id = exp.create_experiment(
        conf.fixtures_path("no_op/single-medium-train-step.yaml"), conf.fixtures_path("no_op")
    )

    # This request starts immediately after the experiment, and will return when it completes.
    # If we timeout, it means it failed to complete, or we didn't close the connection after it did
    response = api.get(
        conf.make_master_url(),
        "api/v1/experiments/{}/metrics-stream/metric-names".format(experiment_id),
    )
    results = [message["result"] for message in map(json.loads, response.text.splitlines())]

    # First let's verify an empty response was sent back before any real work was done
    assert results[0]["searcherMetric"] == "validation_error"
    assert results[0]["trainingMetrics"] == []
    assert results[0]["validationMetrics"] == []

    # Then we verify that all expected responses are eventually received exactly once
    accumulated_training = set()
    accumulated_validation = set()
    for i in range(1, len(results)):
        for training in results[i]["trainingMetrics"]:
            assert training not in accumulated_training
            accumulated_training.add(training)
        for validation in results[i]["validationMetrics"]:
            assert validation not in accumulated_validation
            accumulated_validation.add(validation)
    assert accumulated_training == {"loss"}
    assert accumulated_validation == {"validation_error"}


@pytest.mark.e2e_cpu  # type: ignore
@pytest.mark.timeout(90)  # type: ignore
def test_streaming_metric_batches() -> None:
    experiment_id = exp.create_experiment(
        conf.fixtures_path("no_op/single-medium-train-step.yaml"), conf.fixtures_path("no_op")
    )

    # This request starts immediately after the experiment, and will return when it completes.
    # If we timeout, it means it failed to complete, or we didn't close the connection after it did
    response = api.get(
        conf.make_master_url(),
        "api/v1/experiments/{}/metrics-stream/batches".format(experiment_id),
        params={"training_metric": "loss"},
    )
    results = [message["result"] for message in map(json.loads, response.text.splitlines())]

    # First let's verify an empty response was sent back before any real work was done
    assert results[0]["batches"] == []

    # Then we verify that all expected responses are eventually received exactly once
    accumulated = set()
    for i in range(1, len(results)):
        for batch in results[i]["batches"]:
            assert batch not in accumulated
            accumulated.add(batch)
    assert accumulated == {100, 200, 300, 400, 500}
