import logging
import pathlib
from typing import Any, Dict

import determined as det
from determined import core

logger = logging.getLogger("determined")


class TrialContext:
    """
    TrialContext is the system-provided API to a Trial class.
    """

    def __init__(
        self,
        core_context: core.Context,
        env: det.EnvContext,
    ) -> None:
        self._core = core_context
        self.env = env

        self.distributed = self._core.distributed
        self._stop_requested = False

    @classmethod
    def from_config(cls, config: Dict[str, Any]) -> "TrialContext":
        """
        Create a context object suitable for debugging outside of Determined.

        An example for a subclass of :class:`~determined.pytorch.deepspeed.DeepSpeedTrial`:

        .. code-block:: python

            config = { ... }
            context = det.pytorch.deepspeed.DeepSpeedTrialContext.from_config(config)
            my_trial = MyDeepSpeedTrial(context)

            train_ds = my_trial.build_training_data_loader()
            for epoch_idx in range(3):
                for batch_idx, batch in enumerate(train_ds):
                    metrics = my_trial.train_batch(batch, epoch_idx, batch_idx)
                    ...

        An example for a subclass of :class:`~determined.keras._tf_keras_trial.TFKerasTrial`:

        .. code-block:: python

            config = { ... }
            context = det.keras.TFKerasTrialContext.from_config(config)
            my_trial = tf_keras_one_var_model.OneVarTrial(context)

            model = my_trial.build_model()
            model.fit(my_trial.build_training_data_loader())
            eval_metrics = model.evaluate(my_trial.build_validation_data_loader())

        Arguments:
            config: An experiment config file, in dictionary form.
        """
        core_context, env = det._make_local_execution_env(
            managed_training=False,
            test_mode=False,
            config=config,
            checkpoint_dir="/tmp",
            tensorboard_path=pathlib.Path("/tmp/tensorboard"),
            limit_gpus=1,
        )
        return cls(core_context, env)

    def get_experiment_config(self) -> Dict[str, Any]:
        """
        Return the experiment configuration.
        """
        return self.env.experiment_config

    def get_data_config(self) -> Dict[str, Any]:
        """
        Return the data configuration.
        """
        return self.get_experiment_config().get("data", {})

    def get_experiment_id(self) -> int:
        """
        Return the experiment ID of the current trial.
        """
        return int(self.env.det_experiment_id)

    def get_trial_id(self) -> int:
        """
        Return the trial ID of the current trial.
        """
        return int(self.env.det_trial_id)

    def get_trial_seed(self) -> int:
        return self.env.trial_seed

    def get_hparams(self) -> Dict[str, Any]:
        """
        Return a dictionary of hyperparameter names to values.
        """
        return self.env.hparams

    def get_hparam(self, name: str) -> Any:
        """
        Return the current value of the hyperparameter with the given name.
        """
        if name not in self.env.hparams:
            raise ValueError(
                "Could not find name '{}' in experiment "
                "hyperparameters. Please check your experiment "
                "configuration 'hyperparameters' section.".format(name)
            )
        if name == "global_batch_size":
            logger.warning(
                "Please use `context.get_per_slot_batch_size()` and "
                "`context.get_global_batch_size()` instead of accessing "
                "`global_batch_size` directly."
            )
        return self.env.hparams[name]

    def get_stop_requested(self) -> bool:
        """
        Return whether a trial stoppage has been requested.
        """
        return self._stop_requested

    def set_stop_requested(self, stop_requested: bool) -> None:
        """
        Set a flag to request a trial stoppage. When this flag is set to True,
        we finish the step, checkpoint, then exit.
        """
        if not isinstance(stop_requested, bool):
            raise AssertionError("stop_requested must be a boolean")

        logger.info(
            "A trial stoppage has requested. The trial will be stopped "
            "at the end of the current step."
        )
        self._stop_requested = stop_requested

    def get_initial_batch(self) -> int:
        return self.env.steps_completed

    def get_tensorboard_path(self) -> pathlib.Path:
        """
        Get the path where files for consumption by TensorBoard should be written
        """
        return self._core.train.get_tensorboard_path()
