import logging
from typing import Callable, List

import numpy as np
from torch import Tensor

from determined.pytorch import MetricReducer


def configure_logging(level=logging.INFO):  # type: ignore
    logging.basicConfig(
        format="%(asctime)s - %(levelname)s - %(name)s -   %(message)s",
        datefmt="%m/%d/%Y %H:%M:%S",
        level=level,
    )


def expand_like(arrays: List[np.ndarray], fill: float = -100) -> np.ndarray:
    full_shape = list(arrays[0].shape)
    if len(full_shape) == 1:
        return np.concatenate(arrays)
    full_shape[0] = sum(a.shape[0] for a in arrays)
    full_shape[1] = max(a.shape[1] for a in arrays)
    result = np.full(full_shape, fill)
    row_offset = 0
    for a in arrays:
        result[row_offset : row_offset + a.shape[0], : a.shape[1]] = a
        row_offset += a.shape[0]
    return result


def numpify(x) -> np.ndarray:  # type: ignore
    if isinstance(x, np.ndarray):
        return x
    if isinstance(x, List):
        return np.array(x)
    if isinstance(x, Tensor):
        return x.cpu().numpy()
    raise NotImplementedError


class PredLabelFnReducer(MetricReducer):
    def __init__(self, fn: Callable):
        self.fn = fn
        self.reset()

    def reset(self) -> None:
        self.predictions: List[np.ndarray] = []
        self.labels: List[np.ndarray] = []

    def update(self, preds, labels) -> None:  # type: ignore
        self.predictions.append(numpify(preds))
        self.labels.append(numpify(labels))

    def per_slot_reduce(self) -> np.ndarray:
        return expand_like(self.predictions), expand_like(self.labels)

    def cross_slot_reduce(self, per_slot_metrics: List[np.ndarray]):  # type: ignore
        predictions, labels = zip(*per_slot_metrics)
        return self.fn(expand_like(predictions), expand_like(labels))
