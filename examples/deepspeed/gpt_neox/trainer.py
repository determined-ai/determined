import logging

import gpt2_trial
import yaml

import determined as det
from determined import pytorch
from determined.pytorch import deepspeed as det_ds


def main(config_file: str, local: bool = True):
    info = det.get_cluster_info()

    if local:
        # For convenience, use hparams from const.yaml for local mode.
        with open(config_file, "r") as f:
            experiment_config = yaml.load(f, Loader=yaml.SafeLoader)
        hparams = experiment_config["hyperparameters"]
        latest_checkpoint = None
    else:
        hparams = info.trial.hparams
        latest_checkpoint = (
            info.latest_checkpoint
        )  # (Optional) Configure checkpoint for pause/resume functionality.

    with det_ds.init() as train_context:
        trial_seed = train_context.get_trial_seed()
        trial = gpt2_trial.GPT2Trial(train_context, hparams, trial_seed)
        trainer = det_ds.Trainer(trial, train_context)
        trainer.fit(max_length=pytorch.Batch(200), latest_checkpoint=latest_checkpoint)


if __name__ == "__main__":
    local = det.get_cluster_info() is None
    # Configure logging
    logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)
    main(config_file="zero1.yaml", local=local)
