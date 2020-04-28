import pathlib
import sys
from typing import Any

import cloudpickle
import torch


def load_model(ckpt_dir: pathlib.Path, **kwargs: Any) -> torch.nn.Module:
    code_path = ckpt_dir.joinpath("code")

    # We used MLflow's MLmodel checkpoint format in the past. This format
    # nested the checkpoint in data/. Currently, we have the checkpoint at the
    # top level of the checkpoint directory.
    potential_model_paths = [["model.pth"], ["data", "model.pth"]]

    for nested_path in potential_model_paths:
        maybe_model = ckpt_dir.joinpath(*nested_path)
        if maybe_model.exists():
            break

    if not maybe_model.exists():
        raise AssertionError("checkpoint at {} doesn't include a model.pth file".format(ckpt_dir))

    code_subdirs = [str(x) for x in code_path.iterdir() if x.is_dir()]
    sys.path = [str(code_path)] + code_subdirs + sys.path

    return torch.load(maybe_model, pickle_module=cloudpickle.pickle, **kwargs)  # type: ignore
