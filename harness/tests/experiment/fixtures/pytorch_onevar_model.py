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

from typing import Any, Dict, Tuple

import numpy as np
import torch

from determined import pytorch


class OnesDataset(torch.utils.data.Dataset):
    def __len__(self) -> int:
        return 64

    def __getitem__(self, index: int) -> Tuple:
        return {
            "data": torch.Tensor([float(1)]),
            "label": torch.Tensor([float(1)]),
            "idx": torch.Tensor([float(index)]),
        }


def batch_idx_sum():
    return sum(range(len(OnesDataset())))


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
        data = batch["data"]
        label = batch["label"]

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
        data = batch["data"]
        label = batch["label"]
        idx = batch["idx"]

        loss = self.loss_fn(self.model(data), label)

        # Return the batch index many different ways to test many different reducer types.
        return {
            "val_loss": loss,
            "batch_idx_tensor_fn": idx,
            "batch_idx_tensor_cls": idx,
            "batch_idx_list_fn": [idx, idx],
            "batch_idx_list_cls": [idx, idx],
            "batch_idx_dict_fn": {"a": idx, "b": idx},
            "batch_idx_dict_cls": {"a": idx, "b": idx},
        }

    def evaluation_reducer(self):
        return {
            "val_loss": pytorch.AvgMetricReducer,
            "batch_idx_tensor_cls": TensorSumReducer,
            "batch_idx_tensor_fn": tensor_sum_reducer,
            "batch_idx_list_cls": ListSumReducer,
            "batch_idx_list_fn": list_sum_reducer,
            "batch_idx_dict_cls": DictSumReducer,
            "batch_idx_dict_fn": dict_sum_reducer,
        }

    def build_training_data_loader(self) -> pytorch.DataLoader:
        return pytorch.DataLoader(OnesDataset(), batch_size=self.context.get_per_slot_batch_size())

    def build_validation_data_loader(self) -> pytorch.DataLoader:
        return pytorch.DataLoader(OnesDataset(), batch_size=self.context.get_per_slot_batch_size())


class TensorSumReducer(pytorch.MetricReducer):
    def __init__(self):
        self.sum = 0

    def accumulate(self, val):
        self.sum += val.sum()
        return self.sum

    def cross_slot_reduce(self, per_slot_metrics):
        return sum(per_slot_metrics)


def tensor_sum_reducer(metrics):
    return sum(m.sum() for m in metrics)


class ListSumReducer(pytorch.MetricReducer):
    def __init__(self):
        self.sum = 0

    def accumulate(self, vals):
        self.sum += sum(val.sum() for val in vals)
        return self.sum

    def cross_slot_reduce(self, per_slot_metrics):
        return sum(per_slot_metrics)


def list_sum_reducer(metric_lists):
    return sum(m.sum() for metric_list in metric_lists for m in metric_list)


class DictSumReducer(pytorch.MetricReducer):
    def __init__(self):
        self.sum = 0

    def accumulate(self, val_dict):
        self.sum += sum(val.sum() for val in val_dict.values())
        return self.sum

    def cross_slot_reduce(self, per_slot_metrics):
        return sum(per_slot_metrics)


def dict_sum_reducer(metric_dicts):
    return sum(m.sum() for metric_dict in metric_dicts for m in metric_dict.values())
