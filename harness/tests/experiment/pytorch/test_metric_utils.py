import logging
from typing import Any, Dict, List, Tuple, Union

import numpy as np
import pytest
import torch

import determined.errors
import determined.pytorch._metric_utils as metric_utils
from determined import pytorch

logger = logging.getLogger(__name__)


def test_process_combined_metrics_and_batches() -> None:
    # Test empty metrics. This case can arise when custom reducers are used, so it's okay for
    # evaluate_batch to return an empty dict.
    combined_metrics_and_batches = [
        ({}, 2),  # this is from a rank actually computing metrics but returning an empty dict
        ({}, 2),
    ]  # type: List[Tuple[Dict[Any, Any], int]]
    processed_metrics, processed_batches = metric_utils._process_combined_metrics_and_batches(
        combined_metrics_and_batches
    )
    expected_metrics = {}  # type: dict
    assert processed_metrics == expected_metrics
    assert processed_batches == [2, 2]

    # Test combine with some ranks not reporting metrics.
    combined_metrics_and_batches = [
        ({}, 0),  # this corresponds to a rank that does not compute metrics and gets filtered out
        ({}, 0),
        ({"loss1": [1, 2], "loss2": [-1, -2]}, 2),
        ({"loss1": [3, 4], "loss2": [-3, -4]}, 2),
    ]
    processed_metrics, processed_batches = metric_utils._process_combined_metrics_and_batches(
        combined_metrics_and_batches
    )
    expected_metrics = {"loss1": [[1, 2], [3, 4]], "loss2": [[-1, -2], [-3, -4]]}
    assert processed_metrics == expected_metrics
    assert processed_batches == [2, 2]

    # Test array handling.
    combined_metrics_and_batches = [
        ({}, 0),
        ({}, 0),
        ({"loss1": [1, 2], "loss2": [np.array(-1), np.array(-2)]}, 2),
        ({"loss1": [3, 4], "loss2": [np.array(-3), np.array(-4)]}, 2),
    ]
    processed_metrics, processed_batches = metric_utils._process_combined_metrics_and_batches(
        combined_metrics_and_batches
    )
    expected_metrics = {
        "loss1": [[1, 2], [3, 4]],
        "loss2": [[np.array(-1), np.array(-2)], [np.array(-3), np.array(-4)]],
    }
    assert processed_metrics == expected_metrics
    assert processed_batches == [2, 2]


def test_average_training_metrics() -> None:
    combined_timeseries: Dict[str, Any] = {"loss1": [[1, 2], [3, 4]], "loss2": [[-1, -2], [-3, -4]]}
    combined_num_batches = [2, 2]
    averaged_metrics = metric_utils._average_training_metrics(
        combined_timeseries, combined_num_batches
    )
    expected_metrics: List[Dict[str, Any]] = [
        {"loss1": 2, "loss2": -2},
        {"loss1": 3, "loss2": -3},
    ]
    assert averaged_metrics == expected_metrics

    # Test single array metrics
    combined_timeseries = {
        "loss1": [[1, 2], [3, 4]],
        "loss2": [[np.array(-1), np.array(-2)], [np.array(-3), np.array(-4)]],
    }
    averaged_metrics = metric_utils._average_training_metrics(
        combined_timeseries, combined_num_batches
    )
    expected_metrics = [
        {"loss1": 2, "loss2": np.array(-2)},
        {"loss1": 3, "loss2": np.array(-3)},
    ]
    assert averaged_metrics == expected_metrics


def test_prepare_metric_reducers() -> None:
    metrics_dict = {"loss1": 1, "loss2": 2}

    # Test mismatched keys
    reducer = {}  # type: Union[Dict[str, Any], pytorch.Reducer]
    with pytest.raises(determined.errors.InvalidExperimentException):
        _ = metric_utils._prepare_metrics_reducers(reducer, metrics_dict.keys())

    # Test invalid reducer
    reducer = {"loss1": pytorch.Reducer.AVG, "loss2": 2}
    with pytest.raises(determined.errors.InvalidExperimentException):
        _ = metric_utils._prepare_metrics_reducers(reducer, metrics_dict.keys())

    # Test single reducer
    reducer = pytorch.Reducer.AVG
    prepped_reducers = metric_utils._prepare_metrics_reducers(reducer, metrics_dict.keys())
    assert prepped_reducers == {"loss1": pytorch.Reducer.AVG, "loss2": pytorch.Reducer.AVG}

    # Test reducer dict
    reducer = {"loss1": pytorch.Reducer.AVG, "loss2": pytorch.Reducer.SUM}
    prepped_reducers = metric_utils._prepare_metrics_reducers(reducer, metrics_dict.keys())
    assert prepped_reducers == {"loss1": pytorch.Reducer.AVG, "loss2": pytorch.Reducer.SUM}


def test_convert_metrics_to_numpy() -> None:
    metrics = {"loss1": 1, "loss2": torch.tensor(2)}
    converted_metrics = metric_utils._convert_metrics_to_numpy(metrics)
    assert converted_metrics == {"loss1": 1, "loss2": np.array(2)}
