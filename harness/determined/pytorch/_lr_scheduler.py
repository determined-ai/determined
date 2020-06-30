import enum
from typing import Any, Dict, List

import torch

import determined_common.check as check


class LRScheduler:
    """Wrapper for a PyTorch LRScheduler.

    This wrapper fulfills two main functions:

    1. Save and restore the learning rate when a trial is paused, preempted, etc.
    2. Step the learning rate scheduler at the configured frequency
       (e.g., every batch or every epoch).
    """

    class StepMode(enum.Enum):
        """Specifies when and how scheduler.step() should be executed.

        Attributes:
            STEP_EVERY_EPOCH
            STEP_EVERY_BATCH
            MANUAL_STEP
        """

        STEP_EVERY_EPOCH = 1
        STEP_EVERY_BATCH = 2
        MANUAL_STEP = 3

    def __init__(
        self, scheduler: torch.optim.lr_scheduler._LRScheduler, step_mode: StepMode,
    ):
        """LRScheduler constructor

        Args:
            scheduler (:py:class:`torch.optim.lr_scheduler._LRScheduler`):
                Learning rate scheduler to be used by Determined.
            step_mode (:py:class:`det.pytorch.LRSchedulerStepMode`):
                The strategy Determined will use to call (or not call) scheduler.step().

                1. ``STEP_EVERY_EPOCH``: Determined will call scheduler.step() after
                   every training epoch. No arguments will be passed to step().

                2. ``STEP_EVERY_BATCH``: Determined will call scheduler.step() after every
                   training batch. No arguments will be passed to step().

                3. ``MANUAL_STEP``: Determined will not call scheduler.step() at all.
                   It is up to the user to decide when to call scheduler.step(),
                   and whether to pass any arguments.
        """
        check.check_not_none(scheduler)
        check.check_isinstance(step_mode, LRScheduler.StepMode)

        self._scheduler = scheduler
        self._step_mode = step_mode

    def step(self, *args: Any, **kwargs: Any) -> None:
        self._scheduler.step(*args, **kwargs)

    def get_last_lr(self) -> List:
        return self._scheduler.get_last_lr()  # type: ignore

    def load_state_dict(self, state_dict: Dict[Any, Any]) -> None:
        self._scheduler.load_state_dict(state_dict)

    def state_dict(self) -> Dict[Any, Any]:
        return self._scheduler.state_dict()
