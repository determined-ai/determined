import enum
from typing import Any, Dict, List

import torch


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
            STEP_EVERY_OPTIMIZER_STEP
        """

        STEP_EVERY_EPOCH = 1
        STEP_EVERY_BATCH = 2
        MANUAL_STEP = 3
        STEP_EVERY_OPTIMIZER_STEP = 4

    def __init__(
        self,
        scheduler: torch.optim.lr_scheduler._LRScheduler,
        step_mode: StepMode,
        frequency: int = 1,
    ):
        """LRScheduler constructor.

        Args:
            scheduler (:class:`torch.optim.lr_scheduler._LRScheduler`):
                Learning rate scheduler to be used by Determined.
            step_mode (:class:`determined.pytorch.LRSchedulerStepMode`):
                The strategy Determined will use to call (or not call) scheduler.step().

                1. ``STEP_EVERY_EPOCH``: Determined will call scheduler.step() after
                   every ``frequency`` training epoch(s). No arguments will be passed to step().

                2. ``STEP_EVERY_BATCH``: Determined will call scheduler.step() after every
                   ``frequency`` training batch(es). No arguments will be passed to step().
                   This option does not take into account gradient aggregation;
                   ``STEP_EVERY_OPTIMIZER_STEP`` which is recommended.

                3. ``STEP_EVERY_OPTIMIZER_STEP``: Determined will call scheduler.step() in sync
                   with optimizer steps. With ``optimizations.aggregation_frequency`` unset, this
                   is equivalent to ``STEP_EVERY_BATCH``; when it is set, it ensures the LR
                   scheduler is stepped every _effective_ batch.

                   If the option ``frequency`` is set to some value N, Determined will step the LR
                   scheduler every N optimizer steps.

                4. ``MANUAL_STEP``: Determined will not call scheduler.step() at all.
                   It is up to the user to decide when to call scheduler.step(),
                   and whether to pass any arguments.
            frequency:
                Sets the frequency at which the batch and epoch step modes get triggered.
        """
        if not isinstance(step_mode, LRScheduler.StepMode):
            raise TypeError(f"step_mode must be an LRScheduler.StepMode. Got {type(step_mode)}.")

        self._scheduler = scheduler
        self._step_mode = step_mode
        self._frequency = frequency

    def step(self, *args: Any, **kwargs: Any) -> None:
        self._scheduler.step(*args, **kwargs)

    def get_last_lr(self) -> List:
        return self._scheduler.get_last_lr()

    def load_state_dict(self, state_dict: Dict[Any, Any]) -> None:
        self._scheduler.load_state_dict(state_dict)

    def state_dict(self) -> Dict[Any, Any]:
        return self._scheduler.state_dict()
