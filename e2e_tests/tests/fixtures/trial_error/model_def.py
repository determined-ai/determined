"""
This example raises an error from user code.
"""
from typing import Any

from determined.pytorch import DataLoader, PyTorchTrial, PyTorchTrialContext, TorchData


class ErrorTrial(PyTorchTrial):
    def __init__(self, context: PyTorchTrialContext) -> None:
        raise NotImplementedError

    def build_training_data_loader(self) -> DataLoader:
        raise NotImplementedError

    def build_validation_data_loader(self) -> DataLoader:
        raise NotImplementedError

    def train_batch(self, batch: TorchData, epoch_idx: int, batch_idx: int) -> Any:
        raise NotImplementedError
