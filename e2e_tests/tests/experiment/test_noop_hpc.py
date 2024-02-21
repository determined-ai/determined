import os
import tempfile
import time

import pytest

from determined.common import yaml
from determined.common.api import bindings
from tests import api_utils
from tests import config as conf
from tests import experiment as exp


@pytest.mark.e2e_slurm
@pytest.mark.e2e_pbs
@pytest.mark.timeout(20 * 60)
def test_noop_pause_hpc() -> None:
    sess = api_utils.user_session()

    # The original configuration file, which we will need to modify for HPC
    # clusters. We choose a configuration file that will create an experiment
    # that runs long enough to allow us to pause it after the first check
    # point is recorded.  If we choose an experiment that completes too
    # quickly, then by the time we try to pause it, the experiment may be over,
    # so we won't be able to activate (restart) it.
    config_file = conf.fixtures_path("no_op/single-hpc.yaml")

    # Walk through starting, pausing, and resuming a single no-op experiment.
    experiment_id = exp.create_experiment(sess, config_file, conf.fixtures_path("no_op"), None)
    exp.wait_for_experiment_state(sess, experiment_id, bindings.experimentv1State.RUNNING)

    # Wait for the only trial to get scheduled.
    exp.wait_for_experiment_active_workload(sess, experiment_id)

    # Wait for the only trial to show progress, indicating the image is built and running.
    exp.wait_for_experiment_workload_progress(sess, experiment_id)

    # If we pause the experiment before it gets to write at least one checkpoint,
    # then we're really not testing whether the experiment can pick up from where
    # it left off when it's activated.  In which case, the "activate" simply
    # starts from the beginning upon finding that are no checkpoints to start
    # from.  Therefore, wait a while to give the experiment a chance to write at
    # least one checkpoint.
    exp.wait_for_at_least_one_checkpoint(sess, experiment_id)

    # Pause the experiment. Note that Determined does not currently differentiate
    # between a "stopping paused" and a "paused" state, so we follow this check
    # up by ensuring the experiment cleared all scheduled workloads.
    exp.pause_experiment(sess, experiment_id)
    exp.wait_for_experiment_state(sess, experiment_id, bindings.experimentv1State.PAUSED)

    # Wait at most 420 seconds for the experiment to clear all workloads (each
    # train step should take 5 seconds).
    for _ in range(420):
        workload_active = exp.experiment_has_active_workload(sess, experiment_id)
        if not workload_active:
            break
        else:
            time.sleep(1)
    else:
        raise ValueError("The experiment cannot be paused within 420 seconds.")

    # Resume the experiment and wait for completion.
    exp.activate_experiment(sess, experiment_id)
    exp.wait_for_experiment_state(sess, experiment_id, bindings.experimentv1State.COMPLETED)


def remove_item_from_yaml_file(filename: str, item_name: str) -> str:
    with open(filename) as f:
        file_contents = f.read()
        y = yaml.YAML(typ="safe", pure=True)
        data = y.load(file_contents)

        del data[item_name]

    # Create a temporary file that looks something like
    # ${TMPDIR}/single_axh4946j.yaml
    tmpFile = tempfile.NamedTemporaryFile(
        prefix=os.path.splitext(os.path.basename(filename))[0] + "_", suffix=".yaml", delete=False
    )

    y.dump(data, tmpFile)

    return tmpFile.name
