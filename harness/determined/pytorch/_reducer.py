from enum import Enum
from typing import List, Optional

import numpy as np

import determined_common.check as check


class Reducer(Enum):
    """
    A ``Reducer`` defines a method for reducing (aggregating) evaluation
    metrics. See :meth:`determined.pytorch.PyTorchTrial.evaluation_reducer` for
    details.

    Attributes:
        AVG
        SUM
        MAX
        MIN
    """

    # TODO: Support additional reducers.
    AVG = 1
    SUM = 2
    MAX = 3
    MIN = 4


def _reduce_metrics(
    reducer: Reducer, metrics: np.array, num_batches: Optional[List[int]] = None
) -> np.float:
    if reducer == Reducer.AVG:
        if num_batches:
            check.check_eq(len(metrics), len(num_batches))
        return np.average(metrics, weights=num_batches)
    elif reducer == Reducer.SUM:
        return np.sum(metrics)
    elif reducer == Reducer.MAX:
        return np.max(metrics)
    elif reducer == Reducer.MIN:
        return np.min(metrics)
    else:
        raise NotImplementedError
