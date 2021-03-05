"""
This example shows how to interact with the Determined PyTorch Lightning Adapter
interface to build a basic MNIST network. The PLAdapter utilizes the provided
LightningModule with Determined's PyTorch control loop.
"""

from determined.pytorch import PyTorchTrial, PyTorchTrialContext, DataLoader
from determined.pytorch._lightning import PLAdapter
import mnist

class MNISTTrial(PLAdapter):
    def __init__(self, context: PyTorchTrialContext) -> None:
        lm = mnist.LightningMNISTClassifier(lr=context.get_hparam('learning_rate'))
        data_dir = f"/tmp/data-rank{context.distributed.get_rank()}"
        self.dm = mnist.MNISTDataModule(context.get_data_config()["url"],
                                        data_dir)

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
