# type: ignore
from typing import Optional

from lightning.pytorch import utilities as ptl_utilities
from lightning.pytorch.loggers import logger as ptl_logger

from determined import experimental
from determined.experimental import core_v2


# TODO(ilia): Expand the integration.
class DetLogger(ptl_logger.Logger):
    def __init__(
        self,
        *,
        defaults: Optional[core_v2.DefaultConfig] = None,
        unmanaged: Optional[core_v2.UnmanagedConfig] = None,
        config: Optional[core_v2.Config] = None,
        client: Optional[experimental.Determined] = None,
    ) -> None:
        self._kwargs = {
            "defaults": defaults,
            "client": client,
            "unmanaged": unmanaged,
            "config": config,
        }
        self._initialized = False

    @property
    @ptl_logger.rank_zero_experiment
    def experiment(self) -> None:
        if not self._initialized:
            core_v2.init(**self._kwargs)
            self._initialized = True

    @property
    def name(self):
        return "DetLogger"

    @property
    def version(self):
        # Return the experiment version, int or str.
        return "0.1"

    @ptl_utilities.rank_zero_only
    def log_hyperparams(self, params):
        # params is an argparse.Namespace
        # your code to record hyperparameters goes here
        pass

    @ptl_utilities.rank_zero_only
    def log_metrics(self, metrics, step):
        # metrics is a dictionary of metric names and values
        # your code to record metrics goes here
        core_v2.train.report_training_metrics(step, metrics)

    @ptl_utilities.rank_zero_only
    def save(self):
        # Optional. Any code necessary to save logger data goes here
        pass

    @ptl_utilities.rank_zero_only
    def finalize(self, status):
        # Optional. Any code that needs to be run after training
        # finishes goes here
        core_v2.close()
