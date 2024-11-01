import enum
import sys
from collections import abc
from typing import Optional, Union

from determined import core


class TrainUnit:
    """
    TrainUnit is the base class for the supported training units (Batch, Epoch) containing
    the value of unit, where the value can be an int or an implementable collections.abc.Container.

    TrainUnits are used to define periodic training behavior such as checkpointing and validating.
    """

    def __init__(self, value: Union[int, abc.Container]):
        self.value = value

    @staticmethod
    def _from_searcher_unit(
        length: int, unit: Optional[core.Unit], global_batch_size: Optional[int] = None
    ) -> "TrainUnit":
        if unit == core.Unit.EPOCHS:
            return Epoch(length)
        elif unit == core.Unit.RECORDS:
            if global_batch_size is None:
                raise ValueError("global_batch_size required for searcher unit Records.")
            return Batch._from_records(length, global_batch_size)
        elif unit == core.Unit.BATCHES:
            return Batch(length)
        else:
            raise ValueError(f"unrecognized searcher unit {unit}")

    def _to_searcher_unit(self) -> core.Unit:
        if isinstance(self, Batch):
            return core.Unit.BATCHES
        return core.Unit.EPOCHS

    @staticmethod
    def _from_values(
        batches: Optional[int] = None,
        records: Optional[int] = None,
        epochs: Optional[int] = None,
        global_batch_size: Optional[int] = None,
    ) -> "TrainUnit":
        if sum((batches is not None, records is not None, epochs is not None)) != 1:
            raise ValueError(f"invalid config: batches={batches} records={records} epochs={epochs}")
        if batches is not None:
            if batches < 1:
                batches = sys.maxsize
            return Batch(batches)
        if records is not None:
            assert global_batch_size, "global_batch_size is required for RECORD units."
            if records < 1:
                records = sys.maxsize
            return Batch._from_records(records, global_batch_size)
        if epochs is not None:
            if epochs < 1:
                epochs = sys.maxsize
            return Epoch(epochs)

        # Make mypy happy
        raise ValueError("invalid values")

    def should_stop(self, step_num: int) -> bool:
        if isinstance(self.value, int):
            return self._divides(step_num)
        assert isinstance(self.value, abc.Container)
        return step_num in self.value

    def _divides(self, steps: int) -> bool:
        assert isinstance(steps, int) and isinstance(
            self.value, int
        ), "_divides can only be called on int types."
        # Treat <= 0 values as always step
        if self.value < 1:
            return True
        if steps == 0:
            return False
        return steps % self.value == 0


class Epoch(TrainUnit):
    """
    Defines an Epoch unit for specifying length to PyTorch trainers.

    Epoch(int) values are treated as periods, e.g. Epoch(100) will checkpoint/validate every 100
    epochs.
    Epoch(collections.abc.Container) values are treated as schedules, e.g. Epoch([1,5,10]) will
    checkpoint/validate on epochs 1, 5, and 10.
    """

    pass


class Batch(TrainUnit):
    """
    Defines a Batch unit for specifying length to PyTorch trainers.

    Batch(int) values are treated as periods, e.g. Batch(100) will checkpoint/validate every 100
    batches.
    Batch(collections.abc.Container) values are treated as schedules, e.g. Batch([1,5,10]) will
    checkpoint/validate on batches 1, 5, and 10.
    """

    @staticmethod
    def _from_records(records: int, global_batch_size: int) -> "Batch":
        return Batch(max(records // global_batch_size, 1))


class _ShouldExit(Exception):
    """
    ShouldExit breaks out of the top-level train loop from inside function calls.
    """

    def __init__(self, skip_exit_checkpoint: bool = False):
        self.skip_exit_checkpoint = skip_exit_checkpoint


class _TrialState:
    def __init__(
        self,
        trial_id: int = 0,
        last_ckpt: int = 0,
        step_id: int = 0,
        last_val: int = 0,
        batches_trained: int = 0,
        epochs_trained: int = 0,
    ) -> None:
        # Store TrialID to distinguish between e.g. pause/restart and continue training.
        self.trial_id = trial_id
        self.last_ckpt = last_ckpt
        self.step_id = step_id
        self.last_val = last_val
        self.batches_trained = batches_trained
        self.epochs_trained = epochs_trained


class _TrainBoundaryType(enum.Enum):
    CHECKPOINT = "CHECKPOINT"
    REPORT = "REPORT"
    VALIDATE = "VALIDATE"
    TRAIN = "TRAIN"


class _TrainBoundary:
    def __init__(self, step_type: _TrainBoundaryType, unit: TrainUnit):
        self.step_type = step_type
        self.unit = unit
        self.limit_reached = False
