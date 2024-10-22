import argparse
import logging
import yaml

from attrdict import AttrDict

import determined as det
from determined import pytorch
from determined.pytorch import deepspeed as det_ds

from model_def import DCGANTrial


def parse_args():
    parser = argparse.ArgumentParser()

    # Optional arguments for the launch helper
    parser.add_argument("--local",
                        type=bool,
                        default=False,
                        help="Change behavior for local vs on-cluster"
                             "training")
    return parser.parse_args()

def main(config_file: str, local: bool=True):

    info = det.get_cluster_info()

    if local:
        # For convenience, use hparams from const.yaml for local mode.
        with open(config_file, "r") as f:
            experiment_config = yaml.load(f, Loader=yaml.SafeLoader)
        hparams = AttrDict(experiment_config["hyperparameters"])
        data_config = AttrDict(experiment_config["data"])
        max_length = pytorch.Batch(100)  # Train for 100 batches.
        latest_checkpoint = None
    else:
        print("Cluster training")
        hparams = AttrDict(info.trial.hparams)
        data_config = AttrDict(info.trial._config['data'])
        max_length = None  # On-cluster training trains for the searcher's configured length.
        latest_checkpoint = (
            info.latest_checkpoint
        )  # (Optional) Configure checkpoint for pause/resume functionality.

    with det_ds.init() as train_context:
        trial = DCGANTrial(train_context, hparams, data_config)
        trainer = det_ds.Trainer(trial, train_context)
        trainer.fit(max_length=max_length, latest_checkpoint=latest_checkpoint)

if __name__ == "__main__":
    args = parse_args()
    # Configure logging
    logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)
    main("mnist.yaml", local=args.local)
