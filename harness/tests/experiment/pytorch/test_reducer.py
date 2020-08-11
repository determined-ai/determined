import numpy as np

from determined import pytorch


def do_test_reducer(reducer_cls, metrics_1, metrics_2, expected):
    reducer_1 = reducer_cls()
    reducer_2 = reducer_cls()
    for i in metrics_1:
        last_state_1 = reducer_1.accumulate(np.array(i))
    for i in metrics_2:
        last_state_2 = reducer_2.accumulate(np.array(i))
    reduced = reducer_1.cross_slot_reduce([last_state_1, last_state_2])
    assert reduced == expected


def test_avg_reducer() -> None:
    metrics_1 = list(range(10))
    metrics_2 = list(range(10, 15))
    expected = sum([*metrics_1, *metrics_2]) / (len(metrics_1) + len(metrics_2))
    do_test_reducer(pytorch.AvgMetricReducer, metrics_1, metrics_2, expected)


def test_sum_reducer() -> None:
    metrics_1 = list(range(10))
    metrics_2 = list(range(10, 15))
    expected = sum([*metrics_1, *metrics_2])
    do_test_reducer(pytorch.SumMetricReducer, metrics_1, metrics_2, expected)


def test_max_reducer() -> None:
    metrics_1 = list(range(10))
    metrics_2 = list(range(10, 15))
    expected = max([*metrics_1, *metrics_2])
    do_test_reducer(pytorch.MaxMetricReducer, metrics_1, metrics_2, expected)


def test_min_reducer() -> None:
    metrics_1 = list(range(10))
    metrics_2 = list(range(10, 15))
    expected = min([*metrics_1, *metrics_2])
    do_test_reducer(pytorch.MinMetricReducer, metrics_1, metrics_2, expected)
