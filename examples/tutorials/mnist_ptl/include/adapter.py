from typing import Callable, NewType, Any
import pytorch_lightning as pl
from determined.pytorch import PyTorchTrial, PyTorchTrialContext
from determined.pytorch import DataLoader as DetDataLoader
from typing import Any, Dict, Sequence, Union
import torch

TorchData = Union[Dict[str, torch.Tensor], Sequence[torch.Tensor], torch.Tensor]

GH = NewType('GH', Callable[[str], Any])


class DETLightningModule(pl.LightningModule):
    def __init__(self, get_hparam: GH, *args, **kwargs):  # Py QUESTION should I add this is kwarg?
        super().__init__(*args, **kwargs)
        self.get_hparam = get_hparam

class DETLightningDataModule(pl.LightningDataModule):
    """
    ## user defines these as usual
    def prepare_data
    def setup

    ## user defines these for DET usage. similar to normal ptl datamodule
    def train_det_dataloader: DetDataloader
    def val_det_dataloader: DetDataloader
    def test_det_dataloader: DetDataloader

    ## user gets these for free
    def train_dataloader
    def val_dataloader
    def test_dataloader
    """
    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)


    def train_det_dataloader(self) -> DetDataLoader:
        raise NotImplementedError

    def val_det_dataloader(self) -> DetDataLoader:
        raise NotImplementedError

    # def test...

    def train_dataloader(self) -> torch.utils.data.DataLoader:
        return self.train_det_dataloader().get_data_loader()

    def val_dataloader(self) -> torch.utils.data.DataLoader:
        return self.train_det_dataloader().get_data_loader()
        



class PTLAdapter(PyTorchTrial):
    def __init__(self, context: PyTorchTrialContext, lightning_module: DETLightningModule, data_module: DETLightningDataModule = None) -> None:
        super().__init__(context)
        self.lm = lightning_module(context.get_hparam)
        self.context = context
        self.model = self.context.wrap_model(self.lm)
        self.optimizer = self.context.wrap_optimizer(self.lm.configure_optimizers())
        if data_module is not None:
            self.dm = data_module()
            # QUESTION call only on one gpu (once per node). the expected behavior could change with trainer
            # need to find a place to run this
            # https://pytorch-lightning.readthedocs.io/en/latest/api/pytorch_lightning.core.datamodule.html#pytorch_lightning.core.datamodule.LightningDataModule.prepare_data
            self.dm.prepare_data() # TODO check args
            # FIXME either self.lm or dm

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
        if self.dm is None: raise NotImplementedError()
        if not self.dm._has_setup_fit:
            self.dm.setup() # TODO call once per GPU
        return self.dm.train_det_dataloader()
    
    def build_validation_data_loader(self):
        if self.dm is None: raise NotImplementedError()
        if not self.dm._has_setup_fit:
            self.dm.setup()
        return self.dm.val_det_dataloader()
