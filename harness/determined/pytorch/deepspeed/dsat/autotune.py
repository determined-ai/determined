import argparse
import os
import tempfile

from determined.experimental import client
from determined.pytorch.deepspeed.dsat import _defaults, _utils
from determined.util import merge_dicts


def parse_args() -> argparse.Namespace:
    # TODO: Allow for additional includes args to be specified, as in the CLI.
    parser = argparse.ArgumentParser(description="DS Autotuning")
    parser.add_argument("-m", "--master", type=str)
    parser.add_argument("-u", "--user", type=str, default="determined")
    parser.add_argument("-p", "--password", type=str, default="")
    parser.add_argument("-s", "--search-runner-config", type=str)
    parser.add_argument("-t", "--tuner-type", type=str, default="random")

    parser.add_argument("config_path")
    parser.add_argument("model_dir")
    args = parser.parse_args()

    assert (
        args.tuner_type in _defaults.ALL_SEARCH_METHOD_CLASSES
    ), f"tuner-type must be one of {list(_defaults.ALL_SEARCH_METHOD_CLASSES)}, not {args.tuner_type}"

    return args


def run_autotuning(args: argparse.Namespace) -> None:
    experiment_config_dict = _utils.get_dict_from_yaml_or_json_path(args.config_path)
    config_path_absolute = os.path.abspath(args.config_path)
    model_dir_absolute = os.path.abspath(args.model_dir)

    # Build the default SearchRunner's config from the submitted config. The original config yaml file
    # is added as an include and is reimported by the SearchRunner later.
    # TODO: Revisit this choice. Might be worth giving the user the ability to specify some parts of
    # the SearchRunner config separately, despite the annoying double-config workflow.
    default_entrypoint = f"python3 -m determined.pytorch.deepspeed.dsat._run_dsat"
    default_entrypoint += (
        f" -c {config_path_absolute} -md {model_dir_absolute} -t {args.tuner_type}"
    )

    default_search_runner_overrides = _defaults.DEFAULT_SEARCH_RUNNER_OVERRIDES
    default_search_runner_overrides["entrypoint"] = default_entrypoint
    default_search_runner_config_dict = merge_dicts(
        experiment_config_dict, default_search_runner_overrides
    )
    default_search_runner_config_dict["name"] += " (DS AT Searcher)"

    # Then merge again with the user provided search runner config, if needed.
    if args.search_runner_config is not None:
        submitted_search_runner_config_dict = _utils.get_dict_from_yaml_or_json_path(
            args.search_runner_config
        )
        search_runner_config_dict = merge_dicts(
            default_search_runner_config_dict, submitted_search_runner_config_dict
        )
    else:
        search_runner_config_dict = default_search_runner_config_dict

    # TODO: early sanity check the submitted config. E.g. make sure that searcher.metric and
    # hyperparameters.ds_config.autotuning.metric coincide.

    # Create empty tempdir as the model_dir and upload everything else as an includes in order to
    # preserve the top-level model_dir structure inside the SearchRunner's container.
    with tempfile.TemporaryDirectory() as temp_dir:
        includes = [model_dir_absolute, config_path_absolute]
        client.create_experiment(
            config=search_runner_config_dict, model_dir=temp_dir, includes=includes
        )


if __name__ == "__main__":
    args = parse_args()
    run_autotuning(args)
