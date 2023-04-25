import logging
import os
import pathlib
import pickle

import determined as det
from determined import searcher
from determined.pytorch.deepspeed.dsat import _utils


def main(core_context: det.core.Context) -> None:
    with pathlib.Path("args.pkl").open("rb") as f:
        args = pickle.load(f)
    # On-cluster, the relative paths to the below files just come from the base names.
    args.config_path = os.path.basename(args.config_path)
    args.model_dir = os.path.basename(args.model_dir)
    args.include = [os.path.basename(p) for p in args.include] if args.include is not None else []

    search_method = _utils.get_search_method_from_args(args)
    search_runner = searcher.RemoteSearchRunner(search_method, context=core_context)
    search_runner.run(exp_config=args.config_path, model_dir=args.model_dir, includes=args.include)


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)
    with det.core.init() as core_context:
        main(core_context)
