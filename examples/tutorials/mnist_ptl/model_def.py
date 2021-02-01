"""
This example shows how to interact with the Determined PyTorch interface to
build a basic MNIST network.

In the `__init__` method, the model and optimizer are wrapped with `wrap_model`
and `wrap_optimizer`. This model is single-input and single-output.

The methods `train_batch` and `evaluate_batch` define the forward pass
for training and evaluation respectively.
"""

from determined.pytorch import PyTorchTrial, PyTorchTrialContext
from determined.experimental.pytorch_lightning import PTLAdapter
import ptl

class MNistTrial(PTLAdapter):
    def __init__(self, context: PyTorchTrialContext) -> None:
        super().__init__(context,
                         ptl.LightningMNISTClassifier(get_hparam=context.get_hparam),
                         data_module=ptl.MNISTDataModule())
