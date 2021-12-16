"""
This example shows how to interact with the Determined PyTorch Lightning Adapter
interface to build a basic MNIST network. LightningAdapter utilizes the provided
LightningModule with Determined's PyTorch control loop.
"""

from determined.pytorch import PyTorchTrialContext, DataLoader
from determined.pytorch.lightning import LightningAdapter
from pytorch_lightning import LightningModule
import torchvision

import data

class SampleClassifier(LightningModule):
    def __init__(self):
        super().__init__()
        self.model = torchvision.models.detection.maskrcnn_resnet50_fpn(pretrained=True, progress = False)

    def forward(x, y):
        return self.model(x, y)

class TorchGeoTrial(LightningAdapter):
    def __init__(self, context: PyTorchTrialContext, *args, **kwargs) -> None:
        data_dir = f"/tmp/data-rank{context.distributed.get_rank()}"
        self.dm = data.NAIPDataModule(
            data_dir=data_dir,
            batch_size=context.get_per_slot_batch_size(),
        )

        lm = SampleClassifier()

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
