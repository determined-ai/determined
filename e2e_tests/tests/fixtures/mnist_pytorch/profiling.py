import argparse
import logging

import train as mnist_pytorch

import determined as det
from determined import pytorch


def run(epochs):
    """Initializes the trial and runs the training loop with profiling enabled."""

    info = det.get_cluster_info()
    assert info, "Test must be run on cluster."

    with pytorch.init() as train_context:
        trial = mnist_pytorch.MNistTrial(train_context, hparams=info.trial.hparams)
        trainer = pytorch.Trainer(trial, train_context)
        trainer.fit(
            max_length=pytorch.Epoch(epochs),
            latest_checkpoint=info.latest_checkpoint,
            profiling_enabled=True,
        )


if __name__ == "__main__":
    # Configure logging
    logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)

    parser = argparse.ArgumentParser()
    parser.add_argument("--epochs", type=int, default=1)
    args = parser.parse_args()

    run(args.epochs)
