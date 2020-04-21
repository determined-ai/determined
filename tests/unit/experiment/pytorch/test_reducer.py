import numpy as np

from determined.pytorch import Reducer, _reduce_metrics


def test_reducer() -> None:
    metrics = np.array([0.25, 0.5, 0.75, 1, 25.5, 1.9])
    assert np.around(_reduce_metrics(Reducer.AVG, metrics), decimals=2) == 4.98
    assert _reduce_metrics(Reducer.SUM, metrics) == 29.9
    assert _reduce_metrics(Reducer.MIN, metrics) == 0.25
    assert _reduce_metrics(Reducer.MAX, metrics) == 25.5

    batches_per_process = [1, 2, 5, 4, 5, 6]
    assert np.around(_reduce_metrics(Reducer.AVG, metrics, batches_per_process), decimals=2) == 6.43
