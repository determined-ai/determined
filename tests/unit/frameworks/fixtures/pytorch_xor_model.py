from typing import Any, Dict, cast

import numpy as np
import torch
from torch import nn
from torch.utils.data import TensorDataset

import determined as det
from determined.pytorch import DataLoader, LRScheduler, PyTorchTrial, TorchData, reset_parameters
from determined_common import check


def error_rate(predictions: torch.Tensor, labels: torch.Tensor) -> float:
    """Return the error rate based on dense predictions and dense labels."""
    check.equal_lengths(predictions, labels)
    check.len_eq(labels.shape, 1, "Labels must be a column vector")

    return (  # type: ignore
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
        reset_parameters(self.main_net)

    def forward(self, model_input: Any):
        return self.main_net(model_input)


def xor_data_loader(batch_size: int) -> DataLoader:
    training_data = np.array([[0, 0], [0, 1], [1, 0], [1, 1]], dtype=np.float32)
    training_data = torch.Tensor(training_data)
    training_labels = np.array([0, 1, 1, 0], dtype=np.float32)
    training_labels = torch.Tensor(training_labels)
    training = TensorDataset(training_data, training_labels)
    return DataLoader(training, batch_size=batch_size)


class BaseXORTrial(PyTorchTrial):
    """
    Models a lightweight neural network model with one hidden layer to
    learn a binary XOR function. See Deep Learning Book, chapter 6.1 for
    the solution with a hidden size of 2, and a MSE loss function.

    This model has only one output node "loss".
    """

    def __init__(self, context: Any) -> None:
        self.context = context

    def build_model(self) -> nn.Module:
        return XORNet(self.context)

    def optimizer(self, model: nn.Module) -> torch.optim.Optimizer:
        return torch.optim.SGD(model.parameters(), self.context.get_hparam("learning_rate"))

    def train_batch(
        self, batch: TorchData, model: nn.Module, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        data, labels = batch
        output = model(data)
        loss = torch.nn.functional.binary_cross_entropy(output, labels.view(-1, 1))

        return {"loss": loss}

    def build_training_data_loader(self) -> DataLoader:
        return xor_data_loader(self.context.get_per_slot_batch_size())

    def build_validation_data_loader(self) -> DataLoader:
        return xor_data_loader(self.context.get_per_slot_batch_size())


class XORTrial(BaseXORTrial):
    def evaluate_batch(self, batch: TorchData, model: nn.Module) -> Dict[str, Any]:
        data, labels = batch
        output = model(data)
        loss = error_rate(output, labels)

        return {"loss": loss}


class XORTrialOptimizerState(XORTrial):
    def optimizer(self, model: nn.Module) -> torch.optim.Optimizer:
        return torch.optim.SGD(
            model.parameters(), self.context.get_hparam("learning_rate"), momentum=0.9
        )


class XORNetMulti(XORNet):
    """
    Multi-input multi-output XOR network.

    It uses the same data-label-prediction network as XORNet, but outputs in
    the MIMO format (a dictionary of predictions).
    """

    def forward(self, model_input: Any):
        return {"output": self.main_net(model_input)}


class XORTrialMulti(XORTrial):
    # Same as XORTrial but with multi-output net XORNetMulti.
    def build_model(self) -> nn.Module:
        return XORNetMulti(self.context)

    def train_batch(
        self, batch: TorchData, model: nn.Module, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        data, labels = batch
        output = model(data)
        loss = nn.functional.binary_cross_entropy(output["output"], labels.view(-1, 1))

        return {"loss": loss}

    def evaluate_batch(self, batch: TorchData, model: nn.Module) -> Dict[str, Any]:
        data, labels = batch
        output = model(data)
        error = binary_error_rate(output["output"], labels)

        return {"binary_error": error}


class XORTrialWithTrainingMetrics(XORTrialMulti):
    def train_batch(
        self, batch: TorchData, model: nn.Module, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        data, labels = batch
        output = model(data)
        labels = cast(torch.Tensor, labels)
        loss = nn.functional.binary_cross_entropy(output["output"], labels.view(-1, 1))
        accuracy = error_rate(output["output"], labels)

        return {"loss": loss, "accuracy": accuracy}


class XORTrialWithMultiValidation(XORTrialMulti):
    def evaluate_batch(self, batch: TorchData, model: nn.Module) -> Dict[str, Any]:
        data, labels = batch
        output = model(data)
        accuracy = error_rate(output["output"], labels)
        binary_error = binary_error_rate(output["output"], labels)

        return {"accuracy": accuracy, "binary_error": binary_error}


class XORTrialWithNonScalarValidation(PyTorchTrial):
    def __init__(self, context: det.TrialContext) -> None:
        self.context = context

    def build_model(self) -> nn.Module:
        return XORNetMulti(self.context)

    def optimizer(self, model: nn.Module) -> torch.optim.Optimizer:
        return torch.optim.SGD(model.parameters(), self.context.get_hparam("learning_rate"))

    def build_training_data_loader(self) -> DataLoader:
        return xor_data_loader(self.context.get_per_slot_batch_size())

    def build_validation_data_loader(self) -> DataLoader:
        return xor_data_loader(self.context.get_per_slot_batch_size())

    def train_batch(
        self, batch: TorchData, model: nn.Module, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        data, labels = batch
        output = model(data)
        loss = nn.functional.binary_cross_entropy(output["output"], labels.view(-1, 1))

        return {"loss": loss}

    def evaluate_full_dataset(
        self, data_loader: torch.utils.data.DataLoader, model: nn.Module
    ) -> Dict[str, Any]:
        predictions = []
        binary_error_sum = 0.0
        for data, labels in iter(data_loader):
            if torch.cuda.is_available():
                data, labels = data.cuda(), labels.cuda()
            output = model(data)
            predictions.append(output)
            binary_error_sum += binary_error_rate(output["output"], labels)

        binary_error = binary_error_sum / len(data_loader)
        return {"predictions": predictions, "binary_error": binary_error}


class XORTrialCustomEval(BaseXORTrial):
    def evaluate_full_dataset(
        self, data_loader: torch.utils.data.DataLoader, model: nn.Module
    ) -> Dict[str, Any]:
        loss_sum = 0.0
        for data, labels in iter(data_loader):
            if torch.cuda.is_available():
                data, labels = data.cuda(), labels.cuda()
            output = model(data)
            loss_sum += error_rate(output, labels)

        loss = loss_sum / len(data_loader)
        return {"loss": loss}


class ModifyableLRSchedule(torch.optim.lr_scheduler._LRScheduler):
    def __init__(self, *args, **kwargs):
        self.lr = float(0)
        super().__init__(*args, **kwargs)

    def get_lr(self) -> float:
        return [self.lr for _ in self.base_lrs]

    def set_lr(self, lr: float) -> None:
        self.lr = lr


class XORTrialStepEveryEpoch(XORTrialMulti):
    def create_lr_scheduler(self, optimizer):
        self.scheduler = ModifyableLRSchedule(optimizer)
        return LRScheduler(self.scheduler, step_mode=LRScheduler.StepMode.STEP_EVERY_EPOCH)


class XORTrialRestoreLR(XORTrialMulti):
    def create_lr_scheduler(self, optimizer):
        self.scheduler = ModifyableLRSchedule(optimizer)
        return LRScheduler(self.scheduler, step_mode=LRScheduler.StepMode.STEP_EVERY_BATCH)

    def train_batch(
        self, batch: TorchData, model: nn.Module, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        metrics = super().train_batch(batch, model, epoch_idx, batch_idx)
        lr = self.scheduler.get_lr()[0]
        metrics["lr"] = lr
        self.scheduler.set_lr(lr + 1)
        return metrics


class XORTrialUserStepLRFail(XORTrialMulti):
    def create_lr_scheduler(self, optimizer):
        self.scheduler = ModifyableLRSchedule(optimizer)
        return LRScheduler(self.scheduler, step_mode=LRScheduler.StepMode.STEP_EVERY_BATCH)

    def train_batch(
        self, batch: TorchData, model: nn.Module, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        metrics = super().train_batch(batch, model, epoch_idx, batch_idx)
        self.scheduler.step()
        return metrics


class XORTrialUserStepLR(XORTrialMulti):
    def create_lr_scheduler(self, optimizer):
        self.scheduler = ModifyableLRSchedule(optimizer)
        return LRScheduler(self.scheduler, step_mode=LRScheduler.StepMode.MANUAL_STEP)

    def train_batch(
        self, batch: TorchData, model: nn.Module, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        metrics = super().train_batch(batch, model, epoch_idx, batch_idx)
        self.scheduler.step()
        return metrics


class XORTrialPerMetricReducers(XORTrialWithMultiValidation):
    def evaluation_reducer(self) -> Dict[str, det.pytorch.Reducer]:
        return {"accuracy": det.pytorch.Reducer.AVG, "binary_error": det.pytorch.Reducer.AVG}
