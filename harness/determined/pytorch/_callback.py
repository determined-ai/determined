from typing import Any, Dict, Iterator

import torch


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
        (except for :meth:`on_validation_step_end` and :meth:`on_checkpoint_end`).  To
        configure a callback implementation to execute on a subset of GPUs, please condition
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

        .. warning::
            If distributed training is enabled, every GPU will execute a copy of
            this callback at the end of every training step. If
            ``optimizations.average_training_metrics`` is enabled, then the
            ``metrics`` will be averaged across all GPUs before the callback is
            executed.  If ``optimizations.average_training_metrics`` is
            disabled, then the ``metrics`` will be local to the GPU.
        """
        pass

    def on_before_optimizer_step(self, parameters: Iterator) -> None:
        """
        Run before every before `optimizer.step()`.  For multi-GPU training, executes
        after gradient updates have been communicated. Typically used to perform gradient
        clipping.
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
        Load the state of this using the deserialized ``state_dict``.
        """
        pass


class ClipGradsL2Norm(PyTorchCallback):
    """
    Callback that performs gradient clipping using
    `L2 Norm <https://pytorch.org/docs/stable/nn.html#clip-grad-norm>`_.
    """

    def __init__(self, clip_value: float) -> None:
        self._clip_value = clip_value

    def on_before_optimizer_step(self, parameters: Iterator) -> None:
        torch.nn.utils.clip_grad_norm_(parameters, self._clip_value)  # type: ignore


class ClipGradsL2Value(PyTorchCallback):
    """
    Callback that performs gradient clipping using
    `L2 Value <https://pytorch.org/docs/stable/nn.html#clip-grad-value>`_.
    """

    def __init__(self, clip_value: float) -> None:
        self._clip_value = clip_value

    def on_before_optimizer_step(self, parameters: Iterator) -> None:
        torch.nn.utils.clip_grad_value_(parameters, self._clip_value)  # type: ignore
