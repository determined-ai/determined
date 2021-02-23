"""
This example demonstrates how to modify a model to use PyTorch's native AMP
(automatic mixed precision) support in Determined.

In the `__init__` method, amp_init() is called (and this accepts parameters to
tune the GradScaler).

The methods `train_batch` and `evaluate_batch` are modified to use an autocast
context during the forward pass.
"""

from typing import Any, Dict, Sequence, Tuple, Union, cast

import torch
from model_def import MNistTrial

from determined.pytorch import PyTorchTrialContext

TorchData = Union[Dict[str, torch.Tensor], Sequence[torch.Tensor], torch.Tensor]


class MNistAutoAMPTrial(MNistTrial):
    def __init__(self, context: PyTorchTrialContext) -> None:
        self.scaler = context.experimental.use_amp()
        super().__init__(context)
