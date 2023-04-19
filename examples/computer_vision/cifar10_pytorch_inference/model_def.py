"""
Perform inference on pretrained CIFAR10 from https://github.com/huyvnphan/PyTorch_CIFAR10
"""

import os
import tempfile
from typing import Any, Dict, Sequence, Tuple, Union, cast

import numpy as np
import resnet
import torch
import torchvision
import torchvision.models as models
from torch import nn
from torchvision import transforms

from determined.pytorch import DataLoader, MetricReducer, PyTorchTrial, PyTorchTrialContext

# Constants about the data set.
IMAGE_SIZE = 32
NUM_CHANNELS = 3
NUM_CLASSES = 10

TorchData = Union[Dict[str, torch.Tensor], Sequence[torch.Tensor], torch.Tensor]


def set_parameter_requires_grad(model, feature_extracting):
    if feature_extracting:
        for param in model.parameters():
            param.requires_grad = False


class PredictionsReducer(MetricReducer):
    def __init__(self, output_file):
        self.num_classes = 10
        self.reset()
        self.output_file = output_file

    def reset(self):
        # reset() will be called before each training and validation workload.
        self.predictions = []

    def update(self, predictions):
        # We are responsible for calling update() as part of our train_batch() and evaluate_batch()
        # methods, which means we can specify any arguments we wish.
        self.predictions += predictions.tolist()

    def per_slot_reduce(self):
        return self.predictions

    def cross_slot_reduce(self, per_slot_metrics):
        # TODO: Log or save outputs to persistent store
        predictions = [p for slot_predictions in per_slot_metrics for p in slot_predictions]
        np.save(self.output_file, predictions)

        return {}


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

        # TODO: Load your trained model. Below are example approaches.

        ### Load a checkpoint from the Determined model registry
        # from determined.experimental import client
        # client.login()
        # self.model = client.get_model("mymodel")
        # ckpt_path = self.model.get_version().download()
        # ckpt = torch.load(os.path.join(ckpt_path, 'state_dict.pth'))
        # model.load_state_dict(ckpt['models_state_dict'][0])

        ### Load a checkpoint from a previous experiment
        # from determined.experimental import client
        # client.login()
        # checkpoint = client.get_experiment(id).top_checkpoint()
        # model = checkpoint.load()

        ### Load a model that was not trained by Determined
        self.model = self.context.wrap_model(resnet.resnet18(pretrained=True))

        # IGNORE: Dummy optimizer that needs to be specified but is unused
        self.optimizer = self.context.wrap_optimizer(torch.optim.RMSprop(self.model.parameters()))

        # TODO: Create custom reducer to save inference output
        output_file = os.path.join(self.download_directory, "predictions.npy")
        self.predictions = self.context.wrap_reducer(PredictionsReducer(output_file))

    def train_batch(
        # IGNORE: No-op train_batch that does not train or generate metrics
        self,
        batch: TorchData,
        epoch_idx: int,
        batch_idx: int,
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

        # Log predictions to our custom reducer for aggregation
        self.predictions.update(output.argmax(dim=1))

        # TODO: Optionally log metrics to Determined
        accuracy = accuracy_rate(output, labels)
        return {"validation_accuracy": accuracy, "validation_error": 1.0 - accuracy}

    def build_training_data_loader(self) -> Any:
        # IGNORE: Dummy training data loader that must be specified but is unused
        transform = transforms.Compose(
            [
                transforms.ToTensor(),
                transforms.Normalize((0.5, 0.5, 0.5), (0.5, 0.5, 0.5)),
            ]
        )
        trainset = torchvision.datasets.CIFAR10(
            root=self.download_directory, train=True, download=True, transform=transform
        )
        return DataLoader(trainset, batch_size=self.context.get_per_slot_batch_size())

    def build_validation_data_loader(self) -> Any:
        # TODO: Add your evaluation dataset here
        transform = transforms.Compose(
            [
                transforms.ToTensor(),
                transforms.Normalize((0.4914, 0.4822, 0.4465), (0.2471, 0.2435, 0.2616)),
            ]
        )
        valset = torchvision.datasets.CIFAR10(
            root=self.download_directory,
            train=False,
            download=True,
            transform=transform,
        )

        return DataLoader(valset, batch_size=self.context.get_per_slot_batch_size())
