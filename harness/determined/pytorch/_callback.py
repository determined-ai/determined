from typing import Any, Dict, List


class PyTorchCallback:
    """
    Abstract base class used to define a callback that should execute during
    the lifetime of a PyTorchTrial.

    .. warning::
        If you are defining a stateful callback (e.g., it mutates a ``self``
        attribute over its lifetime), you must also override :meth:`state_dict()` and
        :meth:`load_state_dict()` to ensure this state can be serialized and deserialized
        over checkpoints.

    .. warning::
        If distributed training is enabled, every GPU will execute a copy of this callback
        (except for :meth:`on_validation_end`, :meth:`on_validation_step_end` and
        :meth:`on_checkpoint_end`). To configure a callback implementation to execute on a subset of
        GPUs, please condition your implementation on ``trial.context.distributed.get_rank()``.
    """

    def on_validation_start(self) -> None:
        """
        Run before every validation begins.
        """
        pass

    def on_validation_end(self, metrics: Dict[str, Any]) -> None:
        """
        Run after every validation ends.

        .. warning::
            This callback only executes on the chief GPU when doing distributed training.
        """
        pass

    def on_validation_step_start(self) -> None:
        """
        Run before every validation step begins.
        """
        # TODO(DET-3555): remove this once it has been deprecated long enough.
        pass

    def on_validation_step_end(self, metrics: Dict[str, Any]) -> None:
        """
        Run after every validation step ends.

        .. warning::
            This callback only executes on the chief GPU when doing distributed training.
        """
        # TODO(DET-3555): remove this once it has been deprecated long enough.
        pass

    def on_checkpoint_load_start(self, checkpoint: Dict[str, Any]) -> None:
        """
        Run before state_dict is restored.
        """
        pass

    def on_checkpoint_save_start(self, checkpoint: Dict[str, Any]) -> None:
        """
        Run before checkpoint is persisted.
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
        Load the state of this using the deserialized ``state_dict``.
        """
        pass

    def on_training_epoch_start(self) -> None:
        """
        Run on start of a new training epoch
        """
        pass

    def on_validation_epoch_start(self) -> None:
        """
        Run on start of a new validation epoch
        """
        pass

    def on_validation_epoch_end(self, outputs: List[Any]) -> None:
        """
        Run after a new validation epoch has finished
        """
        pass
