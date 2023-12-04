"""
This example demonstrates how to modify a model to use PyTorch's support for
NVIDIA APEX in Determined.
"""
import typing

from train import MNistTrial

from determined.pytorch import PyTorchTrialContext


class MNistApexAMPTrial(MNistTrial):
    def __init__(self, context: PyTorchTrialContext, hparams: typing.Optional[typing.Dict]) -> None:
        super().__init__(context=context, hparams=hparams)
        self.model, self.optimizer = self.context.configure_apex_amp(
            models=self.model, optimizers=self.optimizer
        )
