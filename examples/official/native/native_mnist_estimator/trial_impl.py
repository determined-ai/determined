"""
This example demonstrates training a simple DNN with tf.estimator using the Determined
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
    parser.add_argument("--local", action="store_true", help="Specifies local mode")
    parser.add_argument("--test", action="store_true", help="Specifies test mode")
    args = parser.parse_args()

    config = {
        "hyperparameters": {
            "learning_rate": det.Log(-4.0, -2.0, 10),
            "global_batch_size": det.Constant(64),
            "hidden_layer_1": det.Constant(250),
            "hidden_layer_2": det.Constant(250),
            "hidden_layer_3": det.Constant(250),
            "dropout": det.Double(0.0, 0.5),
        },
        "searcher": {
            "name": "single",
            "metric": "accuracy",
            "max_length": {
                "batches": 1000,
            },
            "smaller_is_better": False,
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
