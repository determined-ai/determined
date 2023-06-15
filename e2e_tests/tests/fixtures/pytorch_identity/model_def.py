from typing import Any, Dict, Tuple

import torch.utils.data

from determined import pytorch


class MetricsCallback(pytorch.PyTorchCallback):
    def __init__(self):
        self.validation_metrics = []

    def on_validation_end(self, metrics: Dict[str, Any]) -> None:
        self.validation_metrics.append(metrics)


class IdentityDataset(torch.utils.data.Dataset):
    def __init__(self, initial_value: int = 1):
        self.initial_value = initial_value

    def __len__(self) -> int:
        return 64

    def __getitem__(self, index: int) -> Tuple:
        v = float(self.initial_value + 0.1 * index)
        return torch.Tensor([v]), torch.Tensor([v])


class IdentityPyTorchTrial(pytorch.PyTorchTrial):
    def __init__(self, context: pytorch.PyTorchTrialContext) -> None:
        self.context = context

        model = torch.nn.Linear(1, 1, False)
        model.weight.data.fill_(0)
        self.model = context.wrap_model(model)

        self.lr = 0.001

        optimizer = torch.optim.SGD(self.model.parameters(), self.lr)
        self.opt = context.wrap_optimizer(optimizer)

        self.loss_fn = torch.nn.MSELoss(reduction="mean")
        self.metrics_callback = MetricsCallback()

    def train_batch(
        self, batch: pytorch.TorchData, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        data, label = batch

        loss = self.loss_fn(self.model(data), label)

        self.context.backward(loss)

        self.context.step_optimizer(self.opt)

        return {
            "loss": loss,
        }

    def evaluate_batch(self, batch: pytorch.TorchData) -> Dict[str, Any]:
        data, label = batch

        loss = self.loss_fn(self.model(data), label)

        weight = self.model.weight.data.item()

        return {"val_loss": loss, "weight": weight}

    def build_training_data_loader(self) -> pytorch.DataLoader:
        return pytorch.DataLoader(
            IdentityDataset(), batch_size=self.context.get_per_slot_batch_size()
        )

    def build_validation_data_loader(self) -> pytorch.DataLoader:
        return pytorch.DataLoader(
            IdentityDataset(20), batch_size=self.context.get_per_slot_batch_size()
        )

    def build_callbacks(self) -> Dict[str, pytorch.PyTorchCallback]:
        return {"metrics": self.metrics_callback}
