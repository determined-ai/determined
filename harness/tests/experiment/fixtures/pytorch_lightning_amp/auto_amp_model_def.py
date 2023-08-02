"""
This example demonstrates how to modify a LightningAdapter model to
use PyTorch's native AMP (automatic mixed precision) support in Determined.
"""

from typing import Dict, Sequence, Union

import torch
from model_def import MNISTTrial

from determined.pytorch import PyTorchTrialContext

TorchData = Union[Dict[str, torch.Tensor], Sequence[torch.Tensor], torch.Tensor]


class MNistAutoAMPTrial(MNISTTrial):
    def __init__(self, context: PyTorchTrialContext) -> None:
        super().__init__(context, precision=16)
