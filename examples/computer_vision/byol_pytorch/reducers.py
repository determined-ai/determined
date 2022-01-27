from collections import defaultdict
from functools import reduce
from typing import Any, Callable, Dict, List, TypeVar

from determined.pytorch import MetricReducer

A = TypeVar("A")
B = TypeVar("B")


def merge_dicts(d1: Dict[A, B], d2: Dict[A, B], f: Callable[[B, B], B]) -> Dict[A, B]:
    """
    Merges dictionaries with a custom merge function.
    E.g. if k in d1 and k in d2, result[k] == f(d1[k], d2[k]).
    Otherwise, if e.g. k is in only d1, result[k] == d1[k]
    """
    d1_keys = d1.keys()
    d2_keys = d2.keys()
    shared = d1_keys & d2_keys
    d1_exclusive = d1_keys - d2_keys
    d2_exclusive = d2_keys - d1_keys
    new_dict = {k: f(d1[k], d2[k]) for k in shared}
    new_dict.update({k: d1[k] for k in d1_exclusive})
    new_dict.update({k: d2[k] for k in d2_exclusive})
    return new_dict


class ValidatedAccuracyReducer(MetricReducer):
    """
    Given two datasets and a hyperparameter (e.g. LR), report accuracy on second dataset for
    optimal value of the hyperparameter on the first.
    """

    def __init__(self) -> None:
        self.reset()

    def reset(self) -> None:
        self.val_correct_by_param: Dict[Any, int] = defaultdict(int)
        self.test_correct_by_param: Dict[Any, int] = defaultdict(int)
        self.test_ct = 0

    def update(
        self,
        val_correct_by_param: Dict[Any, int],
        test_correct_by_param: Dict[Any, int],
        test_ct: int,
    ) -> None:
        self.val_correct_by_param = merge_dicts(
            self.val_correct_by_param, val_correct_by_param, lambda x, y: x + y
        )
        self.test_correct_by_param = merge_dicts(
            self.test_correct_by_param, test_correct_by_param, lambda x, y: x + y
        )
        self.test_ct += test_ct

    def per_slot_reduce(self) -> Any:
        return self.val_correct_by_param, self.test_correct_by_param, self.test_ct

    def cross_slot_reduce(self, per_slot_metrics: List) -> Any:
        # per_slot_metrics is a list of (sum, counts) tuples
        # returned by the self.pre_slot_reduce() on each slot
        val_correct_by_param, test_correct_by_param, test_ct = zip(*per_slot_metrics)
        val_correct_by_param = reduce(
            lambda x, y: merge_dicts(x, y, lambda a, b: a + b), val_correct_by_param
        )
        test_correct_by_param = reduce(
            lambda x, y: merge_dicts(x, y, lambda a, b: a + b), test_correct_by_param
        )
        test_ct = sum(test_ct)
        max_val_param = max(val_correct_by_param, key=val_correct_by_param.get)
        return test_correct_by_param[max_val_param] / test_ct


class AvgReducer(MetricReducer):
    def __init__(self) -> None:
        self.reset()

    def reset(self) -> None:
        self.sum = 0.0
        self.counts = 0

    def update(self, value: float) -> None:
        self.sum += value
        self.counts += 1

    def per_slot_reduce(self) -> Any:
        return self.sum, self.counts

    def cross_slot_reduce(self, per_slot_metrics: List) -> Any:
        sums, counts = zip(*per_slot_metrics)
        return sum(sums) / sum(counts)
