from typing import Any, Optional, cast

import torch
import torch.nn as nn

import determined as det
from determined import pytorch
from determined_common import check


class PyTorchTrialContext(det.TrialContext):
    """
    Base context class that contains runtime information for any Determined
    workflow that uses the ``pytorch`` API.
    """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)

        # The following three attributes are initialized during the lifetime of
        # a PyTorchTrialContext.
        self.model = None  # type: Optional[nn.Module]
        self.optimizer = None
        self.lr_scheduler = None  # type: Optional[pytorch.LRScheduler]

    def get_model(self) -> torch.nn.Module:
        """
        Get the model associated with the trial. This function should not be
        called from:

            * ``__init__``
            * ``build_model()``
        """

        check.check_not_none(self.model)
        return cast(torch.nn.Module, self.model)

    def get_optimizer(self) -> torch.optim.Optimizer:  # type: ignore
        """
        Get the optimizer associated with the trial. This function should not be
        called from:

            * ``__init__``
            * ``build_model()``
            * ``optimizer()``
        """
        check.check_not_none(self.optimizer)
        return self.optimizer

    def get_lr_scheduler(self) -> Optional[pytorch.LRScheduler]:
        """
        Get the scheduler associated with the trial, if one is defined. This
        function should not be called from:

            * ``__init__``
            * ``build_model()``
            * ``optimizer()``
            * ``create_lr_scheduler()``
        """
        return self.lr_scheduler
