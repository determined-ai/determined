"""
This example demonstrates how to train a simple DNN with PyTorch using the
Determined Native API.
"""
import argparse
import json
import pathlib

from determined import experimental
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
    parser.add_argument("--local", action="store_true", help="Specifies local mode")
    parser.add_argument("--test", action="store_true", help="Specifies test mode")
    args = parser.parse_args()

    config = {
        "data": {
            "url": "https://s3-us-west-2.amazonaws.com/determined-ai-test-data/pytorch_mnist.tar.gz"
        },
        "hyperparameters": {
            "learning_rate": det.Log(minval=-3.0, maxval=-1.0, base=10),
            "global_batch_size": det.Constant(value=64),
            "dropout1": det.Double(minval=0.2, maxval=0.8),
            "dropout2": det.Double(minval=0.2, maxval=0.8),
            "n_filters1": det.Constant(value=32),
            "n_filters2": det.Constant(value=32),
        },
        "searcher": {
            "name": "single",
            "metric": "validation_loss",
            "max_steps": 20,
            "smaller_is_better": True,
        },
    }
    config.update(json.loads(args.config))

    experimental.create(
        trial_def=model_def.MNistTrial,
        config=config,
        local=args.local,
        test=args.test,
        context_dir=str(pathlib.Path.cwd()),
    )
