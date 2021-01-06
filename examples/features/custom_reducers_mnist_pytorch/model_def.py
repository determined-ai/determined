"""
This example shows how to use custom reducers with PyTorch.  In this example,
the custom reducer is a per-class F1 score.  It is based on Determined's
mnist_pytorch tutorial code.

The parts of the file which differ from the tutorial are the PerClassF1Score class
and the calls to wrap_reducer() and self.f1_score.update.
"""

from typing import Any, Dict, Sequence, Tuple, Union, cast

import torch
from torch import nn

from layers import Flatten

from determined.pytorch import DataLoader, PyTorchTrial, PyTorchTrialContext, MetricReducer

import data

TorchData = Union[Dict[str, torch.Tensor], Sequence[torch.Tensor], torch.Tensor]


class PerClassF1Score(MetricReducer):
    def __init__(self):
        # This is MNIST, so 10 classes.
        self.num_classes = 10
        self.reset()

    def reset(self):
        # reset() will be called before each training and validation workload.
        self.true_positives = [0.] * self.num_classes
        self.false_positives = [0.] * self.num_classes
        self.false_negatives = [0.] * self.num_classes

    def update(self, y_true, y_pred):
        # We are responsible for calling update() as part of our train_batch() and evaluate_batch()
        # methods, which means we can specify any arguments we wish.  For our per-class F1 score,
        # it is useful to know y_true and y_pred.
        for cls in range(self.num_classes):
            self.true_positives[cls] += sum((y_pred == cls) * (y_true == cls))
            self.false_positives[cls] += sum((y_pred == cls) * (y_true != cls))
            self.false_negatives[cls] += sum((y_pred != cls) * (y_true == cls))

    def per_slot_reduce(self):
        # per_slot_reduce() is an opportunity to do any calculations on for a single worker before
        # passing them across the network for the final cross_slot_reduce operation.  In the case
        # of our F1 score, the update() function very efficient and there is nothing useful to do
        # here, so we just pass the per-class TP/FP/FN across the network for the cross_slot_reduce.
        return self.true_positives, self.false_positives, self.false_negatives

    def cross_slot_reduce(self, per_slot_metrics):
        # cross_slot_reduce() is the final step of the reduction.  The per_slot_metrics contains
        f1_scores = {}
        # Reshape inputs.
        per_slot_tp, per_slot_fp, per_slot_fn = zip(*per_slot_metrics)
        for cls in range(self.num_classes):
            # Sum across slots.
            tp = sum(slot_tp[cls] for slot_tp in per_slot_tp)
            fp = sum(slot_fp[cls] for slot_fp in per_slot_fp)
            fn = sum(slot_fn[cls] for slot_fn in per_slot_fn)
            # F1 score formula from Wikipedia.
            score = tp / (tp + .5 * (fp + fn))
            f1_scores[f"f1_score_{cls}"] = score
        return f1_scores


class MNistTrial(PyTorchTrial):
    def __init__(self, context: PyTorchTrialContext) -> None:
        self.context = context

        self.download_directory = f"/tmp/data-rank{self.context.distributed.get_rank()}"
        self.data_downloaded = False

        self.model = self.context.wrap_model(nn.Sequential(
            nn.Conv2d(1, self.context.get_hparam("n_filters1"), 3, 1),
            nn.ReLU(),
            nn.Conv2d(
                self.context.get_hparam("n_filters1"), self.context.get_hparam("n_filters2"), 3,
            ),
            nn.ReLU(),
            nn.MaxPool2d(2),
            nn.Dropout2d(self.context.get_hparam("dropout1")),
            Flatten(),
            nn.Linear(144 * self.context.get_hparam("n_filters2"), 128),
            nn.ReLU(),
            nn.Dropout2d(self.context.get_hparam("dropout2")),
            nn.Linear(128, 10),
            nn.LogSoftmax(),
        ))

        self.optimizer = self.context.wrap_optimizer(torch.optim.Adadelta(
            self.model.parameters(), lr=self.context.get_hparam("learning_rate"))
        )

        # We let name=None (by not specifiying name) to return a dictionary of metrics from a
        # single reducer (one f1_score per class).  If we were going to return a single metric
        # (rather than a dictionary of multiple metrics) we would have to specify the name here.
        self.f1_score = self.context.experimental.wrap_reducer(PerClassF1Score())

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

    def train_batch(
        self, batch: TorchData, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        batch = cast(Tuple[torch.Tensor, torch.Tensor], batch)
        data, labels = batch

        output = self.model(data)
        loss = torch.nn.functional.nll_loss(output, labels)

        # Update our custom metric.
        self.f1_score.update(y_true=labels, y_pred=output.argmax(dim=1))

        self.context.backward(loss)
        self.context.step_optimizer(self.optimizer)

        return {"loss": loss}

    def evaluate_batch(self, batch: TorchData) -> Dict[str, Any]:
        batch = cast(Tuple[torch.Tensor, torch.Tensor], batch)
        data, labels = batch

        output = self.model(data)
        validation_loss = torch.nn.functional.nll_loss(output, labels).item()

        # Update our custom metric.
        self.f1_score.update(y_true=labels, y_pred=output.argmax(dim=1))

        pred = output.argmax(dim=1, keepdim=True)
        accuracy = pred.eq(labels.view_as(pred)).sum().item() / len(data)

        return {"validation_loss": validation_loss, "accuracy": accuracy}
