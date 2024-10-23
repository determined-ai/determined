import logging
import sys
from typing import List

import pytest
import torch

from determined.common import api
from determined.common.api import bindings
from tests import api_utils, command
from tests import config as conf
from tests import experiment as exp


def run_failure_test_multiple(
    sess: api.Session, config_file: str, model_def_file: str, errors: List[str]
) -> int:
    # Creates an experiment meant to fail and checks array of error messages
    # If one of the errors are present, then the assertion passes
    experiment_id = exp.create_experiment(
        sess,
        config_file,
        model_def_file,
    )
    exp.wait_for_experiment_state(
        sess, experiment_id, bindings.experimentv1State.ERROR, max_wait_secs=600
    )
    trials = exp.experiment_trials(sess, experiment_id)
    for t in trials:
        trial = t.trial
        if trial.state != bindings.trialv1State.ERROR:
            continue

        logs = exp.trial_logs(sess, trial.id)
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
@pytest.mark.e2e_pbs
@api_utils.skipif_not_hpc()
def test_unsupported_option() -> None:
    sess = api_utils.user_session()
    # Creates an experiment with a yaml file
    # It attempts to supply a slurm option that is controlled by Determined
    # run_failure_test expects the experiment to fail and will assert the log with the string
    # Queries the logs for the error call
    # Waits for experiment to reach a ERROR_STATE. Errors if it does not error

    exp.run_failure_test(
        sess,
        conf.fixtures_path("failures/unsupported-slurm-option.yaml"),
        conf.fixtures_path("failures/"),
        "resources failed with non-zero exit code: unable to launch job: "
        + "slurm option -G is not configurable",
    )


@pytest.mark.e2e_slurm
@pytest.mark.e2e_pbs
@api_utils.skipif_not_hpc()
def test_docker_image() -> None:
    sess = api_utils.user_session()
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
        sess, conf.fixtures_path("failures/bad-image.yaml"), conf.fixtures_path("failures/"), errors
    )


# Without GPUs, this test may hang and eventually gets a timeout instead of the quick
# failure that is intended.  This is the current behavior on mosaic, so for now skip without GPUs.
@pytest.mark.skipif(not torch.cuda.is_available(), reason="no gpu available")
@pytest.mark.e2e_slurm
@pytest.mark.e2e_pbs
@api_utils.skipif_not_hpc()
def test_node_not_available() -> None:
    sess = api_utils.user_session()
    # Creates an experiment with a configuration that cannot be satisfied.
    # Verifies that the error message includes the SBATCH options of the failed submission.
    # Only casablanca displays the SBATCH options. Horizon does not upon failure
    # The line: "SBATCH options:" is not present on horizon's output
    error1 = "Batch job submission failed: Requested node configuration is not available"
    # When doing CPU scheduling, on some Slurm systems you may get a different error
    error2 = "CPU count per node can not be satisfied"
    errors = [error1, error2]
    run_failure_test_multiple(
        sess,
        conf.fixtures_path("failures/slurm-requested-node-not-available.yaml"),
        conf.fixtures_path("failures/"),
        errors,
    )


def bad_option_helper(config_path: str, fixture_path: str, error_string: str) -> None:
    sess = api_utils.user_session()
    exp.run_failure_test(
        sess,
        conf.fixtures_path(config_path),
        conf.fixtures_path(fixture_path),
        error_string,
    )


@pytest.mark.e2e_slurm
@api_utils.skipif_not_slurm()
def test_bad_slurm_option() -> None:
    # Creates an experiment that uses an invalid slurm option.
    # Only casablanca displays the SBATCH options. Horizon does not upon failure
    # The line: "SBATCH options:" is not present on horizon's output

    bad_option_helper("failures/bad-slurm-option.yaml", "failures/", "sbatch: unrecognized option")


@pytest.mark.e2e_pbs
@api_utils.skipif_not_pbs()
def test_bad_pbs_option() -> None:
    # Creates an experiment that uses an invalid pbs option.
    bad_option_helper("failures/bad-pbs-option.yaml", "failures/", "qsub: invalid option")


@pytest.mark.e2e_slurm_internet_connected_cluster
@api_utils.skipif_not_slurm()
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
    sess = api_utils.user_session()
    run_failure_test_multiple(
        sess,
        conf.fixtures_path("failures/docker-login-failure.yaml"),
        conf.fixtures_path("failures/"),
        errors,
    )


@pytest.mark.skipif(not torch.cuda.is_available(), reason="no gpu available")
@pytest.mark.e2e_slurm
@pytest.mark.e2e_pbs
@api_utils.skipif_not_hpc()
def test_mnist_pytorch_distributed() -> None:
    sess = api_utils.user_session()
    config = conf.load_config(conf.tutorials_path("mnist_pytorch/distributed.yaml"))
    config["searcher"]["max_length"] = {"epochs": 1}
    config["records_per_epoch"] = 64
    config["max_restarts"] = 0

    exp.run_basic_test_with_temp_config(sess, config, conf.fixtures_path("mnist_pytorch"), 1)


@pytest.mark.e2e_slurm
@pytest.mark.e2e_pbs
@api_utils.skipif_not_hpc()
def test_start_and_verify_hpc_home() -> None:
    """
    Verify that Slurm/PBS jobs retain the user's $HOME directory from the cluster.
    Using a shell we display the value of the HOME variable and verify that
    we don't find /run/determined/workdir in the output which
    is the default for non-HPC jobs.

    For some reason a command (as opposed to a shell) retains the expected
    user $HOME even when the /etc/nsswitch.conf is mis-configured, so this
    test uses a shell.
    """
    sess = api_utils.user_session()

    foundLineWithUserName = False

    with command.interactive_command(sess, ["shell", "start"]) as shell:
        # In order to identify whether we are running a Podman container, we will
        # check if we're running as "root", because Podman containers run as
        # "root".  Use "$(whoami)" to report the user we running as, because the
        # "$USER" environment variable will report the launching user.
        shell.stdin.write(b'COLUMNS=80 echo "USER=$(whoami), HOME=$HOME"\n')
        # Exit the shell, so we can read output below until EOF
        shell.stdin.write(b"exit\n")
        shell.stdin.close()

        for line in shell.stdout:
            logging.info(f"OUT: {line}")

            # We're only interested in the line containing "USER=".
            if "USER=" not in line:
                continue

            foundLineWithUserName = True

            if "USER=root" in line:
                # If we're running as "root" inside the container, it implies
                # we're using Podman as the container run type. For Podman, the
                # home directory in "/run/determined/etc/passwd" is based on the
                # Determined "work_dir" setting, which is
                # "/run/determined/workdir" by default.
                if "HOME=/run/determined/workdir" not in line:
                    pytest.fail(
                        "FAILURE: $HOME in HPC Job is set to an "
                        f"unexpected value for Podman: {line}"
                    )
            else:
                # For Singularity and Enroot, the home directory will be the
                # user's home directory on the host, which is different for each
                # user, so we can't check for a specific value. The best we can
                # do is to check that it's not set to the Determined "work_dir".
                if "HOME=/run/determined/workdir" in line:
                    pytest.fail(
                        "FAILURE: $HOME in HPC Job is set to an "
                        f"unexpected value for Singularity/Enroot: {line}"
                    )

    if not foundLineWithUserName:
        pytest.fail("FAILURE: Did not find line containing the username")
