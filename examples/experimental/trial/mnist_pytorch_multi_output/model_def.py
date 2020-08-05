"""
This example shows how to interact with the Determined PyTorch interface to build a
multi prediction MNIST network.

The `MultiMNistTrial` class contains methods for building the model, building the
optimizer, and defining the forward pass for training and validation.
"""

from typing import Any, Dict, Tuple, cast

import torch
from torch import nn

import determined as det
from layers import Flatten, Squeeze  # noqa: I100
from determined.pytorch import DataLoader, PyTorchTrial, PyTorchTrialContext, TorchData, reset_parameters

import data


def error_rate(predictions: torch.Tensor, labels: torch.Tensor) -> float:
    """Return the error rate based on dense predictions and dense labels."""
    assert len(predictions) == len(labels), "Predictions and labels must have the same length."
    assert len(labels.shape) == 1, "Labels must be a column vector."

    return (  # type: ignore
        1.0 - float((predictions.argmax(1) == labels.to(torch.long)).sum()) / predictions.shape[0]
    )


class MultiNet(nn.Module):
    """
    MNIST network that takes
    input: data
    output: digit predictions, binary predictions
    """

    def __init__(self, context: det.TrialContext) -> None:
        super().__init__()
        # Set hyperparameters that influence the model architecture.
        self.n_filters1 = context.get_hparam("n_filters1")
        self.n_filters2 = context.get_hparam("n_filters2")
        self.dropout = context.get_hparam("dropout")

        # Define the central model.
        self.model = nn.Sequential(
            nn.Conv2d(1, self.n_filters1, kernel_size=5),
            nn.MaxPool2d(2),
            nn.ReLU(),
            nn.Conv2d(self.n_filters1, self.n_filters2, kernel_size=5),
            nn.MaxPool2d(2),
            nn.ReLU(),
            Flatten(),
            nn.Linear(16 * self.n_filters2, 50),
            nn.ReLU(),
            nn.Dropout2d(self.dropout),
        )  # type: nn.Sequential
        # Predict digit labels from self.model.
        self.digit = nn.Sequential(nn.Linear(50, 10), nn.Softmax(dim=0))
        # Predict binary labels from self.model.
        self.binary = nn.Sequential(nn.Linear(50, 1), nn.Sigmoid(), Squeeze())

    def forward(self, *args: TorchData, **kwargs: Any) -> TorchData:
        assert len(args) == 1
        assert isinstance(args[0], dict)
        # The name "data" is defined by the return value of the
        # `MultiMNistPyTorchDatasetAdapter.get_batch()` method.
        model_out = self.model(args[0]["data"])

        # Define two prediction outputs for a multi-output network.
        digit_predictions = self.digit(model_out)
        binary_predictions = self.binary(model_out)

        # Return the two outputs as a dict of outputs. This dict will become
        # the `predictions` input to the `MultiMNistTrial.losses()` function.
        return {"digit_predictions": digit_predictions, "binary_predictions": binary_predictions}


def compute_loss(predictions: TorchData, labels: TorchData) -> torch.Tensor:
    assert isinstance(predictions, dict)
    assert isinstance(labels, dict)

    labels["binary_labels"] = labels["binary_labels"].type(torch.float32)  # type: ignore

    # Calculate loss functions.
    loss_digit = torch.nn.functional.nll_loss(
        predictions["digit_predictions"], labels["digit_labels"]
    )
    loss_binary = torch.nn.functional.binary_cross_entropy(
        predictions["binary_predictions"], labels["binary_labels"]
    )

    # Rudimentary example of how loss functions may be combined for
    # multi-output training.
    loss = loss_binary + loss_digit
    return loss


class MultiMNistTrial(PyTorchTrial):
    def __init__(self, context: PyTorchTrialContext) -> None:
        self.context = context

        # Create a unique download directory for each rank so they don't overwrite each other.
        self.download_directory = f"/tmp/data-rank{self.context.distributed.get_rank()}"
        self.data_downloaded = False

        self.model = self.context.wrap_model(MultiNet(self.context))

        # If loading backbone weights, do not call reset_parameters() or
        # call before loading the backbone weights.
        reset_parameters(self.model)

        self.optimizer = self.context.wrap_optimizer(torch.optim.SGD(
            self.model.parameters(), lr=self.context.get_hparam("learning_rate"), momentum=0.9
        ))

    def build_training_data_loader(self) -> DataLoader:
        if not self.data_downloaded:
            self.download_directory = data.download_dataset(
                download_directory=self.download_directory,
                data_config=self.context.get_data_config(),
            )
            self.data_downloaded = True

        train_data = data.get_multi_dataset(self.download_directory, train=True)
        return DataLoader(
            train_data,
            batch_size=self.context.get_per_slot_batch_size(),
            collate_fn=data.collate_fn,
        )

    def build_validation_data_loader(self) -> DataLoader:
        if not self.data_downloaded:
            self.download_directory = data.download_dataset(
                download_directory=self.download_directory,
                data_config=self.context.get_data_config(),
            )
            self.data_downloaded = True

        validation_data = data.get_multi_dataset(self.download_directory, train=False)
        return DataLoader(
            validation_data,
            batch_size=self.context.get_per_slot_batch_size(),
            collate_fn=data.collate_fn,
        )

    def train_batch(self, batch: Any, epoch_idx: int, batch_idx: int) -> Dict[str, torch.Tensor]:
        batch = cast(Tuple[TorchData, Dict[str, torch.Tensor]], batch)
        data, labels = batch

        output = self.model(data)
        loss = compute_loss(output, labels)
        error = error_rate(output["digit_predictions"], labels["digit_labels"])

        self.context.backward(loss)
        self.context.step_optimizer(self.optimizer)

        return {"loss": loss, "classification_error": error}

    def evaluate_batch(self, batch: Any) -> Dict[str, Any]:
        batch = cast(Tuple[TorchData, Dict[str, torch.Tensor]], batch)
        data, labels = batch

        output = self.model(data)
        error = error_rate(output["digit_predictions"], labels["digit_labels"])

        return {"validation_error": error}
