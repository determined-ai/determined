from typing import Any, Dict, List, Optional


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

    def on_trial_startup(self, first_batch_idx: int, checkpoint_uuid: Optional[str]) -> None:
        """
        Runs before training, validation, or building dataloaders.

        Arguments:
            first_batch_idx (int):  The first batch index to be trained.  If the trial has already
                completed some amount of training in a previous allocation on the cluster, this will
                be nonzero.
            checkpoint_uuid (str or None):  The checkpoint from which weight, optimizer state, etc
                will be loaded.  When ``first_batch_idx > 0`` this will contain the uuid of the
                most recent checkpoint saved by this trial.  Otherwise, it will contain the uuid of
                the checkpoint from which this trial was configured to warm start from (via
                ``source_trial_id`` or ``source_checkpoint_uuid`` in the searcher config), or None
                if no warm start was configured.
        """
        pass

    def on_trial_shutdown(self) -> None:
        """
        Runs just before shutting down training to get off of the cluster.  This does not imply that
        the trial is complete; it may just be paused or preempted by a higher-priority task.

        .. warning::
            This callback runs each time a Trial shuts down gracefully to come off the cluster.
            This callback does not mean that the Trial is done training.  Additionally, if the trial
            is killed the container will be destroyed without this callback running.
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

    def on_training_start(self) -> None:
        """
        Run after checkpoint loads and before training begins.
        """
        pass

    def on_training_epoch_start(self, epoch_idx: int) -> None:
        """
        Run on start of a new training epoch
        """
        pass

    def on_training_epoch_end(self, epoch_idx: int) -> None:
        """
        Run on end of a training epoch
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
