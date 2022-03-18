import copy
import json
import logging
import os
import re
import subprocess
import sys
from typing import Dict

import determined as det
import determined.common
from determined.common import constants, storage


def match_legacy_trial_class(arg: str) -> bool:
    trial_class_regex = re.compile("^[a-zA-Z0-9_.]+:[a-zA-Z0-9_]+$")
    if trial_class_regex.match(arg):
        return True
    return False


def launch(experiment_config: det.ExperimentConfig) -> int:
    slots_per_trial = experiment_config.slots_per_trial()
    entrypoint = experiment_config.get_entrypoint()

    # If native is enabled, harness will load from native entrypoint command
    if experiment_config.native_enabled():
        if slots_per_trial < 2:
            from determined.exec import harness

            return harness.main(train_entrypoint=None)
        else:
            entrypoint = ["python3", "-m", "determined.launch.autohorovod", "__NATIVE__"]

    if not entrypoint:
        raise AssertionError("Entrypoint not found in experiment config")

    if isinstance(entrypoint, str) and match_legacy_trial_class(entrypoint):
        # Legacy entrypoint ("model_def:Trial") detected
        if slots_per_trial < 2:
            # If non-distributed training, continue in non-distributed training mode
            from determined.exec import harness

            return harness.main(train_entrypoint=entrypoint)
        else:
            # Default to horovod if distributed training
            entrypoint = f"python3 -m determined.launch.autohorovod {entrypoint}"

    if isinstance(entrypoint, str):
        entrypoint = ["sh", "-c", entrypoint]

    logging.info(f"Launching: {entrypoint}")

    return subprocess.Popen(entrypoint).wait()


def mask_config_dict(d: Dict) -> Dict:
    mask = "********"
    new_dict = copy.deepcopy(d)

    # checkpoint_storage
    hidden_checkpoint_storage_keys = ("access_key", "secret_key")
    try:
        for key in new_dict["checkpoint_storage"].keys():
            if key in hidden_checkpoint_storage_keys:
                new_dict["checkpoint_storage"][key] = mask
    except KeyError:
        pass

    return new_dict


if __name__ == "__main__":
    info = det.get_cluster_info()
    assert info is not None, "must be run on-cluster"
    assert info.task_type == "TRIAL", f'must be run with task_type="TRIAL", not "{info.task_type}"'

    # Hack: get the resources id from the environment.
    resources_id = os.environ.get("DET_RESOURCES_ID")
    assert resources_id is not None, "Unable to run with DET_RESOURCES_ID unset"

    # Hack: read the full config.  The experiment config is not a stable API!
    experiment_config = det.ExperimentConfig(info.trial._config)

    determined.common.set_logger(experiment_config.debug_enabled())

    logging.info(
        f"New trial runner in (container {resources_id}) on agent {info.agent_id}: "
        + json.dumps(mask_config_dict(info.trial._config))
    )

    # Perform validations
    try:
        logging.info("Validating checkpoint storage ...")
        storage.validate_config(
            experiment_config.get_checkpoint_storage(),
            container_path=constants.SHARED_FS_CONTAINER_PATH,
        )
    except Exception as e:
        logging.error("Checkpoint storage validation failed: {}".format(e))
        sys.exit(1)

    sys.exit(launch(experiment_config))
