import pathlib
from typing import Any, Dict, Union, cast

import torch

from determined import experimental, util
from determined.pytorch import PyTorchTrial, PyTorchTrialContext


def load_model(
    ckpt_dir: pathlib.Path, metadata: Dict[str, Any], **kwargs: Any
) -> Union[PyTorchTrial, torch.nn.Module]:
    checkpoint = torch.load(str(ckpt_dir.joinpath("state_dict.pth")), **kwargs)  # type: ignore

    trial_cls, trial_context = experimental._load_trial_on_local(
        ckpt_dir.joinpath("code"),
        managed_training=False,
        config=metadata["experiment_config"],
        hparams=metadata["hparams"],
    )

    trial_context = cast(PyTorchTrialContext, trial_context)
    trial = cast(PyTorchTrial, trial_cls(trial_context))
    if "model_state_dict" in checkpoint:
        # Backward compatible with older checkpoint format.
        model = trial.build_model()
        model.load_state_dict(checkpoint["model_state_dict"])
        return model
    else:
        # Backward compatible with older interface
        if util.is_overridden(trial.build_model, PyTorchTrial):
            model = trial.build_model()
            model.load_state_dict(checkpoint["models_state_dict"][0])
            return model
        else:
            for idx, model in enumerate(trial_context.models):
                model.load_state_dict(checkpoint["models_state_dict"][idx])
            return trial
