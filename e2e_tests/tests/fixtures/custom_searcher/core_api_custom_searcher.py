import argparse
import logging
from typing import List, Optional, Tuple

import searchers
from urllib3 import connectionpool

import determined as det
from determined import searcher
from determined.common import util


def load_config(config_path: str):
    with open(config_path) as f:
        config = util.safe_load_yaml_with_exceptions(f)
    return config


def parse_args():
    parser = argparse.ArgumentParser()
    parser.add_argument("--searcher", choices=["asha", "random"], required=True)
    parser.add_argument("--exp-name", type=str, required=True)
    parser.add_argument("--max-length", type=int, required=True)
    parser.add_argument("--max-trials", type=int, required=True)
    parser.add_argument("--max-concurrent-trials", type=int, default=0)
    parser.add_argument("--divisor", type=int, default=3)
    parser.add_argument("--num-rungs", type=int, default=16)
    parser.add_argument("--exception-points", type=str, nargs="+", default=[])
    parser.add_argument("--config-name", type=str, required=True)
    parser.add_argument("--metric-as-dict", action="store_true", default=False)
    return parser.parse_args()


def create_search_method(args, exception_points: Optional[List[str]] = None):
    if args.searcher == "asha":
        return searchers.ASHASearchMethod(
            max_trials=args.max_trials,
            max_length=args.max_length,
            divisor=args.divisor,
            num_rungs=args.num_rungs,
            exception_points=exception_points,
        )
    elif args.searcher == "random":
        return searchers.RandomSearchMethod(
            max_trials=args.max_trials,
            max_length=args.max_length,
            max_concurrent_trials=args.max_concurrent_trials,
            exception_points=exception_points,
            metric_as_dict=args.metric_as_dict,
        )
    else:
        raise ValueError("Unknown searcher type")


class FallibleSearchRunner(searcher.RemoteSearchRunner):
    def __init__(
        self, search_method: searcher.SearchMethod, core_context: det.core.Context
    ) -> None:
        super(FallibleSearchRunner, self).__init__(search_method, core_context)
        self.fail_on_save = False

    def load_state(self, storage_id: str) -> Tuple[int, List[searcher.Operation]]:
        result = super(FallibleSearchRunner, self).load_state(storage_id)

        # on every load remove first exception from the list
        # since that exception was raised in the previous run;
        # this testing approach works as long as the there is
        # at least one save between consecutive exceptions
        if len(search_method.exception_points) > 0:
            self.search_method.exception_points.pop(0)

        if len(self.search_method.exception_points) > 0:
            if self.search_method.exception_points[0] == "after_save":
                self.fail_on_save = True

        return result

    def save_state(self, experiment_id: int, operations: List[searcher.Operation]) -> None:
        super(FallibleSearchRunner, self).save_state(experiment_id, operations)
        if self.fail_on_save:
            logging.info(
                "Raising exception in after saving the state and before posting operations"
            )
            ex = connectionpool.MaxRetryError(
                connectionpool.HTTPConnectionPool(host="dummyhost", port=8080), "http://dummyurl"
            )
            raise ex


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)

    info = det.get_cluster_info()
    assert info is not None, "this example only runs on-cluster"
    args = parse_args()

    config = load_config(args.config_name)
    config["name"] = args.exp_name
    if args.metric_as_dict:
        config["entrypoint"] += " dict"

    with det.core.init() as core_context:
        search_method = create_search_method(args, args.exception_points)
        search_runner = FallibleSearchRunner(search_method, core_context)
        search_runner.run(config)
