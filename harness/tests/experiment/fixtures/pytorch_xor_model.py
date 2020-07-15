from typing import Any, Dict, cast

import numpy as np
import torch
from torch import nn
from torch.utils.data import TensorDataset

import determined as det
from determined import pytorch
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
        pytorch.reset_parameters(self.main_net)

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


class StepableLRSchedule(torch.optim.lr_scheduler._LRScheduler):
    def get_lr(self) -> float:
        return [self._step_count for _ in self.base_lrs]


class ModifyableLRSchedule(torch.optim.lr_scheduler._LRScheduler):
    def __init__(self, *args, **kwargs):
        self.lr = float(0)
        super().__init__(*args, **kwargs)

    def get_lr(self) -> float:
        return [self.lr for _ in self.base_lrs]

    def set_lr(self, lr: float) -> None:
        self.lr = lr


class BaseXORTrial(pytorch.PyTorchTrial):
    """
    Models a lightweight neural network model with one hidden layer to
    learn a binary XOR function. See Deep Learning Book, chapter 6.1 for
    the solution with a hidden size of 2, and a MSE loss function.

    This model has only one output node "loss".
    """

    def __init__(self, context: pytorch.PyTorchTrialContext) -> None:
        self.context = context
        self.model = self.context.Model(XORNet(self.context))
        self.optimizer = self.context.Optimizer(
            torch.optim.SGD(self.model.parameters(), self.context.get_hparam("learning_rate"))
        )

    def train_batch(
        self, batch: pytorch.TorchData, model: nn.Module, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        data, labels = batch
        output = self.model(data)
        loss = torch.nn.functional.binary_cross_entropy(output, labels.view(-1, 1))

        self.context.backward(loss)
        self.context.step_optimizer(self.optimizer)
        return {"loss": loss}

    def build_training_data_loader(self) -> pytorch.DataLoader:
        return xor_data_loader(self.context.get_per_slot_batch_size())

    def build_validation_data_loader(self) -> pytorch.DataLoader:
        return xor_data_loader(self.context.get_per_slot_batch_size())


class XORTrial(BaseXORTrial):
    def evaluate_batch(self, batch: pytorch.TorchData, model: nn.Module) -> Dict[str, Any]:
        data, labels = batch
        output = self.model(data)
        loss = error_rate(output, labels)

        return {"loss": loss}


class XORTrialMulti(XORTrial):
    def __init__(self, context: pytorch.PyTorchTrialContext) -> None:
        self.context = context

        self.model = self.context.Model(XORNetMulti(self.context))
        self.optimizer = self.context.Optimizer(
            torch.optim.SGD(self.model.parameters(), self.context.get_hparam("learning_rate"))
        )

    def train_batch(
        self, batch: pytorch.TorchData, model: nn.Module, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        data, labels = batch
        output = self.model(data)
        loss = nn.functional.binary_cross_entropy(output["output"], labels.view(-1, 1))

        self.context.backward(loss)
        self.context.step_optimizer(self.optimizer)
        return {"loss": loss}

    def evaluate_batch(self, batch: pytorch.TorchData, model: nn.Module) -> Dict[str, Any]:
        data, labels = batch
        output = self.model(data)
        error = binary_error_rate(output["output"], labels)

        return {"binary_error": error}


class XORTrialWithTrainingMetrics(XORTrialMulti):
    def train_batch(
        self, batch: pytorch.TorchData, model: nn.Module, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        data, labels = batch
        output = model(data)
        labels = cast(torch.Tensor, labels)
        loss = nn.functional.binary_cross_entropy(output["output"], labels.view(-1, 1))
        accuracy = error_rate(output["output"], labels)

        self.context.backward(loss)
        self.context.step_optimizer(self.optimizer)
        return {"loss": loss, "accuracy": accuracy}


class XORTrialWithMultiValidation(XORTrialMulti):
    def evaluate_batch(self, batch: pytorch.TorchData, model: nn.Module) -> Dict[str, Any]:
        data, labels = batch
        output = self.model(data)
        accuracy = error_rate(output["output"], labels)
        binary_error = binary_error_rate(output["output"], labels)

        return {"accuracy": accuracy, "binary_error": binary_error}


class XORTrialWithNonScalarValidation(pytorch.PyTorchTrial):
    def __init__(self, context: pytorch.PyTorchTrialContext) -> None:
        self.context = context

        self.model = self.context.Model(XORNetMulti(self.context))
        self.optimizer = self.context.Optimizer(
            torch.optim.SGD(self.model.parameters(), self.context.get_hparam("learning_rate"))
        )

    def build_training_data_loader(self) -> pytorch.DataLoader:
        return xor_data_loader(self.context.get_per_slot_batch_size())

    def build_validation_data_loader(self) -> pytorch.DataLoader:
        return xor_data_loader(self.context.get_per_slot_batch_size())

    def train_batch(
        self, batch: pytorch.TorchData, model: nn.Module, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        data, labels = batch
        output = model(data)
        loss = nn.functional.binary_cross_entropy(output["output"], labels.view(-1, 1))

        self.context.backward(loss)
        self.context.step_optimizer(self.optimizer)
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


class XORTrialWithLRScheduler(XORTrialMulti):
    def __init__(self, context: pytorch.PyTorchTrialContext) -> None:
        self.context = context

        # Same as XORTrial but with multi-output net XORNetMulti.
        self.model = self.context.Model(XORNetMulti(self.context))
        self.optimizer = self.context.Optimizer(
            torch.optim.SGD(self.model.parameters(), self.context.get_hparam("learning_rate"))
        )

        self.lr_scheduler = self.context.LRScheduler(
            StepableLRSchedule(self.optimizer),
            step_mode=pytorch.LRScheduler.StepMode(
                self.context.get_hparam("lr_scheduler_step_mode")
            ),
        )

    def train_batch(
        self, batch: pytorch.TorchData, model: nn.Module, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        metrics = super().train_batch(batch, model, epoch_idx, batch_idx)
        lr = self.lr_scheduler.get_last_lr()[0]
        metrics["lr"] = lr

        if (
            self.context.get_hparam("lr_scheduler_step_mode")
            == pytorch.LRScheduler.StepMode.MANUAL_STEP
        ):
            self.lr_scheduler.step()
        return metrics


class XORTrialPerMetricReducers(XORTrialWithMultiValidation):
    def evaluation_reducer(self) -> Dict[str, det.pytorch.Reducer]:
        return {"accuracy": det.pytorch.Reducer.AVG, "binary_error": det.pytorch.Reducer.AVG}


class Counter(det.pytorch.PyTorchCallback):
    def __init__(self) -> None:
        self.validation_steps_started = 0
        self.validation_steps_ended = 0
        self.checkpoints_ended = 0

    def on_validation_start(self) -> None:
        self.validation_steps_started += 1

    def on_validation_end(self, metrics: Dict[str, Any]) -> None:
        self.validation_steps_ended += 1

    def on_checkpoint_end(self, checkpoint_dir: str):
        self.checkpoints_ended += 1

    def state_dict(self) -> Dict[str, Any]:
        return self.__dict__

    def load_state_dict(self, state_dict: Dict[str, Any]) -> None:
        self.__dict__ = state_dict


class XORTrialCallbacks(XORTrialMulti):
    def __init__(self, context: pytorch.PyTorchTrialContext) -> None:
        super().__init__(context)

        self.context = context
        self.counter = Counter()

    def build_callbacks(self) -> Dict[str, det.pytorch.PyTorchCallback]:
        return {"counter": self.counter}


class XORTrialAccessContext(BaseXORTrial):
    def __init__(self, context: pytorch.PyTorchTrialContext) -> None:
        self.context = context

        self.model_a = self.context.Model(XORNet(self.context))
        self.model_b = self.context.Model(XORNet(self.context))
        self.opt_a = self.context.Optimizer(
            torch.optim.SGD(self.model_a.parameters(), self.context.get_hparam("learning_rate"))
        )
        self.opt_b = self.context.Optimizer(
            torch.optim.SGD(self.model_b.parameters(), self.context.get_hparam("learning_rate"))
        )
        self.lrs_a = self.context.LRScheduler(
            StepableLRSchedule(self.opt_a),
            step_mode=pytorch.LRScheduler.StepMode(
                self.context.get_hparam("lr_scheduler_step_mode")
            ),
        )
        self.lrs_b = self.context.LRScheduler(
            StepableLRSchedule(self.opt_b),
            step_mode=pytorch.LRScheduler.StepMode(
                self.context.get_hparam("lr_scheduler_step_mode")
            ),
        )

    def train_batch(
        self, batch: pytorch.TorchData, model: nn.Module, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        assert self.context.models
        assert self.context.optimizers
        assert self.context.lr_schedulers

        data, labels = batch
        output = self.model_a(data)
        loss = torch.nn.functional.binary_cross_entropy(output, labels.view(-1, 1))

        self.context.backward(loss)
        self.context.step_optimizer(self.opt_a)

        return {"loss": loss}

    def evaluate_batch(self, batch: pytorch.TorchData, model: nn.Module) -> Dict[str, Any]:
        assert self.context.models
        assert self.context.optimizers
        assert self.context.lr_schedulers

        data, labels = batch
        output = self.model_a(data)
        loss = error_rate(output, labels)

        return {"loss": loss}


class XORTrialGradClipping(XORTrial):
    def train_batch(
        self, batch: pytorch.TorchData, model: nn.Module, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        data, labels = batch
        output = self.model(data)
        loss = torch.nn.functional.binary_cross_entropy(output, labels.view(-1, 1))

        self.context.backward(loss)

        if "gradient_clipping_l2_norm" in self.context.get_hparams():
            self.context.step_optimizer(
                self.optimizer,
                clip_grads=lambda params: torch.nn.utils.clip_grad_norm_(
                    params, self.context.get_hparam("gradient_clipping_l2_norm")
                ),
            )

        elif "gradient_clipping_value" in self.context.get_hparams():
            self.context.step_optimizer(
                self.optimizer,
                clip_grads=lambda params: torch.nn.utils.clip_grad_value_(
                    params, self.context.get_hparam("gradient_clipping_value")
                ),
            )

        else:
            self.context.step_optimizer(self.optimizer)

        return {"loss": loss}
