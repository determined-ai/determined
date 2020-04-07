"""
This example raises an error from user code.
"""
from typing import Any, Dict

import torch

import determined as det
from determined.pytorch import DataLoader, PyTorchTrial, TorchData


class ErrorTrial(PyTorchTrial):
    def __init__(self, context: det.TrialContext) -> None:
        self.context = context

    def build_model(self) -> torch.nn.Module:
        raise NotImplementedError

    def optimizer(self, model: torch.nn.Module) -> torch.optim.Optimizer:  # type: ignore
        raise NotImplementedError

    def train_batch(
        self, batch: TorchData, model: torch.nn.Module, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        raise NotImplementedError

    def evaluate_batch(self, batch: TorchData, model: torch.nn.Module) -> Dict[str, Any]:
        raise NotImplementedError

    def build_training_data_loader(self) -> DataLoader:
        raise NotImplementedError

    def build_validation_data_loader(self) -> DataLoader:
        raise NotImplementedError
