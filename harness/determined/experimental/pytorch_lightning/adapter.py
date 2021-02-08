import sys
from inspect import signature
from typing import Any, Callable, Dict, NewType, Sequence, Union

import pytorch_lightning as ptl
import torch

from determined import monkey_patch
from determined.experimental.pytorch_lightning.data_module import DETLightningDataModule
from determined.pytorch import PyTorchTrial, PyTorchTrialContext

TorchData = Union[Dict[str, torch.Tensor], Sequence[torch.Tensor], torch.Tensor]
HyperparamsProvider = Callable[[str], Any]


def bail(msg: str = "", fail: bool = True):
    msg = f"NotSupported: {msg}"
    if fail:
        raise TypeError(msg)
    else:
        # TODO use a logger
        print(msg, file=sys.stderr)


# class DETLightningModule(ptl.LightningModule):
#     """
#     DETLightningModule helps us dictate what extra inputs the user's lightning module should expect.
#     Aleternatively we can avoid this and have the user take care of it.
#     """
#     def __init__(self, *args, get_hparam: HyperparamsProvider = None, **kwargs):
#         super().__init__(*args, **kwargs)
#         self.get_hparam = get_hparam
#         check_compat(self)


def check_compat(lm: ptl.LightningModule):
    if len(signature(lm.training_step).parameters) > 2:
        bail("`optimizer_idx` and `hiddens` are not supported.")
    if len(signature(lm.validation_step).parameters) > 2:
        bail("`dataloader_idx` is not supported.")


class PTLAdapter(PyTorchTrial):
    context: PyTorchTrialContext

    # QUESTION: take uninstantiated lightning and datamodule so we isntatiate it instead? less code for the user but might be better if the user sees this?
    def __init__(self, context: PyTorchTrialContext, lightning_module: ptl.LightningModule):
        super().__init__(context)
        check_compat(lightning_module)
        self.lm = lightning_module
        self.context = context
        self.model = self.context.wrap_model(self.lm)

        # pass context here?
        # returns instantiated lrscheduler and optimizers. link to wrapx
        # lrschduler is initialized with params from optimizer.?
        optimizer = self.lm.configure_optimizers()
        """
        LM optimizer
        - Single optimizer.
        - List or Tuple - List of optimizers.
        - Two lists - The first list has multiple optimizers, the second a list of LR schedulers (or lr_dict).
        - Dictionary, with an ‘optimizer’ key, and (optionally) a ‘lr_scheduler’ key whose value is a single LR scheduler or lr_dict.
        - Tuple of dictionaries as described, with an optional ‘frequency’ key.
        - None - Fit will run without any optimizer.
        """
        if not isinstance(optimizer, torch.optim.Optimizer):
            bail("currently only returning a single optimizer is supported")
        # TODO look at how we wrap/create learning scheduler
        # currently this is only supporting a single optimizer
        self.optimizer = self.context.wrap_optimizer(optimizer)

    def train_batch(
        self, batch: TorchData, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        # no optimizer index to pass down
        # TODO step through all the optimizers freeze other models? check trainer source code
        rv = self.lm.training_step(batch, batch_idx)
        if rv is None:
            return {}  # skip to next batch
        if type(rv) != dict:
            rv = {"loss": rv}

        self.context.backward(rv["loss"])
        self.context.step_optimizer(self.optimizer)
        return rv

    def evaluate_batch(self, batch: TorchData) -> Dict[str, Any]:
        return self.lm.validation_step(batch)
