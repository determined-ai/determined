import pathlib
from typing import Any, Dict, cast

from determined import experimental
from determined.pytorch_lightning import CHECKPOINT_FILE_NAME, LightningTrial, LightningTrialContext


def load_model(ckpt_dir: pathlib.Path, metadata: Dict[str, Any], **kwargs: Any) -> LightningTrial:
    trial_cls, trial_context = experimental._load_trial_on_local(
        ckpt_dir.joinpath("code"),
        managed_training=False,
        config=metadata["experiment_config"],
        hparams=metadata["hparams"],
    )

    trial_context = cast(LightningTrialContext, trial_context)
    trial = cast(LightningTrial, trial_cls(trial_context))

    # HACK: use an internal method checkpoint_connector to restore the trainer states.
    trial_context.trainer.checkpoint_connector.restore(  # type: ignore
        checkpoint_path=str(ckpt_dir.joinpath(CHECKPOINT_FILE_NAME)),
        **kwargs,
    )

    return trial
