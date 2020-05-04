"""
This example demonstrates training a simple DNN with pytorch using the Determined
Native API.
"""
import argparse
import json
import pathlib

from determined import experimental
import determined as det

import model_def


def run_trial(runtime_config=None, mode='local'):
    config = {
        "data": {
            "url": "https://s3-us-west-2.amazonaws.com/determined-ai-test-data/pytorch_mnist.tar.gz"
        },
        "hyperparameters": {
            "learning_rate": det.Log(minval=-3.0, maxval=-1.0, base=10),
            "dropout": det.Double(minval=0.2, maxval=0.8),
            "global_batch_size": det.Constant(value=64),
            "n_filters1": det.Constant(value=32),
            "n_filters2": det.Constant(value=32),
        },
        "searcher": {
            "name": "single",
            "metric": "validation_error",
            "max_steps": 20,
            "smaller_is_better": True,
        },
    }

    if config:
        config.update(json.loads(runtime_config))

    experimental.create(
        trial_def=model_def.MNistTrial,
        config=config,
        mode=experimental.Mode(mode),
        context_dir=str(pathlib.Path.cwd()),
    )


if __name__ == "__main__":
    run_trial()
