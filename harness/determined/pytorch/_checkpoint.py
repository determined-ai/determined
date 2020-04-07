import pathlib
import sys
from typing import Optional

import cloudpickle
import torch
import torch.nn as nn

from determined_common.checkpoint import download


def load(
    trial_id: int,
    latest: bool = False,
    best: bool = False,
    uuid: Optional[str] = None,
    ckpt_path: Optional[str] = None,
    master: Optional[str] = None,
    metric_name: Optional[str] = None,
    smaller_is_better: Optional[bool] = None,
) -> nn.Module:
    if not ckpt_path or not pathlib.Path(ckpt_path).exists():
        ckpt_path, _ = download(
            trial_id,
            latest=latest,
            best=best,
            uuid=uuid,
            output_dir=ckpt_path,
            master=master,
            metric_name=metric_name,
            smaller_is_better=smaller_is_better,
        )

    ckpt_dir = pathlib.Path(ckpt_path)
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

    return torch.load(maybe_model, pickle_module=cloudpickle.pickle)  # type: ignore
