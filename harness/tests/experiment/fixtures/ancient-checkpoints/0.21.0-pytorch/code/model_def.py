from typing import Any, Callable, Dict, List, Sequence, Tuple, Union, cast

import torch
from torch import nn

import determined as det
from determined import pytorch


class OnesDataset(torch.utils.data.Dataset):
    def __len__(self) -> int:
        return 1024

    def __getitem__(self, index: int) -> torch.Tensor:
        return torch.Tensor([1.0])


class OneVarPytorchTrial(pytorch.PyTorchTrial):
    def __init__(self, context: pytorch.PyTorchTrialContext, lr) -> None:
        self.context = context
        self.per_slot_batch_size = 4
        self.model = context.wrap_model(nn.Linear(1, 1, False))

        # initialize weights to 0
        self.model.weight.data.fill_(0)
        self.opt = context.wrap_optimizer(torch.optim.SGD(self.model.parameters(), lr=lr))

    def train_batch(
        self, batch: pytorch.TorchData, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        loss = torch.nn.MSELoss()(self.model(batch), batch)
        self.context.backward(loss)
        self.context.step_optimizer(self.opt)
        return {"loss": loss}

    def evaluate_batch(self, batch: pytorch.TorchData, batch_idx: int) -> Dict[str, Any]:
        data = labels = batch
        loss = torch.nn.MSELoss()(self.model(data), labels)
        return {"loss": loss}

    def build_training_data_loader(self) -> pytorch.DataLoader:
        return pytorch.DataLoader(OnesDataset(), batch_size=self.per_slot_batch_size)

    def build_validation_data_loader(self) -> pytorch.DataLoader:
        return pytorch.DataLoader(OnesDataset(), batch_size=self.per_slot_batch_size)
