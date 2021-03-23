"""
A one-variable linear model with no bias. The datset emits only pairs of (data, label) = (1, 1),
meaning that the one weight in the model should approach 1 as gradient descent continues.

We will use the mean squared error as the loss.  Since each record is the same, the "mean" part of
mean squared error means we can analyze every batch as if were just one record.

Now, we can calculate the mean squared error to ensure that we are getting the gradient we are
expecting.

let:
    R = learning rate (constant)
    l = loss
    w0 = the starting value of the one weight
    w' = the updated value of the one weight

then calculate the loss:

(1)     l = (label - (data * w0)) ** 2

take derivative of loss WRT w

(2)     dl/dw = - 2 * data * (label - (data * w0))

gradient update:

(3)     update = -R * dl/dw = 2 * R * data * (label - (data * w0))

Finally, we can calculate the updated weight (w') in terms of w0:

(4)     w' = w0 + update = w0 + 2 * R * data * (label - (data * w0))

TODO(DET-1597): migrate the all pytorch XOR trial unit tests to variations of the OneVarTrial.
"""

from typing import Any, Dict, List, Optional, Tuple

import numpy as np
import torch
import pytorch_lightning as pl

from determined import pytorch
from determined.pytorch.lightning import LightningAdapter


class OnesDataset(torch.utils.data.Dataset):
    def __len__(self) -> int:
        return 64

    def __getitem__(self, index: int) -> Tuple:
        return torch.Tensor([float(1)]), torch.Tensor([float(1)])

class OneVarLM(pl.LightningModule):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)

        model = torch.nn.Linear(1, 1, False)
        self.model = model

        # Manually initialize the one weight to 0.
        model.weight.data.fill_(0)

        self.lr = 0.001

        self.loss_fn = torch.nn.MSELoss()

    def configure_optimizers(self):
        opt = torch.optim.SGD(self.model.parameters(), self.lr)
        return opt

    def training_step(self, batch, batch_idx, *args, **kwargs):
        data, label = batch

        # Measure the weight right now.
        w_before = self.model.weight.data.item()

        # Calculate expected values for loss (eq 1) and weight (eq 4).
        loss_exp = (label[0] - data[0] * w_before) ** 2
        w_exp = w_before + 2 * self.lr * data[0] * (label[0] - (data[0] * w_before))

        loss = self.loss_fn(self.model(data), label)

        # Return values that we can compare as part of the tests.
        return {
            "loss": loss,
            "loss_exp": loss_exp,
            "w_before": w_before,
            "w_exp": w_exp,
        }

    def validation_step(self, batch, *args, **kwargs):
        data, label = batch

        loss = self.loss_fn(self.model(data), label)
        return {"val_loss": loss}

class OneDatasetLDM(pl.LightningDataModule):
    def __init__(self, batch_size: int = 32, *args, **kwargs):
        self.batch_size = batch_size
        super().__init__(*args, **kwargs)

    def train_dataloader(self) -> torch.utils.data.DataLoader:
        return torch.utils.data.DataLoader(OnesDataset(), batch_size=self.batch_size)

    def val_dataloader(self) -> torch.utils.data.DataLoader:
        return torch.utils.data.DataLoader(OnesDataset(), batch_size=self.batch_size)


class OneVarTrial(LightningAdapter):
    def __init__(self, context: pytorch.PyTorchTrialContext) -> None:
        self.context = context
        lm = OneVarLM()
        self.dm = OneDatasetLDM()
        super().__init__(context, lm)

    @staticmethod
    def check_batch_metrics(metrics: Dict[str, Any], batch_idx: int) -> None:
        """A check to be applied to the output of every train_batch in a test."""

        def float_eq(a: np.ndarray, b: np.ndarray) -> bool:
            epsilon = 0.000001
            return (abs(a - b) < epsilon).all()

        assert float_eq(
            metrics["loss"], metrics["loss_exp"]
        ), f'{metrics["loss"]} does not match {metrics["loss_exp"]} at batch {batch_idx}'

        assert float_eq(
            metrics["w_after"], metrics["w_exp"]
        ), f'{metrics["w_after"]} does not match {metrics["w_exp"]} at batch {batch_idx}'

    def build_training_data_loader(self) -> pytorch.DataLoader:
        return pytorch.DataLoader(self.dm.train_dataloader().dataset,
                                  batch_size=self.context.get_per_slot_batch_size())

    def build_validation_data_loader(self) -> pytorch.DataLoader:
        return pytorch.DataLoader(self.dm.val_dataloader().dataset,
                                  batch_size=self.context.get_per_slot_batch_size())

if __name__ == '__main__':
    model = OneVarLM()
    trainer = pl.Trainer(max_epochs=2, default_root_dir='/tmp/lightning')

    dm = OneDatasetLDM()
    trainer.fit(model, datamodule=dm)
