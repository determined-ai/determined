import logging
import os
import urllib.parse
from typing import Dict, List, Union

import filelock
import numpy as np
import requests
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


def download_url(download_directory: str, url: str) -> str:
    url_path = urllib.parse.urlparse(url).path
    basename = url_path.rsplit("/", 1)[1]

    os.makedirs(download_directory, exist_ok=True)
    filepath = os.path.join(download_directory, basename)
    lock = filelock.FileLock(filepath + ".lock")

    with lock:
        if not os.path.exists(filepath):
            logging.info("Downloading {} to {}".format(url, filepath))

            r = requests.get(url, stream=True)
            with open(filepath, "wb") as f:
                for chunk in r.iter_content(chunk_size=8192):
                    if chunk:
                        f.write(chunk)
    return filepath


def compute_num_training_steps(experiment_config: Dict, global_batch_size: int) -> int:
    max_length_unit = list(experiment_config["searcher"]["max_length"].keys())[0]
    max_length: int = experiment_config["searcher"]["max_length"][max_length_unit]
    if max_length_unit == "batches":
        return max_length
    if max_length_unit == "epochs":
        if "records_per_epoch" in experiment_config:
            return max_length * int(experiment_config["records_per_epoch"] / global_batch_size)
        raise Exception(
            "Missing num_training_steps hyperparameter in the experiment "
            "configuration, which is needed to configure the learning rate scheduler."
        )
    # Otherwise, max_length_unit=='records'
    return int(max_length / global_batch_size)
