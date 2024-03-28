"""
This example demonstrates how to modify a model to use PyTorch's native AMP
(automatic mixed precision) support in Determined.

In the `__init__` method, amp_init() is called (and this accepts parameters to
tune the GradScaler).

The methods `train_batch` and `evaluate_batch` are modified to use an autocast
context during the forward pass.
"""

from typing import Any, Dict, Optional, Sequence, Tuple, Union, cast

import torch
import train
from torch.cuda import amp

from determined import pytorch

TorchData = Union[Dict[str, torch.Tensor], Sequence[torch.Tensor], torch.Tensor]


class MNistManualAMPTrial(train.MNistTrial):
    def __init__(self, context: pytorch.PyTorchTrialContext, hparams: Optional[Dict]) -> None:
        self.scaler = context.wrap_scaler(amp.GradScaler())
        super().__init__(context=context, hparams=hparams)

    def train_batch(
        self, batch: TorchData, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        batch = cast(Tuple[torch.Tensor, torch.Tensor], batch)
        data, labels = batch

        with amp.autocast():
            output = self.model(data)
            loss = torch.nn.functional.nll_loss(output, labels)

        self.context.backward(self.scaler.scale(loss))
        self.context.step_optimizer(self.optimizer, scaler=self.scaler)
        self.scaler.update()

        return {"loss": loss}

    def evaluate_batch(self, batch: TorchData) -> Dict[str, Any]:
        batch = cast(Tuple[torch.Tensor, torch.Tensor], batch)
        data, labels = batch

        with amp.autocast():
            output = self.model(data)
            validation_loss = torch.nn.functional.nll_loss(output, labels).item()

        pred = output.argmax(dim=1, keepdim=True)
        accuracy = pred.eq(labels.view_as(pred)).sum().item() / len(data)

        return {"validation_loss": validation_loss, "accuracy": accuracy}
