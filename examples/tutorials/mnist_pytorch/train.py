"""
This example shows how to interact with the Determined PyTorch training APIs to
build a basic MNIST network.

In the `__init__` method, the model and optimizer are wrapped with `wrap_model`
and `wrap_optimizer`. This model is single-input and single-output.

The methods `train_batch` and `evaluate_batch` define the forward pass
for training and evaluation respectively.

Then, configure and run the training loop with PyTorch Trainer.
The model can be trained either locally or on-cluster.

"""
import logging
import pathlib
from typing import Any, Dict

import data
import model
import torch
from ruamel import yaml
from torch import nn

import determined as det
from determined import pytorch


class MNistTrial(pytorch.PyTorchTrial):
    def __init__(self, context: pytorch.PyTorchTrialContext, hparams: Dict) -> None:
        self.context = context

        # Trial-level constants.
        self.data_dir = "data"
        self.data_url = (
            "https://s3-us-west-2.amazonaws.com/determined-ai-test-data/pytorch_mnist.tar.gz"
        )

        self.batch_size = 64
        self.per_slot_batch_size = self.batch_size // self.context.distributed.get_size()

        # Define loss function.
        self.loss_fn = nn.NLLLoss()

        # Define model.
        self.model = self.context.wrap_model(model.build_model(hparams=hparams))

        # Configure optimizer.
        self.optimizer = self.context.wrap_optimizer(
            torch.optim.Adadelta(self.model.parameters(), lr=hparams["learning_rate"])
        )

    def build_training_data_loader(self) -> pytorch.DataLoader:
        train_data = data.get_dataset(self.data_dir, train=True)
        return pytorch.DataLoader(train_data, batch_size=self.per_slot_batch_size)

    def build_validation_data_loader(self) -> pytorch.DataLoader:
        validation_data = data.get_dataset(self.data_dir, train=False)
        return pytorch.DataLoader(validation_data, batch_size=self.per_slot_batch_size)

    def train_batch(
        self, batch: pytorch.TorchData, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        batch_data, labels = batch

        output = self.model(batch_data)
        loss = self.loss_fn(output, labels)

        self.context.backward(loss)
        self.context.step_optimizer(self.optimizer)

        return {"loss": loss}

    def evaluate_batch(self, batch: pytorch.TorchData, batch_idx: int) -> Dict[str, Any]:
        batch_data, labels = batch

        output = self.model(batch_data)
        validation_loss = self.loss_fn(output, labels).item()

        pred = output.argmax(dim=1, keepdim=True)
        accuracy = pred.eq(labels.view_as(pred)).sum().item() / len(batch_data)

        return {"validation_loss": validation_loss, "accuracy": accuracy}


def run(local: bool = False):
    """Initializes the trial and runs the training loop.

    This method configures the appropriate training parameters for both local and on-cluster training modes. It
    is an example of a standalone training script that can run both locally and on-cluster without any code
    changes. To run the training code solely locally or on-cluster, simply remove the conditional parameter logic for
    the unnecessary mode.

    Arguments:
        local: Whether or not to run this script locally. Defaults to false (on-cluster training).
    """

    info = det.get_cluster_info()

    if local:
        # For convenience, use hparams from const.yaml for local mode.
        conf = yaml.safe_load(pathlib.Path("./const.yaml").read_text())
        hparams = conf["hyperparameters"]
        max_length = pytorch.Batch(100)  # Train for 100 batches.
        latest_checkpoint = None
    else:
        hparams = info.trial.hparams  # Get instance of hparam values from Determined cluster info.
        max_length = None  # On-cluster training trains for the searcher's configured length.
        latest_checkpoint = (
            info.latest_checkpoint
        )  # (Optional) Configure checkpoint for pause/resume functionality.

    with pytorch.init() as train_context:
        trial = MNistTrial(train_context, hparams=hparams)
        trainer = pytorch.Trainer(trial, train_context)
        trainer.fit(max_length=max_length, latest_checkpoint=latest_checkpoint)


if __name__ == "__main__":
    # Configure logging
    logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)
    local_training = det.get_cluster_info() is None
    run(local=local_training)
