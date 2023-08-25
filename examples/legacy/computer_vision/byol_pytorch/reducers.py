from collections import defaultdict
from functools import reduce
from typing import Any, Dict, List

from utils import merge_dicts

from determined.pytorch import MetricReducer


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
