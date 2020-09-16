"""
This example demonstrates training a simple CNN with tf.keras using the Determined
Native API.
"""
import argparse
import json
import pathlib

import determined as det
from determined import experimental


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
        "hyperparameters": {
            "global_batch_size": det.Constant(value=32),
            "dense1": det.Constant(value=128),
        },
        "records_per_epoch": 50000,
        "searcher": {
            "name": "single",
            "metric": "val_accuracy",
            "max_length": {
                "epochs": 5,
            }
        },
        "entrypoint": "model_def:FashionMNISTTrial"
    }
    config.update(json.loads(args.config))

    experimental.create(
        config=config,
        local=args.local,
        test=args.test,
        context_dir=str(pathlib.Path.cwd()),
    )
