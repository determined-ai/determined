import tensorflow as tf


class RunHook(tf.estimator.SessionRunHook):  # type: ignore
    """
    Abstract base class which extends
    `SessionRunHook <https://www.tensorflow.org/api_docs/python/tf/estimator/SessionRunHook>`_
    and is used to define callbacks that should execute during the lifetime of a EstimatorTrial.

    Hooks should be passed in to
    `Train Spec <https://www.tensorflow.org/api_docs/python/tf/estimator/TrainSpec>`_.

    """

    def on_checkpoint_load(self, checkpoint_dir: str) -> None:
        """
        Run at startup when the task environment starts up. If not resuming
        from checkpoint this is never called.
        """
        pass

    def on_checkpoint_end(self, checkpoint_dir: str) -> None:
        """
        Run after every checkpoint.

        .. warning::
            If distributed or parallel training is enabled, this callback is executed
            only on the chief GPU (rank = 0) which performs the checkpoint.
        """
        pass

    def on_trial_close(self) -> None:
        """
        Run when the trial close. This is the place users should execute
        post-trial cleanup.

        .. note:
            This callback will execute every time trial stops and when
            it finishes.
        """
        pass
