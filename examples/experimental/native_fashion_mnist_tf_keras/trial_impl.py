"""
This example demonstrates training a simple CNN with tf.keras using the Determined
Native API.
"""
import argparse
import json
import pathlib

import determined as det
from determined import experimental

import model_def


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--config",
        dest="config",
        help="Specifies Determined Experiment configuration.",
        default="{}",
    )
    parser.add_argument(
        "--mode", dest="mode", help="Specifies local mode or cluster mode.", default="cluster"
    )
    args = parser.parse_args()

    config = {
        "hyperparameters": {
            "global_batch_size": det.Constant(value=32),
            "dense1": det.Constant(value=128),
        },
        "searcher": {"name": "single", "metric": "val_accuracy", "max_steps": 40},
    }
    config.update(json.loads(args.config))

    experimental.create(
        trial_def=model_def.FashionMNISTTrial,
        config=config,
        mode=experimental.Mode(args.mode),
        context_dir=str(pathlib.Path.cwd()),
    )
