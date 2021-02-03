"""
This example shows how to interact with the Determined PyTorch interface to
build a basic MNIST network.

In the `__init__` method, the model and optimizer are wrapped with `wrap_model`
and `wrap_optimizer`. This model is single-input and single-output.

The methods `train_batch` and `evaluate_batch` define the forward pass
for training and evaluation respectively.
"""

from determined.pytorch import PyTorchTrial, PyTorchTrialContext, DataLoader as DetDataLoader
from determined.experimental.pytorch_lightning import PTLAdapter
import ptl

class MNistTrial(PTLAdapter):
    def __init__(self, context: PyTorchTrialContext) -> None:
        super().__init__(context,
                         ptl.LightningMNISTClassifier(get_hparam=context.get_hparam))

        self.dm = ptl.MNISTDataModule()

    def build_training_data_loader(self) -> DetDataLoader:
        self.dm.setup()
        dl = self.dm.train_dataloader()
        # if we can support something like this more comprehensivly that'll give us a way to
        # directly support pytorch datalaoder => det dataloader and thus lightning datamodule
        return DetDataLoader(dl.dataset, batch_size=dl.batch_size, num_workers=dl.num_workers)

    def build_validation_data_loader(self) -> DetDataLoader:
        self.dm.setup()
        dl = self.dm.val_dataloader()
        return DetDataLoader(dl.dataset, batch_size=dl.batch_size, num_workers=dl.num_workers)
