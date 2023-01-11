from model_def import MNistTrial

from determined.pytorch import PyTorchTrialContext


class MNistTrialStopRequested(MNistTrial):
    def __init__(self, context: PyTorchTrialContext) -> None:
        context.set_stop_requested(True)
        super().__init__(context)
