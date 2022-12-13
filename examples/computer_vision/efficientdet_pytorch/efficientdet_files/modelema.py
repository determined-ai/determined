"""
This is based on pytorch-image-models' ModelEMA
https://github.com/rwightman/pytorch-image-models/blob/9a25fdf3ad0414b4d66da443fe60ae0aa14edc84/timm/utils/model_ema.py

This altered version refactors the load and save functionality to support Determined's fault tolerance features.
"""
from collections import OrderedDict
from copy import deepcopy
from typing import Any, Dict, Sequence, Tuple, Union, cast

import torch

from determined.pytorch import PyTorchCallback


class ModelEma:
    """Model Exponential Moving Average
    Keep a moving average of everything in the model state_dict (parameters and buffers).
    This is intended to allow functionality like
    https://www.tensorflow.org/api_docs/python/tf/train/ExponentialMovingAverage
    A smoothed version of the weights is necessary for some training schemes to perform well.
    E.g. Google's hyper-params for training MNASNet, MobileNet-V3, EfficientNet, etc that use
    RMSprop with a short 2.4-3 epoch decay period and slow LR decay rate of .96-.99 requires EMA
    smoothing of weights to match results. Pay attention to the decay constant you are using
    relative to your update count per epoch.
    To keep EMA from using GPU resources, set device='cpu'. This will save a bit of memory but
    disable validation of the EMA weights. Validation will have to be done manually in a separate
    process, or after the training stops converging.
    This class is sensitive where it is initialized in the sequence of model init,
    GPU assignment and distributed training wrappers.
    I've tested with the sequence in my own train.py for torch.DataParallel, apex.DDP, and single-GPU.
    """

    def __init__(self, model, decay=0.9999, context="", resume=""):
        # make a copy of the model for accumulating moving average of weights
        self.ema = deepcopy(model)
        self.ema.eval()
        self.decay = decay
        self.context = context
        self.ema_has_module = hasattr(self.ema, "module")
        if resume:
            self._load_checkpoint(resume)
        for p in self.ema.parameters():
            p.requires_grad_(False)

    def callback_object(self):
        class Emacallback(PyTorchCallback):
            def state_dict(this) -> Dict[str, Any]:
                return {"model": self.ema.state_dict()}

            def load_state_dict(this, state_dict: Dict[str, Any]) -> None:
                self.ema.load_state_dict(state_dict["model"])

        return Emacallback()

    def _load_checkpoint(self, checkpoint_path):
        checkpoint = torch.load(checkpoint_path, map_location="cpu")
        assert isinstance(checkpoint, dict)
        if "state_dict_ema" in checkpoint:
            new_state_dict = OrderedDict()
            for k, v in checkpoint["state_dict_ema"].items():
                # ema model may have been wrapped by DataParallel, and need module prefix
                if self.ema_has_module:
                    name = "module." + k if not k.startswith("module") else k
                else:
                    name = k
                new_state_dict[name] = v
            self.ema.load_state_dict(new_state_dict)

    def update(self, model):
        # correct a mismatch in state dict keys
        needs_module = hasattr(model, "module") and not self.ema_has_module
        with torch.no_grad():
            msd = model.state_dict()
            for k, ema_v in self.ema.state_dict().items():
                if needs_module:
                    k = "module." + k
                model_v = msd[k].detach()
                model_v = self.context.to_device(model_v)
                ema_v.copy_(ema_v * self.decay + (1.0 - self.decay) * model_v)
