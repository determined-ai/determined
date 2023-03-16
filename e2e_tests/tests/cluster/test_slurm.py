import sys
from typing import List

import pytest
import torch

from determined.common.api.bindings import experimentv1State
from tests import config as conf
from tests import experiment as exp


def run_failure_test_multiple(config_file: str, model_def_file: str, errors: List[str]) -> int:
    # Creates an experiment meant to fail and checks array of error messages
    # If one of the errors are present, then the assertion passes
    experiment_id = exp.create_experiment(
        config_file,
        model_def_file,
    )
    exp.wait_for_experiment_state(experiment_id, experimentv1State.STATE_ERROR, max_wait_secs=600)
    trials = exp.experiment_trials(experiment_id)
    for t in trials:
        trial = t.trial
        if trial.state != experimentv1State.STATE_ERROR:
            continue

        logs = exp.trial_logs(trial.id)
        totalAssertion = False
        for e in errors:
            totalAssertion = totalAssertion or any(e in line for line in logs)
        if not totalAssertion:
            print("******** Start of logs for trial {} ********".format(trial.id), file=sys.stderr)
            print("".join(logs), file=sys.stderr)
            print("******** End of logs for trial {} ********".format(trial.id), file=sys.stderr)
            newline = "\n"
            print(
                f"Trial {trial.id} log did not contain any of the "
                + f"expected messages:{newline.join(errors)}",
                file=sys.stderr,
            )
        assert totalAssertion
    return experiment_id


@pytest.mark.e2e_slurm
def test_unsupported_option() -> None:
    # Creates an experiment with a yaml file
    # It attempts to supply a slurm option that is controlled by Determined
    # run_failure_test expects the experiment to fail and will assert the log with the string
    # Queries the logs for the error call
    # Waits for experiment to reach a ERROR_STATE. Errors if it does not error
    exp.run_failure_test(
        conf.fixtures_path("failures/unsupported-slurm-option.yaml"),
        conf.fixtures_path("failures/"),
        "resources failed with non-zero exit code: unable to launch job: "
        + "slurm option -G is not configurable",
    )


@pytest.mark.e2e_slurm
def test_docker_image() -> None:
    # Creates an experiment with a bad docker image file that will error
    errors = [
        # Singularity message
        "FATAL:   Unable to handle docker://docker.io/badhost/missing.image uri: "
        + "failed to get checksum for docker://docker.io/badhost/missing.image",
        # PodMan message
        "Error: initializing source docker://badhost/missing.image:latest",
        # Enroot message is:
        # `No such file or directory:
        # /home/launcher/.local/share/enroot/docker.io+badhost+missing.image`
        # But error message does not support patterning, so just match the partial file name.
        ".local/share/enroot/docker.io+badhost+missing.image",
    ]

    run_failure_test_multiple(
        conf.fixtures_path("failures/bad-image.yaml"), conf.fixtures_path("failures/"), errors
    )


# Without GPUs, this test may hang and eventually gets a timeout instead of the quick
# failure that is intended.  This is the current behavior on mosaic, so for now skip without GPUs.
@pytest.mark.skipif(not torch.cuda.is_available(), reason="no gpu available")
@pytest.mark.e2e_slurm
def test_node_not_available() -> None:
    # Creates an experiment with a configuration that cannot be satisfied.
    # Verifies that the error message includes the SBATCH options of the failed submission.
    # Only casablanca displays the SBATCH options. Horizon does not upon failure
    # The line: "SBATCH options:" is not present on horizon's output
    error1 = "Batch job submission failed: Requested node configuration is not available"
    # When doing CPU scheduling, on some Slurm systems you may get a different error
    error2 = "CPU count per node can not be satisfied"
    errors = [error1, error2]
    run_failure_test_multiple(
        conf.fixtures_path("failures/slurm-requested-node-not-available.yaml"),
        conf.fixtures_path("failures/"),
        errors,
    )


@pytest.mark.e2e_slurm
def test_bad_slurm_option() -> None:
    # Creates an experiment that uses an invalid slurm option.
    # Only casablanca displays the SBATCH options. Horizon does not upon failure
    # The line: "SBATCH options:" is not present on horizon's output
    exp.run_failure_test(
        conf.fixtures_path("failures/bad-slurm-option.yaml"),
        conf.fixtures_path("failures/"),
        "sbatch: unrecognized option",
    )


@pytest.mark.e2e_slurm_internet_connected_cluster
def test_docker_login() -> None:
    # Creates an experiment that references a valid docker image,
    # but it fails to download due to the lack of a docker login.
    # There are two potential errors that may occur. This is due
    # to docker download rate limitations. The error that occurs
    # depends on the amount of docker downloads on a given day.
    # The Docker error is split into two errors due to slightly
    # different potential outputs from the launcher.

    errorPermission = (
        "latest in docker.io/ilumb/mylolcow: errors: "
        + "denied: requested access to the resource is denied"
    )
    errorDocker = "lstat /root/.config/containers/registries.conf.d: permission denied"
    errors = [errorPermission, errorDocker]
    run_failure_test_multiple(
        conf.fixtures_path("failures/docker-login-failure.yaml"),
        conf.fixtures_path("failures/"),
        errors,
    )


# A devcluster needs to be run with the master host entered incorrectly.
@pytest.mark.e2e_slurm_misconfigured
def test_master_host() -> None:
    # Creates an experiment normally, should error if the back communication channel is broken
    exp.run_failure_test(
        conf.fixtures_path("metric_maker/const.yaml"),
        conf.fixtures_path("metric_maker"),
        "Failed to download model definition from master.  This may be due to an address\n"
        + "resolution problem, a certificate problem, a firewall problem, or some other\n"
        + "networking error.",
    )


@pytest.mark.e2e_slurm
def test_cifar10_pytorch_distributed() -> None:
    config = conf.load_config(conf.cv_examples_path("cifar10_pytorch/distributed.yaml"))
    config["searcher"]["max_length"] = {"epochs": 1}
    config["records_per_epoch"] = 5000
    config["max_restarts"] = 0

    exp.run_basic_test_with_temp_config(config, conf.cv_examples_path("cifar10_pytorch"), 1)


# Test to ensure that determined is able to handle preemption gracefully when using dispatcher RM.
# Preemption:
# When users launch a set of experiments requesting different levels of priorities
# and resources, and when there are a limited set of resources, high priority experiments can
# cancel or requeue low priority experiments.
# Preemption is dependent upon the underlying HPC system and the WLM (SLURM/PBS) setup.
# Nodes in an HPC systems are typically divided into multiple partitions (logical grouping of
# nodes into possibly overlapping sets) used for different purposes. Using WLMs sysadmins
# typically assign varying levels of priority for each partition. Also, users can request the
# WLM to provide specific partition and priority level for their jobs.
# In the following test case we test an example preemption scenario. We launch the two experiments
# cifar10_pytorch_cancellable and cifar10_pytorch_high_priority in order. Ensure that the
# cifar10_pytorch_cancellable experiment is requeued, cifar10_pytorch_high_priority experiment
# runs to completion. After that, cifar10_pytorch_cancellable experiment is resumed and it runs
# to completion.
# NB: The clusters casablanca-login and znode have one node (8-GPUs) being used in two partitions:
#   1. defq_GPU_cancellable - partition for low priority and jobs are requeued if necessary
#   2. defq_GPU_hipri - partition for high priority non-cancellable jobs
@pytest.mark.e2e_slurm_preemption
def test_slurm_preemption() -> None:
    # Launch the cifar10_pytorch_cancellable experiment requesting 8 GPUs on defq_GPU_cancellable
    # partition
    cancelable_exp_id = exp.create_experiment(
        conf.cv_examples_path("cifar10_pytorch/cifar10_pytorch_cancelable.yaml"),
        conf.cv_examples_path("cifar10_pytorch"),
        None,
    )
    exp.wait_for_experiment_state(cancelable_exp_id, experimentv1State.STATE_RUNNING)
    # Launch the cifar10_pytorch_high_priority experiment requesting 8 GPUs on defq_GPU_hipri
    # partition
    high_priority_exp_id = exp.create_experiment(
        conf.cv_examples_path("cifar10_pytorch/cifar10_pytorch_high_priority.yaml"),
        conf.cv_examples_path("cifar10_pytorch"),
        None,
    )
    # In this scenario, cifar10_pytorch_high_priority experiment will cause the
    # cifar10_pytorch_cancelable experiment to get requeued. The experiment
    # cifar10_pytorch_high_priority will execute to completion.
    exp.wait_for_experiment_state(cancelable_exp_id, experimentv1State.STATE_QUEUED)
    exp.wait_for_experiment_state(high_priority_exp_id, experimentv1State.STATE_RUNNING)
    exp.wait_for_experiment_state(high_priority_exp_id, experimentv1State.STATE_COMPLETED)
    # Now, the experiment cifar10_pytorch_cancelable will resume as soon as the requested
    # resources are available.
    exp.wait_for_experiment_state(cancelable_exp_id, experimentv1State.STATE_RUNNING)
    # Finally, the experiment cifar10_pytorch_cancelable will complete if there are no other
    # interruptions.
    exp.wait_for_experiment_state(cancelable_exp_id, experimentv1State.STATE_COMPLETED)
