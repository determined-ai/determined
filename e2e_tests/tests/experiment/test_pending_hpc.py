import tempfile
import time

import pytest

from determined.common import util
from determined.common.api import bindings
from tests import api_utils
from tests import config as conf
from tests import experiment as exp


# Test only works on resource pool with 1 node 8 CPU slots.
# Queries the determined master for resource pool information to determine if
# resource pool is suitable for this test.
def skip_if_not_suitable_resource_pool() -> None:
    session = api_utils.user_session()
    rps = bindings.get_GetResourcePools(session)
    assert rps.resourcePools and len(rps.resourcePools) > 0, "missing resource pool"
    if (
        len(rps.resourcePools) != 1
        or rps.resourcePools[0].slotType != bindings.devicev1Type.CPU
        or rps.resourcePools[0].slotsAvailable != 8
    ):
        errorMessage = "required config: 1 resource pool with 1 node, with only 8 slots."
        pytest.skip(errorMessage)


@pytest.mark.e2e_slurm
@pytest.mark.e2e_pbs
@api_utils.skipif_not_hpc()
def test_hpc_job_pending_reason() -> None:
    skip_if_not_suitable_resource_pool()
    sess = api_utils.user_session()

    config = conf.load_config(conf.tutorials_path("mnist_pytorch/const.yaml"))
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_slots_per_trial(config, 1)
    config = conf.set_profiling_enabled(config)
    config = conf.set_entrypoint(
        config, "python3 -m determined.launch.horovod --autohorovod python3 train.py"
    )
    config["max_restarts"] = 0

    # The experiment will request 6 CPUs
    config.setdefault("slurm", {})
    config["slurm"]["slots_per_node"] = 6
    config.setdefault("pbs", {})
    config["pbs"]["slots_per_node"] = 6

    with tempfile.NamedTemporaryFile() as tf:
        with open(tf.name, "w") as f:
            util.yaml_safe_dump(config, f)
        running_exp_id = exp.create_experiment(
            sess, tf.name, conf.tutorials_path("mnist_pytorch"), None
        )
    print(f"Created experiment {running_exp_id}")
    exp.wait_for_experiment_state(sess, running_exp_id, bindings.experimentv1State.RUNNING)

    # Launch another experiment requesting 6 CPUs
    with tempfile.NamedTemporaryFile() as tf:
        with open(tf.name, "w") as f:
            util.yaml_safe_dump(config, f)
        pending_exp_id = exp.create_experiment(
            sess, tf.name, conf.tutorials_path("mnist_pytorch"), None
        )
    print(f"Created experiment {pending_exp_id}")

    exp.wait_for_experiment_state(sess, pending_exp_id, bindings.experimentv1State.QUEUED)
    print(f"Experiment {pending_exp_id} pending")

    # Kill the first experiment to shorten the test run. First wait for 60 seconds
    # for the pending job to have a chance to refresh the state and write out the
    # state reason in experiment logs
    time.sleep(60)
    exp.kill_experiments(sess, [running_exp_id])

    # Make sure the second experiment will start running after the first experinemt
    # releases the CPUs
    exp.wait_for_experiment_state(sess, pending_exp_id, bindings.experimentv1State.RUNNING)
    print(f"Experiment {pending_exp_id} running")

    # Now kill the second experiment to shorten the test run
    exp.kill_experiments(sess, [pending_exp_id])

    trials = exp.experiment_trials(sess, pending_exp_id)
    print(f"Check logs for exp {pending_exp_id}")
    slurm_result = exp.check_if_string_present_in_trial_logs(
        sess,
        trials[0].trial.id,
        "HPC job waiting to be scheduled: The job is waiting for resources to become available.",
    )
    pbs_result = exp.check_if_string_present_in_trial_logs(
        sess,
        trials[0].trial.id,
        "HPC job waiting to be scheduled: Not Running: Insufficient amount of resource: ncpus ",
    )

    assert pbs_result or slurm_result
