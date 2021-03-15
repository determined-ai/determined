from typing import Any, Callable, List, Union

import numpy as np
import torch

import determined.pytorch as det_torch


def expand_like(arrays: List[np.ndarray], fill: float = -100) -> np.ndarray:
    """
    Stacks a list of arrays along the first dimension; the arrays are allowed to differ in
    the second dimension but should match for dim > 2.

    The output will have dimension
    (sum([l.shape[0] for l in arrays]), max([l.shape[1] for l in in arrays]), ...)
    For arrays that have fewer entries in the second dimension than the max, we will
    pad with the fill value.

    Args:
        arrays: List of np.ndarray to stack along the first dimension
        fill: Value to fill in when padding to max size in the second dimension

    Returns:
        stacked array
    """
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


def numpify(x: Union[List, np.ndarray, torch.Tensor]) -> np.ndarray:
    """
    Converts List or torch.Tensor to numpy.ndarray.
    """
    if isinstance(x, np.ndarray):
        return x
    if isinstance(x, List):
        return np.array(x)
    if isinstance(x, torch.Tensor):
        return x.cpu().numpy()
    raise NotImplementedError


class PredLabelFnReducer(det_torch.MetricReducer):
    def __init__(self, fn: Callable):
        """
        Custom reducer that will apply the provided fn to predictions and labels aggregated
        across all ranks.

        We will collected batched predictions and labels in each slot.  Then the batched
        predictions and labels will be stacked together along the first dimensions.
        The batch predictions can differ in the second dimension and we will simply fill
        to the max size with -100.

        See the expand_like function above for more details.
        """
        self.fn = fn
        self.reset()

    def reset(self) -> None:
        self.predictions: List[np.ndarray] = []
        self.labels: List[np.ndarray] = []

    def update(
        self,
        preds: Union[List, np.ndarray, torch.Tensor],
        labels: Union[List, np.ndarray, torch.Tensor],
    ) -> None:
        self.predictions.append(numpify(preds))
        self.labels.append(numpify(labels))

    def per_slot_reduce(self) -> np.ndarray:
        return expand_like(self.predictions), expand_like(self.labels)

    def cross_slot_reduce(self, per_slot_metrics: List[np.ndarray]) -> Any:
        predictions, labels = zip(*per_slot_metrics)
        return self.fn(expand_like(predictions), expand_like(labels))
