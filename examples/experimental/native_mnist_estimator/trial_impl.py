"""
This example demonstrates training a simple DNN with tf.estimator using the Determined
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
            "max_steps": 10,
            "smaller_is_better": False,
        },
    }
    config.update(json.loads(args.config))

    det.create(
        trial_def=model_def.MNistTrial,
        config=config,
        mode=det.Mode(args.mode),
        context_dir=str(pathlib.Path.cwd()),
    )
