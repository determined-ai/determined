"""
This is a script for testing tf-native dtrain with the DeterminedCallback.

Tf-native dtrain depends on environment variables and singletons and such, which makes it hard to
test other than in a totally separate script, executed as a sub-process.

See ./test_callback.py::test_multi_gpu() for additional details.
"""

import argparse

import test_callback

from determined import core


def main(path: str, eager: bool) -> None:
    distributed, strategy = core.DistributedContext.from_tf_config()

    with strategy.scope():
        model = test_callback.build_model(eager=eager)

    events = test_callback.do_fit(path, model=model, distributed=distributed)
    if distributed.rank == 0:
        test_callback.assert_events_match(
            events,
            "!load_model",
            "after_train_begin",
            "set_status:training",
            "set_status:validating",
            "report_metrics:validation",
            "before_epoch_end:0",
            "report_metrics:training",
            "report_progress:0.5000",
            "set_status:checkpointing",
            "save_model",
            "after_epoch_end:0",
            "before_epoch_end:1",
            "report_progress:1.000",
            "save_model",
            "after_epoch_end:1",
            "before_train_end",
            "!save_model",  # No final checkpoint.
            "set_status:finishing",
        )
    else:
        test_callback.assert_events_match(
            events,
            "!load_model",
            "after_train_begin",
            "before_epoch_end:0",
            "save_model",
            "after_epoch_end:0",
            "before_epoch_end:1",
            "save_model",
            "after_epoch_end:1",
            "before_train_end",
            "!save_model",  # No final checkpoint.
        )


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("path")
    parser.add_argument("--eager", action="store_true")
    args = parser.parse_args()

    main(args.path, args.eager)
