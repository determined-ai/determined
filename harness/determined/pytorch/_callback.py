from typing import Any, Dict


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
        If distributed training is enabled, every GPU will execute a copy of
        this callback (except for on_validation_step_end and on_checkpoint_end).  To configure a
        callback implementation to execute on a subset of GPUs, please condition
        your implementation on ``trial.context.distributed.get_rank()``.
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
            If distributed training is enabled, every GPU will execute a copy of
            this callback at the end of every training step. If
            ``optimizations.average_training_metrics`` is enabled, then the
            ``metrics`` will be averaged across all GPUs before the callback is
            executed.  If ``optimizations.average_training_metrics`` is
            disabled, then the ``metrics`` will be local to the GPU.
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
            This callback only executes on the chief GPU when doing distributed training.
        """
        pass

    def on_checkpoint_end(self, checkpoint_dir: str) -> None:
        """
        Run after every checkpoint.

        .. warning::
            This callback only executes on the chief GPU when doing distributed training.
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
