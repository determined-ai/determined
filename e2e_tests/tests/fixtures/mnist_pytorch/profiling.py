import logging

import train as mnist_pytorch

import determined as det
from determined import pytorch


def run():
    """Initializes the trial and runs the training loop with profiling enabled."""

    info = det.get_cluster_info()
    assert info, "Test must be run on cluster."

    with pytorch.init() as train_context:
        trial = mnist_pytorch.MNistTrial(train_context, hparams=info.trial.hparams)
        trainer = pytorch.Trainer(trial, train_context)
        trainer.configure_profiler(enabled=True)
        trainer.fit(latest_checkpoint=info.latest_checkpoint)


if __name__ == "__main__":
    # Configure logging
    logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)
    run()
