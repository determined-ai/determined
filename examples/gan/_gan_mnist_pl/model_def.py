"""
This example shows how to interact with the Determined PyTorch Lightning Adapter
interface to build a basic MNIST network. The PLAdapter utilizes the provided
LightningModule with Determined's trainer.
"""

from determined.pytorch import PyTorchTrial, PyTorchTrialContext, DataLoader
from determined.pytorch._lightning import PLAdapter
import gan

class GANTrial(PLAdapter):
    def __init__(self, context: PyTorchTrialContext) -> None:
        self.dm = gan.MNISTDataModule(batch_size=context.get_global_batch_size())
        channels, width, height = self.dm.size()
        lm = gan.GAN(channels, width, height,
                    batch_size=context.get_global_batch_size(),
                    lr=context.get_hparam('lr'),
                    b1=context.get_hparam('b1'),
                    b2=context.get_hparam('b2'),
        )

        super().__init__(context, lightning_module=lm)
        self.dm.prepare_data()

    def build_training_data_loader(self) -> DataLoader:
        self.dm.setup()
        dl = self.dm.train_dataloader()
        return DataLoader(dl.dataset, batch_size=dl.batch_size, num_workers=dl.num_workers)

    def build_validation_data_loader(self) -> DataLoader:
        self.dm.setup()
        dl = self.dm.val_dataloader()
        return DataLoader(dl.dataset, batch_size=dl.batch_size, num_workers=dl.num_workers)
