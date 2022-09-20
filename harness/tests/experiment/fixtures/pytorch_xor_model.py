# type: ignore
import logging
from typing import Any, Dict, List, cast

import numpy as np
import torch
from torch import nn
from torch.utils.data import TensorDataset

import determined as det
from determined import pytorch
from determined.common import check
from tests.experiment.fixtures.pytorch_counter_callback import Counter


def error_rate(predictions: torch.Tensor, labels: torch.Tensor) -> float:
    """Return the error rate based on dense predictions and dense labels."""
    check.equal_lengths(predictions, labels)
    check.len_eq(labels.shape, 1, "Labels must be a column vector")

    return (
        1.0 - float((predictions.argmax(1) == labels.to(torch.long)).sum()) / predictions.shape[0]
    )


def binary_error_rate(predictions: torch.Tensor, labels: torch.Tensor) -> float:
    """Return the classification error rate for binary classification."""
    check.eq(predictions.shape[0], labels.shape[0])
    check.is_in(len(predictions.shape), [1, 2])
    if len(predictions.shape) == 2:
        check.eq(predictions.shape[1], 1)
    check.len_eq(labels.shape, 1, "Labels must be a column vector")

    if len(predictions.shape) > 1:
        predictions = torch.squeeze(predictions)

    errors = torch.sum(labels.to(torch.long) != torch.round(predictions).to(torch.long))
    result = float(errors) / predictions.shape[0]  # type: float
    return result


def xor_data_loader(batch_size: int) -> pytorch.DataLoader:
    training_data = np.array([[0, 0], [0, 1], [1, 0], [1, 1]], dtype=np.float32)
    training_data = torch.Tensor(training_data)
    training_labels = np.array([0, 1, 1, 0], dtype=np.float32)
    training_labels = torch.Tensor(training_labels)
    training = TensorDataset(training_data, training_labels)
    return pytorch.DataLoader(training, batch_size=batch_size)


class XORNet(nn.Module):
    """
    XOR network with a single output (the loss). As is necessary for PyTorch
    models used in Determined, the forward method takes both the inputs and labels
    as arguments.
    """

    def __init__(self, context):
        super(XORNet, self).__init__()
        self.main_net = nn.Sequential(
            nn.Linear(2, context.get_hparam("hidden_size")),
            nn.Sigmoid(),
            nn.Linear(context.get_hparam("hidden_size"), 1),
            nn.Sigmoid(),
        )

    def forward(self, model_input: Any):
        return self.main_net(model_input)


class XORNetMulti(XORNet):
    """
    Multi-input multi-output XOR network.

    It uses the same data-label-prediction network as XORNet, but outputs in
    the MIMO format (a dictionary of predictions).
    """

    def forward(self, model_input: Any):
        return {"output": self.main_net(model_input)}


class ModifyableLRSchedule(torch.optim.lr_scheduler._LRScheduler):
    def __init__(self, *args, **kwargs):
        self.lr = float(0)
        super().__init__(*args, **kwargs)

    def get_lr(self) -> List[float]:
        return [self.lr for _ in self.base_lrs]

    def set_lr(self, lr: float) -> None:
        self.lr = lr


class BaseXORTrial(pytorch.PyTorchTrial):
    """
    Models a lightweight neural network model with one hidden layer to
    learn a binary XOR function. See Deep Learning Book, chapter 6.1 for
    the solution with a hidden size of 2, and an MSE loss function.

    This model has only one output node "loss".
    """

    def __init__(self, context: pytorch.PyTorchTrialContext) -> None:
        self.context = context
        self.model = self.context.wrap_model(XORNet(self.context))
        self.optimizer = self.context.wrap_optimizer(
            torch.optim.SGD(self.model.parameters(), self.context.get_hparam("learning_rate"))
        )

    def train_batch(
        self, batch: pytorch.TorchData, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        data, labels = batch
        output = self.model(data)
        loss = torch.nn.functional.binary_cross_entropy(output, labels.contiguous().view(-1, 1))

        self.context.backward(loss)
        self.context.step_optimizer(self.optimizer)
        return {"loss": loss}

    def build_training_data_loader(self) -> pytorch.DataLoader:
        return xor_data_loader(self.context.get_per_slot_batch_size())

    def build_validation_data_loader(self) -> pytorch.DataLoader:
        return xor_data_loader(self.context.get_per_slot_batch_size())


class XORTrial(BaseXORTrial):
    _searcher_metric = "loss"

    def evaluate_batch(self, batch: pytorch.TorchData) -> Dict[str, Any]:
        data, labels = batch
        output = self.model(data)
        loss = error_rate(output, labels)

        return {"loss": loss}


class XORTrialMulti(XORTrial):
    _searcher_metric = "binary_error"

    def __init__(self, context: pytorch.PyTorchTrialContext) -> None:
        self.context = context

        self.model = self.context.wrap_model(XORNetMulti(self.context))
        self.optimizer = self.context.wrap_optimizer(
            torch.optim.SGD(self.model.parameters(), self.context.get_hparam("learning_rate"))
        )

    def train_batch(
        self, batch: pytorch.TorchData, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        data, labels = batch
        output = self.model(data)
        loss = nn.functional.binary_cross_entropy(output["output"], labels.contiguous().view(-1, 1))

        self.context.backward(loss)
        self.context.step_optimizer(self.optimizer)
        return {"loss": loss}

    def evaluate_batch(self, batch: pytorch.TorchData) -> Dict[str, Any]:
        data, labels = batch
        output = self.model(data)
        error = binary_error_rate(output["output"], labels)

        return {"binary_error": error}


class XORTrialWithTrainingMetrics(XORTrialMulti):
    def train_batch(
        self, batch: pytorch.TorchData, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        data, labels = batch
        output = self.model(data)
        labels = cast(torch.Tensor, labels)
        loss = nn.functional.binary_cross_entropy(output["output"], labels.contiguous().view(-1, 1))
        accuracy = error_rate(output["output"], labels)

        self.context.backward(loss)
        self.context.step_optimizer(self.optimizer)
        return {"loss": loss, "accuracy": accuracy}


class XORTrialWithMultiValidation(XORTrialMulti):
    _searcher_metric = "accuracy"

    def evaluate_batch(self, batch: pytorch.TorchData) -> Dict[str, Any]:
        data, labels = batch
        output = self.model(data)
        accuracy = error_rate(output["output"], labels)
        binary_error = binary_error_rate(output["output"], labels)

        return {"accuracy": accuracy, "binary_error": binary_error}


class XORTrialPerMetricReducers(XORTrialWithMultiValidation):
    def evaluation_reducer(self) -> Dict[str, det.pytorch.Reducer]:
        return {"accuracy": det.pytorch.Reducer.AVG, "binary_error": det.pytorch.Reducer.AVG}


class EphemeralLegacyCallbackCounter(det.pytorch.PyTorchCallback):
    """
    Callback with legacy signature for on_training_epoch_start
    that takes no arguments. It is ephemeral: it does not implement
    state_dict and load_state_dict.
    """

    def __init__(self) -> None:
        self.legacy_on_training_epochs_start_calls = 0

    def on_training_epoch_start(self) -> None:
        logging.debug(f"calling {__name__} without arguments")
        self.legacy_on_training_epochs_start_calls += 1


class XORTrialCallbacks(XORTrialMulti):
    def __init__(self, context: pytorch.PyTorchTrialContext) -> None:
        super().__init__(context)

        self.context = context
        self.counter = Counter()
        self.legacy_counter = EphemeralLegacyCallbackCounter()

    def build_callbacks(self) -> Dict[str, det.pytorch.PyTorchCallback]:
        return {"counter": self.counter, "legacyCounter": self.legacy_counter}
