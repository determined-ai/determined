import enum
import typing

import torch

import determined_common.check as check


class LRScheduler:
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

    def __init__(self, scheduler: torch.optim.lr_scheduler._LRScheduler, step_mode: StepMode):
        """Wrapper for a PyTorch LRScheduler.

        Usage of this wrapper is required to properly scheduler the optimizer's learning rate.

        This wrapper fulfills two main functions:
            1. Save and restore of the learning rate in case a trial is paused, preempted, etc.
            2. Step the learning rate scheduler for predefined frequencies
               (every batch or every epoch).

        Args:
            scheduler (:py:class:`torch.optim.lr_scheduler._LRScheduler`):
                Learning rate scheduler to be used by Determined.
            step_mode (:py:class:`det.pytorch.LRSchedulerStepMode`):
                The strategy Determined will use to call (or not call) scheduler.step().

                1. `STEP_EVERY_EPOCH`: Determined will call scheduler.step() after
                   every training epoch. No arguments will be passed to step().

                2. `STEP_EVERY_BATCH`: Determined will call scheduler.step() after every
                   training batch. No arguments will be passed to step().

                3. `MANUAL_STEP`: Determined will not call scheduler.step() at all.
                   It is up to the user to decide when to call scheduler.step(),
                   and whether to pass any arguments.

        """

        check.check_not_none(scheduler)
        check.check_isinstance(step_mode, LRScheduler.StepMode)

        self.scheduler = scheduler
        self.step_mode = step_mode

    def step(self, *args: typing.Any, **kwargs: typing.Any) -> None:
        """Call step() on the wrapped LRScheduler instance.
        """
        check.check_eq(
            self.step_mode,
            LRScheduler.StepMode.MANUAL_STEP,
            "Please use the MANUAL_STEP step mode to call step() on the scheduler.",
        )
        return self.scheduler.step(*args, **kwargs)

    def get_lr(self) -> typing.List:
        """Compute the current learning rate of the scheduler.

        This function is equivalent to calling get_lr() on the wrapped LRScheduler.
        """

        return self.scheduler.get_lr()  # type: ignore


class _LRHelper:
    def __init__(self, lr_scheduler: typing.Optional[LRScheduler]):
        self._lr_scheduler = None
        if lr_scheduler:
            check.check_type(
                lr_scheduler,
                LRScheduler,
                "`create_lr_scheduler` must return a `det.pytorch.LRScheduler`",
            )
            self._lr_scheduler = lr_scheduler
            self._lr_scheduler_count = lr_scheduler.scheduler._step_count  # type: ignore

    def __bool__(self) -> bool:
        return self._lr_scheduler is not None

    def should_step_lr(self, batch_idx: int, epoch_length: int, aggregation_frequency: int) -> bool:
        if self._lr_scheduler:
            if self._lr_scheduler.step_mode == LRScheduler.StepMode.STEP_EVERY_BATCH:
                return True
            elif self._lr_scheduler.step_mode == LRScheduler.StepMode.STEP_EVERY_EPOCH:
                mod = batch_idx % epoch_length
                if mod == 0 or mod < aggregation_frequency:
                    return True
        return False

    def step(self) -> None:
        check.check_eq(
            self._lr_scheduler.scheduler._step_count,  # type: ignore
            self._lr_scheduler_count,
            "You cannot call `scheduler.step()` if you have configured "
            "Determined to manage the learning rate scheduler.",
        )
        self._lr_scheduler.scheduler.step()  # type: ignore
        self._lr_scheduler_count += 1

    def load_state_dict(
        self, state_dict: typing.Optional[typing.Dict[typing.Any, typing.Any]]
    ) -> None:
        if self._lr_scheduler:
            state_dict = typing.cast(typing.Dict[typing.Any, typing.Any], state_dict)
            self._lr_scheduler.scheduler.load_state_dict(state_dict)
            self._lr_scheduler_count = self._lr_scheduler.scheduler._step_count  # type: ignore

    def state_dict(self) -> typing.Dict[typing.Any, typing.Any]:
        self._lr_scheduler = typing.cast(LRScheduler, self._lr_scheduler)
        return self._lr_scheduler.scheduler.state_dict()
