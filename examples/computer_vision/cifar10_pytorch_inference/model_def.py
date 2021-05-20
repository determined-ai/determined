"""
Perform inference on pretrained CIFAR10 from https://github.com/huyvnphan/PyTorch_CIFAR10
"""

import tempfile
from typing import Any, Dict, Sequence, Tuple, Union, cast

import torch
import torchvision
from torch import nn
from torchvision import transforms
import torchvision.models as models

from determined.pytorch import DataLoader, PyTorchTrial, PyTorchTrialContext
import resnet

# Constants about the data set.
IMAGE_SIZE = 32
NUM_CHANNELS = 3
NUM_CLASSES = 10

TorchData = Union[Dict[str, torch.Tensor], Sequence[torch.Tensor], torch.Tensor]


def accuracy_rate(predictions: torch.Tensor, labels: torch.Tensor) -> float:
    """Return the accuracy rate based on dense predictions and sparse labels."""
    assert len(predictions) == len(labels), "Predictions and labels must have the same length."
    assert len(labels.shape) == 1, "Labels must be a column vector."

    return (  # type: ignore
        float((predictions.argmax(1) == labels.to(torch.long)).sum()) / predictions.shape[0]
    )

class CIFARTrial(PyTorchTrial):
    def __init__(self, context: PyTorchTrialContext) -> None:
        self.context = context
        self.download_directory = tempfile.mkdtemp()

        # TODO: Load your trained model
        self.model = self.context.wrap_model(resnet.resnet18(pretrained=True))

        # IGNORE: Dummy optimizer that needs to be specified but is unused
        self.optimizer = self.context.wrap_optimizer(torch.optim.RMSprop(
            self.model.parameters()))

    def train_batch(
        # IGNORE: No-op train_batch that does not train or generate metrics
        self, batch: TorchData, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        return {}

    def evaluate_batch(self, batch: TorchData) -> Dict[str, Any]:
        """
        Calculate validation metrics for a batch and return them as a dictionary.
        This method is not necessary if the user defines evaluate_full_dataset().
        """

        # TODO: Perform your evaluation step
        batch = cast(Tuple[torch.Tensor, torch.Tensor], batch)
        data, labels = batch
        output = self.model(data)

        # TODO: Optionally log or save outputs to persistent store
        print(output)
        print(labels)
        '''
        with open("/path/to/output.txt", "w+") as f:
            f.write(output)
            f.write("\n")
        '''

        # TODO: Optionally log metrics to Determined
        accuracy = accuracy_rate(output, labels)
        return {"validation_accuracy": accuracy, "validation_error": 1.0 - accuracy}

    def build_training_data_loader(self) -> Any:
        # IGNORE: Dummy training data loader that must be specified but is unused
        transform = transforms.Compose(
            [transforms.ToTensor(), transforms.Normalize((0.5, 0.5, 0.5), (0.5, 0.5, 0.5))]
        )
        trainset = torchvision.datasets.CIFAR10(
            root=self.download_directory, train=True, download=True, transform=transform
        )
        return DataLoader(trainset, batch_size=self.context.get_per_slot_batch_size())

    def build_validation_data_loader(self) -> Any:
        # TODO: Add your evaluation dataset here
        transform = transforms.Compose(
            [transforms.ToTensor(), transforms.Normalize((0.4914, 0.4822, 0.4465), (0.2471, 0.2435, 0.2616))]
        )
        valset = torchvision.datasets.CIFAR10(
            root=self.download_directory, train=False, download=True, transform=transform
        )

        return DataLoader(valset, batch_size=self.context.get_per_slot_batch_size())
