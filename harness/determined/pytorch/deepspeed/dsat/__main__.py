import argparse
import os
import pathlib
import pickle
import tempfile

from determined.experimental import client
from determined.pytorch.deepspeed.dsat import _defaults, _utils


def parse_args() -> argparse.Namespace:
    # TODO: Allow for additional includes args to be specified, as in the CLI.
    parser = argparse.ArgumentParser(description="DS Autotuning")
    parser.add_argument("config_path")
    parser.add_argument("model_dir")
    parser.add_argument("-i", "--include", type=str, nargs="+")

    parser.add_argument(
        "-t", "--tuner-type", type=str, default=_defaults.AUTOTUNING_ARG_DEFAULTS["tuner-type"]
    )
    parser.add_argument(
        "-mt", "--max-trials", type=int, default=_defaults.AUTOTUNING_ARG_DEFAULTS["max-trials"]
    )
    parser.add_argument("-ms", "--max-slots", type=int)
    parser.add_argument(
        "-mct",
        "--max-concurrent-trials",
        type=int,
        default=_defaults.AUTOTUNING_ARG_DEFAULTS["max-concurrent-trials"],
    )
    parser.add_argument("-es", "--early-stopping", type=int)
    parser.add_argument("-sc", "--search-runner-config", type=str)
    parser.add_argument("-msrr", "--max-search-runner-restarts", type=int)
    parser.add_argument(
        "-z",
        "--zero-stages",
        type=int,
        nargs="+",
        default=_defaults.AUTOTUNING_ARG_DEFAULTS["zero-stages"],
        choices=list(range(4)),
    )
    # Searcher specific args (TODO: refactor)
    parser.add_argument(
        "-trc",
        "--trials-per-random-config",
        type=int,
        default=_defaults.AUTOTUNING_ARG_DEFAULTS["trials-per-random-config"],
    )

    # DS-specific args.
    parser.add_argument(
        "-sps",
        "--start_profile-step",
        type=int,
        default=_defaults.AUTOTUNING_ARG_DEFAULTS["start-profile-step"],
    )
    parser.add_argument(
        "-eps",
        "--end-profile-step",
        type=int,
        default=_defaults.AUTOTUNING_ARG_DEFAULTS["end-profile-step"],
    )
    parser.add_argument(
        "-m",
        "--metric",
        type=str,
        default=_defaults.AUTOTUNING_ARG_DEFAULTS["metric"],
        choices=_defaults.SMALLER_IS_BETTER_METRICS + _defaults.LARGER_IS_BETTER_METRICS,
    )
    parser.add_argument(
        "-r",
        "--random_seed",
        type=int,
        default=42,
    )

    args = parser.parse_args()

    # Convert the paths to absolute paths
    args.config_path = os.path.abspath(args.config_path)
    args.model_dir = os.path.abspath(args.model_dir)
    args.include = [os.path.abspath(p) for p in args.include] if args.include is not None else []

    assert (
        args.tuner_type in _defaults.ALL_SEARCH_METHOD_CLASSES
    ), f"tuner-type must be one of {list(_defaults.ALL_SEARCH_METHOD_CLASSES)}, not {args.tuner_type}"

    return args


def run_autotuning(args: argparse.Namespace) -> None:
    # Build the default SearchRunner's config from the submitted config. The original config yaml file
    # is added as an include and is reimported by the SearchRunner later.

    config = _utils.get_search_runner_config_from_args(args)
    # TODO: early sanity check the submitted config.

    # Create empty tempdir as the model_dir and upload everything else as an includes in order to
    # preserve the top-level model_dir structure inside the SearchRunner's container.

    with tempfile.TemporaryDirectory() as temp_dir:
        # Upload the args, which will be used by the search runner on-cluster.
        args_path = pathlib.Path(temp_dir).joinpath(_defaults.ARGS_PKL_PATH)
        with args_path.open("wb") as f:
            pickle.dump(args, f)
        includes = [args.model_dir, args.config_path] + args.include
        client.create_experiment(config=config, model_dir=temp_dir, includes=includes)


if __name__ == "__main__":
    args = parse_args()
    run_autotuning(args)
