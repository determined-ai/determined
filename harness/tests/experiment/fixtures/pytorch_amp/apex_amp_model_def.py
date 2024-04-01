"""
This example demonstrates how to modify a model to use PyTorch's support for
NVIDIA APEX in Determined.
"""
import typing

import train

from determined import pytorch


class MNistApexAMPTrial(train.MNistTrial):
    def __init__(
        self, context: pytorch.PyTorchTrialContext, hparams: typing.Optional[typing.Dict]
    ) -> None:
        super().__init__(context=context, hparams=hparams)
        self.model, self.optimizer = self.context.configure_apex_amp(
            models=self.model, optimizers=self.optimizer
        )
