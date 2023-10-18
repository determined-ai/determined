"""
This example shows how to interact with the Determined PyTorch interface to
build a basic MNIST network.

In the `__init__` method, the model and optimizer are wrapped with `wrap_model`
and `wrap_optimizer`. This model is single-input and single-output.

The methods `train_batch` and `evaluate_batch` define the forward pass
for training and evaluation respectively.
"""
import logging
from typing import Any, Dict, Sequence, Tuple, Union, cast

import data
import torch
import layers
from torch import nn

import determined as det
from determined import pytorch

TorchData = Union[Dict[str, torch.Tensor], Sequence[torch.Tensor], torch.Tensor]


class MNistTrial(pytorch.PyTorchTrial):
    def __init__(self, context: pytorch.PyTorchTrialContext, hparams: Dict) -> None:
        self.context = context

        # Create a unique download directory for each rank so they don't overwrite each
        # other when doing distributed training.
        self.download_directory = f"/tmp/data-rank{self.context.distributed.get_rank()}"
        self.data_downloaded = False

        self.model = self.context.wrap_model(
            nn.Sequential(
                nn.Conv2d(1, self.context.get_hparam("n_filters1"), 3, 1),
                nn.ReLU(),
                nn.Conv2d(
                    self.context.get_hparam("n_filters1"),
                    self.context.get_hparam("n_filters2"),
                    3,
                ),
                nn.ReLU(),
                nn.MaxPool2d(2),
                nn.Dropout2d(self.context.get_hparam("dropout1")),
                layers.Flatten(),
                nn.Linear(144 * self.context.get_hparam("n_filters2"), 128),
                nn.ReLU(),
                nn.Dropout2d(self.context.get_hparam("dropout2")),
                nn.Linear(128, 10),
                nn.LogSoftmax(),
            )
        )

        self.optimizer = self.context.wrap_optimizer(
            torch.optim.Adadelta(
                self.model.parameters(), lr=self.context.get_hparam("learning_rate")
            )
        )

    def build_training_data_loader(self) -> pytorch.DataLoader:
        if not self.data_downloaded:
            self.download_directory = data.download_dataset(
                download_directory=self.download_directory,
                data_config=self.context.get_data_config(),
            )
            self.data_downloaded = True

        train_data = data.get_dataset(self.download_directory, train=True)
        return pytorch.DataLoader(train_data, batch_size=self.context.get_per_slot_batch_size())

    def build_validation_data_loader(self) -> pytorch.DataLoader:
        if not self.data_downloaded:
            self.download_directory = data.download_dataset(
                download_directory=self.download_directory,
                data_config=self.context.get_data_config(),
            )
            self.data_downloaded = True

        validation_data = data.get_dataset(self.download_directory, train=False)
        return pytorch.DataLoader(validation_data, batch_size=self.context.get_per_slot_batch_size())

    def train_batch(
        self, batch: TorchData, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        batch = cast(Tuple[torch.Tensor, torch.Tensor], batch)
        data, labels = batch

        output = self.model(data)
        loss = torch.nn.functional.nll_loss(output, labels)

        self.context.backward(loss)
        self.context.step_optimizer(self.optimizer)

        return {"loss": loss}

    def evaluate_batch(self, batch: TorchData) -> Dict[str, Any]:
        batch = cast(Tuple[torch.Tensor, torch.Tensor], batch)
        data, labels = batch

        output = self.model(data)
        validation_loss = torch.nn.functional.nll_loss(output, labels).item()

        pred = output.argmax(dim=1, keepdim=True)
        accuracy = pred.eq(labels.view_as(pred)).sum().item() / len(data)

        return {"validation_loss": validation_loss, "accuracy": accuracy}


def run(local: bool = False):
    info = det.get_cluster_info()

    max_length = None
    latest_checkpoint = None
    hparams = None

    if local:
        max_length = pytorch.Batch(100)
    else:
        latest_checkpoint = info.latest_checkpoint
        hparams = info.trial.hparams

    with pytorch.init() as train_context:
        trial = MNistTrial(train_context, hparams=hparams)
        trainer = pytorch.Trainer(trial, train_context)
        trainer.fit(
            max_length=max_length,
            latest_checkpoint=latest_checkpoint
        )


if __name__ == "__main__":
    # Configure logging
    logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)
    local = det.get_cluster_info() is None
    run(local=local)

