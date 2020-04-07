"""
CNN on Cifar10 from Keras example:
https://github.com/fchollet/keras/blob/master/examples/cifar10_cnn.py
- const.yaml, ~50% accuracy after 50 steps
- adaptive.yaml, accuracy comparable to Keras example after 50 steps
"""

from typing import Any, Dict, Sequence, Tuple, Union, cast

import torch
import torchvision
from torch import nn
from torchvision import transforms

import determined as det
from determined.pytorch import DataLoader, PyTorchTrial, reset_parameters

# Constants about the data set.
IMAGE_SIZE = 32
NUM_CHANNELS = 3
NUM_CLASSES = 10

TorchData = Union[Dict[str, torch.Tensor], Sequence[torch.Tensor], torch.Tensor]


def error_rate(predictions: torch.Tensor, labels: torch.Tensor) -> float:
    """Return the error rate based on dense predictions and dense labels."""
    assert len(predictions) == len(labels), "Predictions and labels must have the same length."
    assert len(labels.shape) == 1, "Labels must be a column vector."

    return (  # type: ignore
        1.0 - float((predictions.argmax(1) == labels.to(torch.long)).sum()) / predictions.shape[0]
    )


class Flatten(nn.Module):
    def forward(self, *args: TorchData, **kwargs: Any) -> torch.Tensor:
        assert len(args) == 1
        x = args[0]
        assert isinstance(x, torch.Tensor)
        return x.view(x.size(0), -1)


class CIFARTrial(PyTorchTrial):
    def __init__(self, context: det.TrialContext) -> None:
        self.context = context

    def build_model(self) -> nn.Module:
        model = nn.Sequential(
            nn.Conv2d(NUM_CHANNELS, IMAGE_SIZE, kernel_size=(3, 3)),
            nn.ReLU(),
            nn.Conv2d(32, 32, kernel_size=(3, 3)),
            nn.ReLU(),
            nn.MaxPool2d((2, 2)),
            nn.Dropout2d(self.context.get_hparam("layer1_dropout")),
            nn.Conv2d(32, 64, (3, 3), padding=1),
            nn.ReLU(),
            nn.Conv2d(64, 64, (3, 3)),
            nn.ReLU(),
            nn.MaxPool2d((2, 2)),
            nn.Dropout2d(self.context.get_hparam("layer2_dropout")),
            Flatten(),
            nn.Linear(2304, 512),
            nn.ReLU(),
            nn.Dropout2d(self.context.get_hparam("layer3_dropout")),
            nn.Linear(512, NUM_CLASSES),
            nn.Softmax(dim=0),
        )

        # If loading backbone weights, do not call reset_parameters() or
        # call before loading the backbone weights.
        reset_parameters(model)
        return model

    def optimizer(self, model: nn.Module) -> torch.optim.Optimizer:  # type: ignore
        return torch.optim.RMSprop(  # type: ignore
            model.parameters(),
            lr=self.context.get_hparam("learning_rate"),
            weight_decay=self.context.get_hparam("learning_rate_decay"),
            alpha=0.9,
        )

    def train_batch(
        self, batch: TorchData, model: nn.Module, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        batch = cast(Tuple[torch.Tensor, torch.Tensor], batch)
        data, labels = batch

        output = model(data)
        loss = torch.nn.functional.nll_loss(output, labels)
        error = error_rate(output, labels)
        return {"loss": loss, "train_error": error}

    def evaluate_batch(self, batch: TorchData, model: nn.Module) -> Dict[str, Any]:
        """
        Calculate validation metrics for a batch and return them as a dictionary.
        This method is not necessary if the user overwrites evaluate_full_dataset().
        """
        batch = cast(Tuple[torch.Tensor, torch.Tensor], batch)
        data, labels = batch

        output = model(data)
        error = error_rate(output, labels)
        return {"validation_error": error}

    def build_training_data_loader(self) -> Any:
        # Create a unique download directory for each rank so they don't overwrite each other.
        download_directory = f"./data-rank{self.context.distributed.get_rank()}"

        transform = transforms.Compose(
            [transforms.ToTensor(), transforms.Normalize((0.5, 0.5, 0.5), (0.5, 0.5, 0.5))]
        )
        trainset = torchvision.datasets.CIFAR10(
            root=download_directory, train=False, download=True, transform=transform
        )
        return DataLoader(trainset, batch_size=self.context.get_per_slot_batch_size())

    def build_validation_data_loader(self) -> Any:
        # Create a unique download directory for each rank so they don't overwrite each other.
        download_directory = f"./data-rank{self.context.distributed.get_rank()}"

        transform = transforms.Compose(
            [transforms.ToTensor(), transforms.Normalize((0.5, 0.5, 0.5), (0.5, 0.5, 0.5))]
        )
        valset = torchvision.datasets.CIFAR10(
            root=download_directory, train=False, download=True, transform=transform
        )

        return DataLoader(valset, batch_size=self.context.get_per_slot_batch_size())
