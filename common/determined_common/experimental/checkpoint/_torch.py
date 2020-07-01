import pathlib
from typing import Any, Dict, cast

import torch

from determined.experimental._native import _local_trial_from_context
from determined.pytorch import PyTorchTrial


def load_model(ckpt_dir: pathlib.Path, metadata: Dict[str, Any], **kwargs: Any) -> torch.nn.Module:
    trial = _local_trial_from_context(
        ckpt_dir.joinpath("code"),
        config=metadata["experiment_config"],
        hparams=metadata["hparams"],
    )

    trial = cast(PyTorchTrial, trial)
    model = trial.build_model()
    checkpoint = torch.load(ckpt_dir.joinpath("state_dict.pth"), map_location="cpu")  # type: ignore

    # TODO(DET-3456): The checkpoint schema is changed for mutliple models so this function should
    #       be updated accordingly.
    model.load_state_dict(checkpoint["models_state_dict"][0])

    return model
