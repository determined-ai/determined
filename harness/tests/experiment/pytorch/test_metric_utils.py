import logging
from typing import Any, Dict, Union

import numpy as np
import pytest
import torch

import determined.errors
from determined.pytorch import Reducer
from determined.pytorch._metric_utils import (
    _average_training_metrics,
    _convert_metrics_to_numpy,
    _prepare_metrics_reducers,
    _process_combined_metrics_and_batches,
)

logger = logging.getLogger(__name__)


def test_process_combined_metrics_and_batches() -> None:
    combined_metrics_and_batches = [
        ({}, 0),
        ({}, 0),
        ({"loss1": [1, 2], "loss2": [-1, -2]}, 2),
        ({"loss1": [3, 4], "loss2": [-3, -4]}, 2),
    ]
    processed_metrics, processed_batches = _process_combined_metrics_and_batches(
        combined_metrics_and_batches
    )
    expected_metrics = {"loss1": [[1, 2], [3, 4]], "loss2": [[-1, -2], [-3, -4]]}
    assert processed_metrics == expected_metrics
    assert processed_batches == [2, 2]

    combined_metrics_and_batches = [
        ({}, 0),
        ({}, 0),
        ({"loss1": [1, 2], "loss2": [np.array(-1), np.array(-2)]}, 2),
        ({"loss1": [3, 4], "loss2": [np.array(-3), np.array(-4)]}, 2),
    ]
    processed_metrics, processed_batches = _process_combined_metrics_and_batches(
        combined_metrics_and_batches
    )
    expected_metrics = {
        "loss1": [[1, 2], [3, 4]],
        "loss2": [[np.array(-1), np.array(-2)], [np.array(-3), np.array(-4)]],
    }
    assert processed_metrics == expected_metrics
    assert processed_batches == [2, 2]


def test_average_training_metrics() -> None:
    combined_timeseries = {"loss1": [[1, 2], [3, 4]], "loss2": [[-1, -2], [-3, -4]]}
    combined_num_batches = [2, 2]
    averaged_metrics = _average_training_metrics(combined_timeseries, combined_num_batches)
    expected_metrics = [
        {"loss1": 2, "loss2": -2},
        {"loss1": 3, "loss2": -3},
    ]
    assert averaged_metrics == expected_metrics

    # Test single array metrics
    combined_timeseries = {
        "loss1": [[1, 2], [3, 4]],
        "loss2": [[np.array(-1), np.array(-2)], [np.array(-3), np.array(-4)]],
    }
    averaged_metrics = _average_training_metrics(combined_timeseries, combined_num_batches)
    expected_metrics = [
        {"loss1": 2, "loss2": np.array(-2)},
        {"loss1": 3, "loss2": np.array(-3)},
    ]
    assert averaged_metrics == expected_metrics


def test_prepare_metric_reducers() -> None:
    metrics_dict = {"loss1": 1, "loss2": 2}

    # Test mismatched keys
    reducer = {}  # type: Union[Dict[str, Any], Reducer]
    with pytest.raises(determined.errors.InvalidExperimentException):
        _ = _prepare_metrics_reducers(reducer, metrics_dict.keys())

    # Test invalid reducer
    reducer = {"loss1": Reducer.AVG, "loss2": 2}
    with pytest.raises(determined.errors.InvalidExperimentException):
        _ = _prepare_metrics_reducers(reducer, metrics_dict.keys())

    # Test single reducer
    reducer = Reducer.AVG
    prepped_reducers = _prepare_metrics_reducers(reducer, metrics_dict.keys())
    assert prepped_reducers == {"loss1": Reducer.AVG, "loss2": Reducer.AVG}

    # Test reducer dict
    reducer = {"loss1": Reducer.AVG, "loss2": Reducer.SUM}
    prepped_reducers = _prepare_metrics_reducers(reducer, metrics_dict.keys())
    assert prepped_reducers == {"loss1": Reducer.AVG, "loss2": Reducer.SUM}


def test_convert_metrics_to_numpy() -> None:
    metrics = {"loss1": 1, "loss2": torch.tensor(2)}
    converted_metrics = _convert_metrics_to_numpy(metrics)
    assert converted_metrics == {"loss1": 1, "loss2": np.array(2)}
