import tempfile
import time

import pytest

from determined.common import yaml
from determined.common.api.bindings import experimentv1State
from tests import config as conf
from tests import experiment as exp


@pytest.mark.e2e_slurm
@pytest.mark.e2e_pbs
def test_hpc_job_pending_reason() -> None:
    # This test only works on slurm/pbs GCP VM with 1 node 8 CPU slots.
    # HPC Resource details: [{TotalAvailableNodes:1 PartitionName:debug IsDefault:true ...
    # TotalAvailableGpuSlots:0 TotalNodes:1 TotalGpuSlots:0
    # TotalAvailableCPUSlots:8 TotalCPUSlots:8 Accelerator:}]

    config = conf.load_config(conf.cv_examples_path("cifar10_pytorch/const.yaml"))
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_slots_per_trial(config, 1)
    config = conf.set_profiling_enabled(config)
    config = conf.set_entrypoint(
        config, "python3 -m determined.launch.horovod --autohorovod --trial model_def:CIFARTrial"
    )
    config["max_restarts"] = 0

    # The experiment will request 6 CPUs
    config.setdefault("slurm", {})
    config["slurm"]["slots_per_node"] = 6
    config.setdefault("pbs", {})
    config["pbs"]["slots_per_node"] = 6

    with tempfile.NamedTemporaryFile() as tf:
        with open(tf.name, "w") as f:
            yaml.dump(config, f)
        running_exp_id = exp.create_experiment(
            tf.name, conf.cv_examples_path("cifar10_pytorch"), None
        )
    print(f"Created experiment {running_exp_id}")
    exp.wait_for_experiment_state(running_exp_id, experimentv1State.RUNNING)

    # Launch another experiment requesting 6 CPUs
    with tempfile.NamedTemporaryFile() as tf:
        with open(tf.name, "w") as f:
            yaml.dump(config, f)
        pending_exp_id = exp.create_experiment(
            tf.name, conf.cv_examples_path("cifar10_pytorch"), None
        )
    print(f"Created experiment {pending_exp_id}")

    exp.wait_for_experiment_state(pending_exp_id, experimentv1State.QUEUED)
    print(f"Experiment {pending_exp_id} pending")

    # Kill the first experiment to shorten the test run. First wait for 60 seconds
    # for the pending job to have a chance to refresh the state and write out the
    # state reason in experiment logs
    time.sleep(60)
    exp.kill_experiments([running_exp_id])

    # Make sure the second experiment will start running after the first experinemt
    # releases the CPUs
    exp.wait_for_experiment_state(pending_exp_id, experimentv1State.RUNNING)
    print(f"Experiment {pending_exp_id} running")

    # Now kill the second experiment to shorten the test run
    exp.kill_experiments([pending_exp_id])

    trials = exp.experiment_trials(pending_exp_id)
    print(f"Check logs for exp {pending_exp_id}")
    slurm_result = exp.check_if_string_present_in_trial_logs(
        trials[0].trial.id,
        "HPC job waiting to be scheduled: The job is waiting for resources to become available.",
    )
    pbs_result = exp.check_if_string_present_in_trial_logs(
        trials[0].trial.id,
        "HPC job waiting to be scheduled: Not Running: Insufficient amount of resource: ncpus ",
    )

    assert pbs_result or slurm_result
