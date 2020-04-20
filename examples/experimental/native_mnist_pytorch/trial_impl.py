"""
This example demonstrates training a simple DNN with pytorch using the Determined
Native API.
"""
import argparse
import json
import pathlib

import determined as det

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
    config.update(json.loads(args.config))

    det.create(
        trial_def=model_def.MNistTrial,
        config=config,
        mode=det.Mode(args.mode),
        context_dir=str(pathlib.Path.cwd()),
    )
