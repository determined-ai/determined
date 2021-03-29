"""
This example demonstrates how to modify a model to use PyTorch's support for
NVIDIA APEX in Determined.
"""

from model_def import MNISTTrial

from determined.pytorch import PyTorchTrialContext


class MNistApexAMPTrial(MNISTTrial):
    def __init__(self, context: PyTorchTrialContext) -> None:
        super().__init__(context, precision=16, amp_backend="apex")
