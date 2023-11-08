import typing

from train import MNistTrial

from determined.pytorch import PyTorchTrialContext


class MNistTrialStopRequested(MNistTrial):
    def __init__(self, context: PyTorchTrialContext, hparams: typing.Optional[typing.Dict]) -> None:
        context.set_stop_requested(True)
        super().__init__(context=context, hparams=hparams)
