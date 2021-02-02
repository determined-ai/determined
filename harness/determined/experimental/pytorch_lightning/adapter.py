from typing import Callable, NewType, Any
import pytorch_lightning as ptl
from determined.pytorch import PyTorchTrial, PyTorchTrialContext
from determined.experimental.pytorch_lightning.data_module import DETLightningDataModule
from determined import monkey_patch
from typing import Any, Dict, Sequence, Union
from inspect import signature
import torch
import sys

TorchData = Union[Dict[str, torch.Tensor], Sequence[torch.Tensor], torch.Tensor]
HyperparamsProvider = Callable[[str], Any]

def bail(msg: str = '', fail: bool = True):
    msg = f'NotSupported: {msg}'
    if fail:
        raise TypeError(msg)
    else:
        # TODO use a logger
        print(msg, file=sys.stderr)


class DETLightningModule(ptl.LightningModule):
    """
    DETLightningModule helps us dictate what extra inputs the user's lightning module should expect.
    Aleternatively we can avoid this and have the user take care of it.
    """
    def __init__(self, *args, get_hparam: HyperparamsProvider = None, **kwargs):
        super().__init__(*args, **kwargs)
        self.get_hparam = get_hparam
        check_compat(self)


def check_compat(lm: DETLightningModule):
    if len(signature(lm.training_step).parameters) > 2:
        bail('`optimizer_idx` and `hiddens` are not supported.')
    if len(signature(lm.validation_step).parameters) > 2:
        bail('`dataloader_idx` is not supported.')


class PTLAdapter(PyTorchTrial):
    # QUESTION: take uninstantiated lightning and datamodule so we isntatiate it instead? less code for the user but might be better if the user sees this?
    def __init__(self, context: PyTorchTrialContext, lightning_module: DETLightningModule, data_module: DETLightningDataModule = None):
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
            bail('currently only returning a single optimizer is supported')
        # TODO look at how we wrap/create learning scheduler
        # currently this is only supporting a single optimizer
        self.optimizer = self.context.wrap_optimizer(optimizer)

        if data_module is not None:
            self.dm = data_module
            # QUESTION call only on one gpu (once per node). the expected behavior could change with trainer
            # need to find a place to run this
            # https://pytorch-lightning.readthedocs.io/en/latest/api/pytorch_lightning.core.datamodule.html#pytorch_lightning.core.datamodule.LightningDataModule.prepare_data
            # there are some methods on lm that overlaps with dm
            self.dm.prepare_data() # TODO check args

    def train_batch(
        self, batch: TorchData, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        # no optimizer index to pass down
        # TODO step through all the optimizers freeze other models? check trainer source code
        rv = self.lm.training_step(batch, batch_idx)
        if rv is None:  return {} # skip to next batch
        if type(rv) != dict:
            rv = {'loss': rv}

        self.context.backward(rv['loss'])
        self.context.step_optimizer(self.optimizer)
        return rv

    def evaluate_batch(self, batch: TorchData) -> Dict[str, Any]:
        return self.lm.validation_step(batch)


    def build_training_data_loader(self):
        if self.dm is None: raise NotImplementedError()
        if not self.dm._has_setup_fit:
            self.dm.setup()
        return self.dm.train_det_dataloader()

    def build_validation_data_loader(self):
        if self.dm is None: raise NotImplementedError()
        if not self.dm._has_setup_fit:
            self.dm.setup()
        return self.dm.val_det_dataloader()
