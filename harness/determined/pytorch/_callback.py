from typing import Any, Dict

import torch


class PyTorchCallback:
    """
    Abstract base class used to define a callback that should execute during
    the lifetime of a PyTorchTrial.

    .. warning::
        If you are defining a stateful callback (e.g. it mutates a self
        attribute over it's lifetime), you must also override state_dict() and
        load_state_dict() to ensure this state can be serialized and deserialized
        over checkpoints.

    .. warning::
        If distributed or parallel training is enabled, every shard will
        execute a copy of this callback (except for on_validation_step_end).
        To configure a callback implementation to execute on a subset of shards,
        please condition your implementation on
        ``trial.context.distributed.get_rank()``.
    """

    def on_train_step_start(self, step_id: int) -> None:
        """
        Run before every training step begins.
        """
        pass

    def on_train_step_end(self, step_id: int, metrics: Dict[str, Any]) -> None:
        """
        Run after every training step ends.

        ..warning::
            If distributed or parallel training is enabled, every shard will
            execute a copy of this callback on train step end. If
            ``optimizations.average_training_metrics`` is enabled, then the
            ``metrics`` will be averaged across all shards before the callback
            is executed.  If ``optimizations.average_training_metrics`` is
            disabled, then the ``metrics`` will be local to the shard.
        """
        pass

    def on_validation_step_start(self) -> None:
        """
        Run before every validation step begins.
        """
        pass

    def on_validation_step_end(self, metrics: Dict[str, Any]) -> None:
        """
        Run after every validation step ends.

        .. warning::
            This callback currently only executes on the chief shard in the
            distributed and/or parallel training setting.
        """
        pass

    def state_dict(self) -> Dict[str, Any]:
        """
        Serialize the state of this callback to a dictionary. Return value must
        be pickle-able.
        """
        return {}

    def load_state_dict(self, state_dict: Dict[str, Any]) -> None:
        """
        Load the state of this using the deserialized state_dict.
        """
        pass


class ReduceLROnPlateauEveryValidationStep(PyTorchCallback):
    def __init__(
        self, reduce_lr_on_plateau: torch.optim.lr_scheduler.ReduceLROnPlateau, metric_name: str
    ):
        self.reduce_lr_on_plateau = reduce_lr_on_plateau
        self.metric_name = metric_name

    def on_validation_step_end(self, metrics: Dict[str, Any]) -> None:
        self.reduce_lr_on_plateau.step(metrics[self.metric_name])

    def state_dict(self) -> Dict[str, Any]:
        return self.reduce_lr_on_plateau.state_dict()

    def load_state_dict(self, state_dict: Dict[str, Any]) -> None:
        self.reduce_lr_on_plateau.load_state_dict(state_dict)
