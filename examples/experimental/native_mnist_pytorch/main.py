import argparse
import json

from trial_impl import run_trial

import determined as det

if __name__ == '__main__':
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
    run_trial(args.config, args.mode)
