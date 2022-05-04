import copy
import json
import logging
import os
import subprocess
import sys
from typing import Dict

import determined as det
import determined.common
from determined.common import constants, storage


def launch(experiment_config: det.ExperimentConfig) -> int:
    entrypoint = experiment_config.get_entrypoint()

    if isinstance(entrypoint, str) and det.util.match_legacy_trial_class(entrypoint):
        # Legacy entrypoint ("model_def:Trial") detected
        entrypoint = [
            "python3",
            "-m",
            "determined.launch.horovod",
            "--autohorovod",
            "--trial",
            entrypoint,
        ]

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
    except (KeyError, AttributeError):
        pass

    try:
        if new_dict["environment"]["registry_auth"].get("password") is not None:
            new_dict["environment"]["registry_auth"]["password"] = mask
    except (KeyError, AttributeError):
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
