import logging
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
        (except for :meth:`on_validation_end`, :meth:`on_validation_step_end` and
        :meth:`on_checkpoint_end`). To configure a callback implementation to execute on a subset of
        GPUs, please condition your implementation on ``trial.context.distributed.get_rank()``.
    """

    # TODO(DET-3262): remove this backward compatibility of old interface.
    def on_before_optimizer_step(self, parameters: Iterator) -> None:
        """
        Run before every ``optimizer.step()``.  For multi-GPU training, executes
        after gradient updates have been communicated. Typically used to perform
        gradient clipping.

        .. warning::
            This is deprecated. Please pass a function into
            ``context.optimizer.step(clip_gradients=...)`` if you want to clip gradients.
        """
        pass

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


# TODO(DET-3262): remove this backward compatibility of old interface.
class ClipGradsL2Norm(PyTorchCallback):
    """Callback that performs gradient clipping using
    `L2 Norm <https://pytorch.org/docs/stable/nn.html#clip-grad-norm>`_.

    .. warning::

        This is deprecated. Please use clip_grads argument in
        ``PytorchTrialContext.step_optimizer(optimizer, clip_grads=...)``
        for clipping the gradients.
    """

    def __init__(self, clip_value: float) -> None:
        logging.warning(
            "The ClipGradsL2Norm callback is deprecated. Please use clip_grads "
            "argument in PytorchTrialContext.step_optimizer(optimizer, clip_grads=...) "
            "for clipping the gradients."
        )
        self._clip_value = clip_value

    def on_before_optimizer_step(self, parameters: Iterator) -> None:
        torch.nn.utils.clip_grad_norm_(parameters, self._clip_value)  # type: ignore


# TODO(DET-3262): remove this backward compatibility of old interface.
class ClipGradsL2Value(PyTorchCallback):
    """Callback that performs gradient clipping using
    `L2 Value <https://pytorch.org/docs/stable/nn.html#clip-grad-value>`_.

    .. warning::
        This is deprecated. Please use clip_grads argument in
        ``PytorchTrialContext.step_optimizer(optimizer, clip_grads=...)``
        for clipping the gradients.
    """

    def __init__(self, clip_value: float) -> None:
        logging.warning(
            "The ClipGradsL2Value callback is deprecated. Please use clip_grads "
            "argument in PytorchTrialContext.step_optimizer(optimizer, clip_grads=...) "
            "for clipping the gradients."
        )
        self._clip_value = clip_value

    def on_before_optimizer_step(self, parameters: Iterator) -> None:
        torch.nn.utils.clip_grad_value_(parameters, self._clip_value)  # type: ignore
