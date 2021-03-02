import random
from abc import abstractmethod
from typing import Any, cast

import numpy as np
import torch

import determined as det
from determined import horovod
from determined import pytorch_lightning as dl
from determined.horovod import hvd
from determined_common import check


class LightningTrialController(det.LoopTrialController):
    def __init__(self, trial_inst: det.Trial, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)

        check.is_instance(
            trial_inst, LightningTrial, "LightningTrialController needs an LightningTrial"
        )
        self.trial = cast(LightningTrial, trial_inst)
        self.context = cast(dl.LightningTrialContext, self.context)

    @staticmethod
    def pre_execute_hook(env: det.EnvContext, hvd_config: horovod.HorovodContext) -> None:
        if hvd_config.use:
            hvd.require_horovod_type("torch", "LightningTrial is in use.")
            hvd.init()

        LightningTrialController._set_random_seeds(env.trial_seed)

    @staticmethod
    def from_trial(*args: Any, **kwargs: Any) -> det.TrialController:
        return LightningTrialController(*args, **kwargs)

    @staticmethod
    def from_native(*args: Any, **kwargs: Any) -> det.TrialController:
        raise NotImplementedError("LightningTrial only supports the Trial API")

    @staticmethod
    def _set_random_seeds(seed: int) -> None:
        # Set identical random seeds on all training processes.
        # When using horovod, each worker will start at a unique
        # offset in the dataset, ensuring it's processing a unique
        # training batch.
        random.seed(seed)
        np.random.seed(seed)
        torch.random.manual_seed(seed)  # type: ignore

    def run(self) -> None:
        self.trial.train()


class LightningTrial(det.Trial):
    """
    PyTorch Lightning trials are created by subclassing this abstract class.

    We can do the following things in this trial class:

    1. Initialize a trainer by calling ``context.init_trainer`` in :meth:`__init__`.
    2. Start a fitting loop on the initialized trainer by calling ``context.fit``
       in :meth:`train`.
    """

    trial_controller_class = LightningTrialController
    trial_context_class = dl.LightningTrialContext

    @abstractmethod
    def __init__(self, context: dl.LightningTrialContext) -> None:
        """
        Initializes a trial using the provided ``context``. You can initialize
        models, data modules, or any other classes that will be used in training
        in this function. You should initialize a trainer by calling
        ``context.init_trainer``.

        Here is a code example.

        .. code-block:: python

            self.context = context

            self.dm = MyDataModule()
            self.model = MyModel()

            self.context.init_trainer()
        """
        pass

    @abstractmethod
    def train(self) -> None:
        """
        Defines the training procedure. You should start a fitting loop on the
        initialized trainer by calling ``context.fit``.

        Here is a code example.

        .. code-block:: python

            self.context.fit(self.model, datamodule=self.dm)
        """
        pass
