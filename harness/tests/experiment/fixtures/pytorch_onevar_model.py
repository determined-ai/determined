"""
A one-variable linear model with no bias. The datset emits only pairs of (data, label) = (1, 1),
meaning that the one weight in the model should approach 1 as gradient descent continues.

We will use the mean squared error as the loss.  Since each record is the same, the "mean" part of
mean squared error means we can analyze every batch as if were just one record.

Now, we can calculate the mean squared error to ensure that we are getting the gradient we are
expecting.

let:
    l = loss
    w = the value of the one weight
    R = learning rate (constant)

then calculate the loss:

(1)     l = (label - (data * w)) ** 2

take derivative of loss WRT w

(2)     dl/dw = - 2 * data * (label - (data * w))

gradient update:

(3)     update = -R * dl/dw = 2 * R * data * (label - (data * w))

Finally, we can calculate the updated w' in terms of w:

(4)     w' = w + update = w + 2 * R * data * (label - (data * w))

TODO(DET-1597): migrate the all pytorch XOR trial unit tests to variations of the OneVarTrial.
"""

from typing import Any, Dict, Tuple

import numpy as np
import torch

from determined import pytorch


class OnesDataset(torch.utils.data.Dataset):
    def __len__(self) -> int:
        return 64

    def __getitem__(self, index: int) -> Tuple:
        return torch.Tensor([float(1)]), torch.Tensor([float(1)])


class OneVarTrial(pytorch.PyTorchTrial):
    def __init__(self, context: pytorch.PyTorchTrialContext) -> None:
        self.context = context

        model = torch.nn.Linear(1, 1, False)

        # Manually initialize the one weight to 0.
        model.weight.data.fill_(0)

        self.model = context.wrap_model(model)

        self.lr = 0.001

        opt = torch.optim.SGD(self.model.parameters(), self.lr)
        self.opt = context.wrap_optimizer(opt)

        self.loss_fn = torch.nn.MSELoss()

    def train_batch(
        self, batch: pytorch.TorchData, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        data, label = batch

        # Measure the weight right now.
        w_before = self.model.weight.data.item()

        # Calculate expected values for loss (eq 1) and weight (eq 4).
        loss_exp = (label[0] - data[0] * w_before) ** 2
        w_exp = w_before + 2 * self.lr * data[0] * (label[0] - (data[0] * w_before))

        loss = self.loss_fn(self.model(data), label)

        self.context.backward(loss)
        self.context.step_optimizer(self.opt)

        # Measure the weight after the update.
        w_after = self.model.weight.data.item()

        # Return values that we can compare as part of the tests.
        return {
            "loss": loss,
            "loss_exp": loss_exp,
            "w_before": w_before,
            "w_after": w_after,
            "w_exp": w_exp,
        }

    @staticmethod
    def check_batch_metrics(metrics: Dict[str, Any], batch_idx: int) -> None:
        """A check to be applied to the output of every train_batch in a test."""

        def float_eq(a: np.ndarray, b: np.ndarray) -> bool:
            epsilon = 0.000001
            return (np.abs(a - b) < epsilon).all()

        assert float_eq(
            metrics["loss"], metrics["loss_exp"]
        ), f'{metrics["loss"]} does not match {metrics["loss_exp"]} at batch {batch_idx}'

        assert float_eq(
            metrics["w_after"], metrics["w_exp"]
        ), f'{metrics["w_after"]} does not match {metrics["w_exp"]} at batch {batch_idx}'

    def evaluate_batch(self, batch: pytorch.TorchData) -> Dict[str, Any]:
        # Return the loss.
        data, label = batch
        loss = self.loss_fn(self.model(data), label)
        return {"val_loss": loss}

    def build_training_data_loader(self) -> pytorch.DataLoader:
        return pytorch.DataLoader(OnesDataset(), batch_size=self.context.get_per_slot_batch_size())

    def build_validation_data_loader(self) -> pytorch.DataLoader:
        return pytorch.DataLoader(OnesDataset(), batch_size=self.context.get_per_slot_batch_size())
