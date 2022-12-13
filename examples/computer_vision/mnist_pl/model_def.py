"""
This example shows how to interact with the Determined PyTorch Lightning Adapter
interface to build a basic MNIST network. LightningAdapter utilizes the provided
LightningModule with Determined's PyTorch control loop.
"""

import data
import mnist

from determined.pytorch import DataLoader, PyTorchTrialContext
from determined.pytorch.lightning import LightningAdapter


class MNISTTrial(LightningAdapter):
    def __init__(self, context: PyTorchTrialContext, *args, **kwargs) -> None:
        lm = mnist.LitMNIST(
            hidden_size=context.get_hparam("hidden_size"),
            learning_rate=context.get_hparam("learning_rate"),
        )
        data_dir = f"/tmp/data-rank{context.distributed.get_rank()}"
        self.dm = data.MNISTDataModule(
            data_url=context.get_data_config()["url"],
            data_dir=data_dir,
            batch_size=context.get_per_slot_batch_size(),
        )

        super().__init__(context, lightning_module=lm, *args, **kwargs)
        self.dm.prepare_data()

    def build_training_data_loader(self) -> DataLoader:
        self.dm.setup()
        dl = self.dm.train_dataloader()
        return DataLoader(dl.dataset, batch_size=dl.batch_size, num_workers=dl.num_workers)

    def build_validation_data_loader(self) -> DataLoader:
        self.dm.setup()
        dl = self.dm.val_dataloader()
        return DataLoader(dl.dataset, batch_size=dl.batch_size, num_workers=dl.num_workers)
