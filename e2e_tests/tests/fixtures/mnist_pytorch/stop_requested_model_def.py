import logging
import typing

from train import MNistTrial

import determined as det
from determined import pytorch


class MNistTrialStopRequested(MNistTrial):
    def __init__(
        self, context: pytorch.PyTorchTrialContext, hparams: typing.Optional[typing.Dict]
    ) -> None:
        context.set_stop_requested(True)
        super().__init__(context=context, hparams=hparams)


if __name__ == "__main__":
    # Configure logging
    logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)
    info = det.get_cluster_info()
    assert info, "Test must be run on cluster."

    with pytorch.init() as train_context:
        trial = MNistTrialStopRequested(train_context, hparams=info.trial.hparams)
        trainer = pytorch.Trainer(trial, train_context)
        trainer.fit(latest_checkpoint=info.latest_checkpoint)
