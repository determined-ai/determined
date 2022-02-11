from typing import Any, Dict, Tuple

import torch.utils.data

from determined import pytorch


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

    def train_batch(
        self, batch: pytorch.TorchData, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        data, label = batch
        w_before = self.model.weight.data.item()

        n = len(data)
        print(f"n={n}, data={torch.flatten(data)}, label={torch.flatten(label)}")
        loss_exp = sum(((l - d * w_before) ** 2 for d, l in zip(data, label))) / n
        step_exp = 2.0 * self.lr * sum((d * (l - d * w_before) for d, l in zip(data, label))) / n
        w_exp = w_before + step_exp

        loss = self.loss_fn(self.model(data), label)

        self.context.backward(loss)

        gradient = next(self.model.parameters()).grad.item()

        self.context.step_optimizer(self.opt)

        w_after = self.model.weight.data.item()

        return {
            "loss": loss,
            "loss_exp": loss_exp,
            "step_exp": step_exp,
            "gradient": gradient,
            "w_before": w_before,
            "w_after": w_after,
            "w_exp": w_exp,
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
