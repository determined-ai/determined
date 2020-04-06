"""
This example shows how to interact with the Determined PyTorch interface to
build a basic MNIST network.

The method `build_model` returns the model to be trained, in this case an
instance of `nn.Sequential`. This model is single-input and single-output. For
an example of a multi-output model, see the `build_model` method in the
definition of `MultiMNistTrial` in model_def_multi_output.py. In that case,
`build_model` returns an instance of a custom `nn.Module`.

Predictions are the output of the `forward` method of the model (for
`nn.Sequential`, that is automatically defined). The predictions are then fed
directly into the `losses` method and the `validation_metrics` method.

The method `MNistTrial.losses` calculates the loss of the training, which for
this model is a single tensor value.

The output of `losses` is then fed directly into `validation_metrics`, which
returns a dictionary mapping metric names to metric values.
"""

from typing import Any, Dict, Sequence, Tuple, Union, cast

import torch
from torch import nn

from layers import Flatten  # noqa: I100

import determined as det
from determined.pytorch import DataLoader, PyTorchTrial, reset_parameters

import data

TorchData = Union[Dict[str, torch.Tensor], Sequence[torch.Tensor], torch.Tensor]


def error_rate(predictions: torch.Tensor, labels: torch.Tensor) -> float:
    """Return the error rate based on dense predictions and dense labels."""
    assert len(predictions) == len(labels), "Predictions and labels must have the same length."
    assert len(labels.shape) == 1, "Labels must be a column vector."

    return (  # type: ignore
        1.0 - float((predictions.argmax(1) == labels.to(torch.long)).sum()) / predictions.shape[0]
    )


class MNistTrial(PyTorchTrial):
    def __init__(self, context: det.TrialContext) -> None:
        self.context = context

        # Create a unique download directory for each rank so they don't overwrite each other.
        self.download_directory = f"/tmp/data-rank{self.context.distributed.get_rank()}"
        self.data_downloaded = False

    def build_training_data_loader(self) -> DataLoader:
        if not self.data_downloaded:
            self.download_directory = data.download_dataset(
                download_directory=self.download_directory,
                data_config=self.context.get_data_config(),
            )
            self.data_downloaded = True

        train_data = data.get_dataset(self.download_directory, train=True)
        return DataLoader(train_data, batch_size=self.context.get_per_slot_batch_size())

    def build_validation_data_loader(self) -> DataLoader:
        if not self.data_downloaded:
            self.download_directory = data.download_dataset(
                download_directory=self.download_directory,
                data_config=self.context.get_data_config(),
            )
            self.data_downloaded = True

        validation_data = data.get_dataset(self.download_directory, train=False)
        return DataLoader(validation_data, batch_size=self.context.get_per_slot_batch_size())

    def build_model(self) -> nn.Module:
        model = nn.Sequential(
            nn.Conv2d(1, self.context.get_hparam("n_filters1"), kernel_size=5),
            nn.MaxPool2d(2),
            nn.ReLU(),
            nn.Conv2d(
                self.context.get_hparam("n_filters1"),
                self.context.get_hparam("n_filters2"),
                kernel_size=5,
            ),
            nn.MaxPool2d(2),
            nn.ReLU(),
            Flatten(),
            nn.Linear(16 * self.context.get_hparam("n_filters2"), 50),
            nn.ReLU(),
            nn.Dropout2d(self.context.get_hparam("dropout")),
            nn.Linear(50, 10),
            nn.LogSoftmax(),
        )

        # If loading backbone weights, do not call reset_parameters() or
        # call before loading the backbone weights.
        reset_parameters(model)
        return model

    def optimizer(self, model: nn.Module) -> torch.optim.Optimizer:  # type: ignore
        return torch.optim.SGD(
            model.parameters(), lr=self.context.get_hparam("learning_rate"), momentum=0.9
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
        batch = cast(Tuple[torch.Tensor, torch.Tensor], batch)
        data, labels = batch

        output = model(data)
        error = error_rate(output, labels)

        return {"validation_error": error}
