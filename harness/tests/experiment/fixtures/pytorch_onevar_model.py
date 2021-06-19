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
import yaml

from determined import experimental, pytorch


class OnesDataset(torch.utils.data.Dataset):
    def __len__(self) -> int:
        return 64

    def __getitem__(self, index: int) -> Tuple:
        return torch.Tensor([float(1)]), torch.Tensor([float(1)])


class TriangleLabelSum(pytorch.MetricReducer):
    """Return a sum of (label_sum * batch_index) for every batch (labels are always 1 here)."""

    @staticmethod
    def expect(batch_size, idx_start, idx_end):
        """What to expect during testing."""
        return sum(batch_size * idx for idx in range(idx_start, idx_end))

    def __init__(self) -> None:
        self.reset()

    def reset(self) -> None:
        self.sum = 0
        # We don't actually expose a batch_idx for evaluation, so we track the number of batches
        # since the last reset(), which is only accurate during evaluation workloads or the very
        # first training workload.
        self.count = 0

    def update(self, label_sum: torch.Tensor, batch_idx: Optional[int]) -> None:
        self.sum += label_sum * (batch_idx if batch_idx is not None else self.count)
        self.count += 1

    def per_slot_reduce(self) -> Any:
        return self.sum

    def cross_slot_reduce(self, per_slot_metrics) -> Any:
        return sum(per_slot_metrics)


def triangle_label_sum(updates: List) -> Any:
    out = 0
    for update_idx, (label_sum, batch_idx) in enumerate(updates):
        if batch_idx is not None:
            out += batch_idx * label_sum
        else:
            out += update_idx * label_sum
    return out


class OneVarTrial(pytorch.PyTorchTrial):
    _searcher_metric = "val_loss"

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

        self.cls_reducer = context.wrap_reducer(TriangleLabelSum(), name="cls_reducer")
        self.fn_reducer = context.wrap_reducer(triangle_label_sum, name="fn_reducer")

    def train_batch(
        self, batch: pytorch.TorchData, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        data, label = batch

        self.cls_reducer.update(sum(label), batch_idx)
        self.fn_reducer.update((sum(label), batch_idx))

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
            return (abs(a - b) < epsilon).all()

        assert float_eq(
            metrics["loss"], metrics["loss_exp"]
        ), f'{metrics["loss"]} does not match {metrics["loss_exp"]} at batch {batch_idx}'

        assert float_eq(
            metrics["w_after"], metrics["w_exp"]
        ), f'{metrics["w_after"]} does not match {metrics["w_exp"]} at batch {batch_idx}'

    def evaluate_batch(self, batch: pytorch.TorchData) -> Dict[str, Any]:
        data, label = batch

        self.cls_reducer.update(sum(label), None)
        self.fn_reducer.update((sum(label), None))

        loss = self.loss_fn(self.model(data), label)
        return {"val_loss": loss}

    def build_training_data_loader(self) -> pytorch.DataLoader:
        return pytorch.DataLoader(OnesDataset(), batch_size=self.context.get_per_slot_batch_size())

    def build_validation_data_loader(self) -> pytorch.DataLoader:
        return pytorch.DataLoader(OnesDataset(), batch_size=self.context.get_per_slot_batch_size())


if __name__ == "__main__":
    conf = yaml.safe_load(
        """
    description: test-native-api-local-test-mode
    hyperparameters:
      global_batch_size: 32
    scheduling_unit: 1
    searcher:
      name: single
      metric: val_loss
      max_length:
        batches: 1
      smaller_is_better: true
    max_restarts: 0
    """
    )
    experimental.create(OneVarTrial, conf, context_dir=".", local=True, test=True)
