"""
noop/ is a fixture that acts based on the actions you feed it through hparams.

It is useful when you need to test how the master reacts to specific user code behaviors.

Actions are fed in through hyperparameters.actions, which must be a dict of dicts.  The reason for
not using a list of dicts is that our nested hyperparameter logic works on dicts only.

You should not populate your own hparams; you should pass a list of noop.Actions into one of:

  - noop.generate_config(), to create a bare config
  - noop.create_experiment(), to create an experiment
  - noop.cli_config_overrides(), to create --config cli args for overriding hyperparameters
"""

import base64
import logging
import pathlib
import sys
import time
from typing import Iterator, Optional, Tuple

import determined as det
from determined import core


def read_metrics(metrics):
    if metrics == "nan":
        return float("nan")
    if isinstance(metrics, (int, float, str)):
        return float(metrics)
    if isinstance(metrics, list):
        return [read_metrics(m) for m in metrics]
    if isinstance(metrics, dict):
        return {k: read_metrics(v) for k, v in metrics.items()}
    raise ValueError(f"unexpected metrics: {metrics}")


def save_state(action_id, steps_completed, trial_id, checkpoint_directory) -> None:
    with checkpoint_directory.joinpath("state").open("w") as f:
        f.write(f"{action_id},{steps_completed},{trial_id}")


def load_state(trial_id, checkpoint_directory) -> Tuple[int, int]:
    checkpoint_directory = pathlib.Path(checkpoint_directory)
    with checkpoint_directory.joinpath("state").open("r") as f:
        action_id, steps_completed, ckpt_trial_id = [int(field) for field in f.read().split(",")]
    if ckpt_trial_id == trial_id:
        return action_id, steps_completed
    else:
        # This is a new trial; preserve nothing
        return 0, 0


def main(
    core_context: core.Context,
    trial_id: int,
    actions: list,
    latest_checkpoint: Optional[str],
) -> None:
    starting_action_id, steps_completed = 0, 0
    if latest_checkpoint is not None:
        with core_context.checkpoint.restore_path(latest_checkpoint) as path:
            last_action_id, steps_completed = load_state(trial_id, path)
            starting_action_id = last_action_id + 1

    operations = None  # type: Iterator[core.SearcherOperation]

    for action_id, action in enumerate(actions[starting_action_id:], start=starting_action_id):
        logging.info(f"executing {action}")
        if action["action"] == "exit":
            sys.exit(action.get("code", 0))
        elif action["action"] == "sleep":
            time.sleep(action["time"])
        elif action["action"] == "report":
            if action["group"] == "training":
                # pretend we actually did training
                steps_completed += 1
            core_context.train.report_metrics(
                group=action["group"],
                steps_completed=steps_completed,
                metrics=read_metrics(action["metrics"]),
            )
        elif action["action"] == "checkpoint":
            checkpoint_metadata = {"steps_completed": steps_completed}
            with core_context.checkpoint.store_path(checkpoint_metadata) as (path, uuid):
                save_state(action_id, steps_completed, trial_id, path)
        elif action["action"] == "log":
            msg = base64.b64decode(action["base64"]).decode("utf8")
            logging.log(action["level"], msg)
        elif action["action"] == "complete_searcher_operation":
            # Get operations if we haven't already.
            if not operations:
                operations = core_context.searcher.operations(core.SearcherMode.ChiefOnly)
            op = next(operations)
            op.report_completed(action["metric"])
        else:
            raise ValueError(f"unexpected action type: {action}")

        # check if we've been preempted
        if core_context.preempt.should_preempt():
            # save a checkpoint if we didn't just do that
            if action["action"] != "checkpoint":
                checkpoint_metadata = {"steps_completed": steps_completed}
                with core_context.checkpoint.store_path(checkpoint_metadata) as (path, uuid):
                    save_state(action_id, steps_completed, trial_id, path)
            break


if __name__ == "__main__":
    logging.basicConfig(level=logging.DEBUG, format=det.LOG_FORMAT)
    info = det.get_cluster_info()
    assert info
    # Actions is a dictionary and we sort by integer-converted keys to get a list of values.
    actions = [v for _, v in sorted(info.trial.hparams["actions"].items(), key=lambda i: int(i[0]))]
    # We don't actually support dtrain; just kill off anything which isn't the chief.
    if info.container_rank > 0:
        logging.warning("non-chief container exiting now")
        sys.exit(0)
    distributed = core.DistributedContext(
        rank=0,
        size=1,
        local_rank=0,
        local_size=1,
        cross_rank=0,
        cross_size=1,
    )
    with core.init(distributed=distributed) as core_context:
        main(core_context, info.trial.trial_id, actions, info.latest_checkpoint)
