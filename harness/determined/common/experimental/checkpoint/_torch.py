import pathlib
from typing import Any, Dict, Union, cast

import torch

from determined import errors, experimental, util
from determined.pytorch import PyTorchTrial, PyTorchTrialContext


def load_model(
    ckpt_dir: pathlib.Path, metadata: Dict[str, Any], **kwargs: Any
) -> Union[PyTorchTrial, torch.nn.Module]:
    checkpoint = torch.load(str(ckpt_dir.joinpath("state_dict.pth")), **kwargs)  # type: ignore

    trial_cls, trial_context = experimental._load_trial_for_checkpoint_export(
        ckpt_dir.joinpath("code"),
        managed_training=False,
        config=metadata["experiment_config"],
        hparams=metadata["hparams"],
    )

    trial_context = cast(PyTorchTrialContext, trial_context)
    trial = cast(PyTorchTrial, trial_cls(trial_context))
    if "model_state_dict" in checkpoint:
        # Backward compatible with older checkpoint
        model_func = util.get_member_func(trial, "build_model")
        if model_func is not None:
            model = cast(torch.nn.Module, model_func())
            model.load_state_dict(checkpoint["model_state_dict"])
            return model
        raise errors.InvalidCheckpointException()
    else:
        # Backward compatible with older checkpoint
        model_func = util.get_member_func(trial, "build_model")
        if model_func is not None:
            model = cast(torch.nn.Module, model_func())
            model.load_state_dict(checkpoint["models_state_dict"][0])
            return model
        else:
            for idx, model in enumerate(trial_context.models):
                model.load_state_dict(checkpoint["models_state_dict"][idx])
            return trial
