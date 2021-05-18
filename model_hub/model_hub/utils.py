from typing import List, Union

import numpy as np
import torch


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
    raise TypeError("Expected input of type List, np.ndarray, or torch.Tensor.")
