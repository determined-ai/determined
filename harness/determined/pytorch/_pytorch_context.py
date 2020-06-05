from typing import Any, List, Optional, cast

import torch

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
        self.model = None  # type: Optional[torch.nn.Module]
        self.optimizers = []  # type: List[torch.optim.Optimizer] # type: ignore
        self.lr_schedulers = []  # type: List[pytorch.LRScheduler]

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
        check.len_eq(self.optimizers, 1)
        return self.optimizers[0]

    def _get_optimizers(self) -> List[torch.optim.Optimizer]:  # type: ignore
        """
        Get the optimizer(s) associated with the trial. This function should not be
        called from:

            * ``__init__``
            * ``build_model()``
            * ``build_optimizers()``
        """
        return self.optimizers

    def get_lr_scheduler(self) -> Optional[pytorch.LRScheduler]:
        """
        Get the scheduler associated with the trial, if one is defined. This
        function should not be called from:

            * ``__init__``
            * ``build_model()``
            * ``optimizer()``
            * ``create_lr_scheduler()``
        """
        check.len_eq(self.lr_schedulers, 1)
        return self.lr_schedulers[0]

    def _get_lr_schedulers(self) -> List[pytorch.LRScheduler]:
        """
        Get the scheduler(s) associated with the trial, if one is defined. This
        function should not be called from:

            * ``__init__``
            * ``build_model()``
            * ``build_optimizers()``
            * ``build_lr_schedulers()``
        """
        return self.lr_schedulers
