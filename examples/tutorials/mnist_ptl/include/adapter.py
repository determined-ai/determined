from typing import Callable, NewType, Any
import pytorch_lightning as pl
from determined.pytorch import DataLoader, PyTorchTrial, PyTorchTrialContext
from typing import Any, Dict, Sequence, Union
import torch

TorchData = Union[Dict[str, torch.Tensor], Sequence[torch.Tensor], torch.Tensor]

GH = NewType('GH', Callable[[str], Any])


class DETLightningModule(pl.LightningModule):
    def __init__(self, get_hparam: GH, *args, **kwargs):  # Py QUESTION should I add this is kwarg?
        super().__init__(*args, **kwargs)
        self.get_hparam = get_hparam



class PTLAdapter(PyTorchTrial):
    def __init__(self, context: PyTorchTrialContext, lightning_module: DETLightningModule, data_module: pl.LightningDataModule = None) -> None:
        super().__init__(context)
        self.lm = lightning_module(context.get_hparam)
        self.context = context
        self.model = self.context.wrap_model(self.lm)
        self.optimizer = self.context.wrap_optimizer(self.lm.configure_optimizers())

        # set the data module
        if data_module is not None:
            print(data_module)
            self.lm.datamodule = data_module
        if self.lm.datamodule is not None:
            # QUESTION call only on one gpu (once per node). the expected behavior could change with trainer
            # need to find a place to run this
            # https://pytorch-lightning.readthedocs.io/en/latest/api/pytorch_lightning.core.datamodule.html#pytorch_lightning.core.datamodule.LightningDataModule.prepare_data
            self.lm.prepare_data() # TODO check args

    def train_batch(
        self, batch: TorchData, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        rv = self.lm.training_step(batch, batch_idx)

        # TODO option to set loss
        self.context.backward(rv['loss'])
        self.context.step_optimizer(self.optimizer)
        return rv

    def evaluate_batch(self, batch: TorchData) -> Dict[str, Any]:
        return self.lm.validation_step(batch)


    def build_training_data_loader(self):
        dm = self.lm.datamodule
        if dm is None: raise NotImplementedError()
        if not dm._has_setup_fit:
            dm.setup() # TODO call once per GPU
        return dm.train_dataloader()
    
    def build_validation_data_loader(self):
        dm = self.lm.datamodule
        if dm is None: raise NotImplementedError()
        if not dm._has_setup_fit:
            dm.setup()
        return dm.train_dataloader()
