import argparse
import os
import pathlib
import pickle
import tempfile

from determined.experimental import client
from determined.pytorch import dsat
from determined.pytorch.dsat import defaults


def parse_args() -> argparse.Namespace:
    parser = dsat.get_full_parser()
    args = parser.parse_args()
    assert args.max_trials > 1, "--max-trials must be larger than 1"

    # Convert the paths to absolute paths
    args.config_path = os.path.abspath(args.config_path)
    args.model_dir = os.path.abspath(args.model_dir)
    args.include = [os.path.abspath(p) for p in args.include] if args.include is not None else []

    return args


def run_autotuning(args: argparse.Namespace) -> None:
    # Build the default SearchRunner's config from the submitted config. The original
    # config yaml file is added as an include and is reimported by the SearchRunner later.

    config = dsat.get_search_runner_config_from_args(args)

    # Create empty tempdir as the model_dir and upload everything else as an includes in order to
    # preserve the top-level model_dir structure inside the SearchRunner's container.

    with tempfile.TemporaryDirectory() as temp_dir:
        # Upload the args, which will be used by the search runner on-cluster.
        args_path = pathlib.Path(temp_dir).joinpath(defaults.ARGS_PKL_PATH)
        with args_path.open("wb") as f:
            pickle.dump(args, f)
        includes = [args.model_dir, args.config_path] + args.include
        exp = client.create_experiment(config=config, model_dir=temp_dir, includes=includes)
        # Note: Simulating the same print functionality as our CLI when making an experiment.
        # This line is needed for the e2e tests
        print(f"Created experiment {exp.id}")


if __name__ == "__main__":
    args = parse_args()
    run_autotuning(args)
