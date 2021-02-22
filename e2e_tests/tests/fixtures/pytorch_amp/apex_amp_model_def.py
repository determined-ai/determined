"""
This example demonstrates how to modify a model to use PyTorch's support for
NVIDIA APEX in Determined.
"""

from model_def import MNistTrial

from determined.pytorch import PyTorchTrialContext


class MNistApexAMPTrial(MNistTrial):
    def __init__(self, context: PyTorchTrialContext) -> None:
        super().__init__(context)
        self.model, self.optimizer = self.context.configure_apex_amp(models=self.model, optimizers=self.optimizer)

