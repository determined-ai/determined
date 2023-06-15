"""
Report steps, validations and training in all possible combinations, on each step.
"""
import itertools

import determined as det

with det.core.init() as core_context:
    actions = ["train", "val", "ckpt"]
    permutations_of_combinations_actions = []
    for size in range(len(actions) + 1):
        for c in itertools.combinations(actions, size):
            for p in itertools.permutations(c):
                permutations_of_combinations_actions.append(p)

    step = 0
    for action_set in permutations_of_combinations_actions:
        for action in action_set:
            if action == "train":
                core_context.train.report_validation_metrics(
                    steps_completed=step, metrics={"x": step}
                )
            elif action == "val":
                core_context.train.report_training_metrics(
                    steps_completed=step, metrics={"x": step}
                )
            elif action == "ckpt":
                checkpoint_metadata = {"steps_completed": step}
                with core_context.checkpoint.store_path(checkpoint_metadata) as (path, uuid):
                    with path.joinpath("state").open("w") as f:
                        f.write(f"step: {step}")
        step += 1
