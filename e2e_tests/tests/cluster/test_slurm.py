import pytest

from tests import config as conf
from tests import experiment as exp


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
        "ERROR: resources failed with non-zero exit code: unable to create "
        + "the Slurm launcher manifest: slurm option -G is not configurable",
    )


@pytest.mark.e2e_slurm
def test_docker_image() -> None:
    # Creates an experiment with a bad docker image file that will error
    exp.run_failure_test(
        conf.fixtures_path("failures/bad-image.yaml"),
        conf.fixtures_path("failures/"),
        "FATAL:   Unable to handle docker://missing.image uri: "
        + "failed to get checksum for docker://missing.image",
    )


@pytest.mark.e2e_slurm
def test_node_not_available() -> None:
    # Creates an experiment with a configuration that cannot be satisfied.
    # Verifies that the error message includes the SBATCH options of the failed submission.
    # Only casablanca displays the SBATCH options. Horizon does not upon failure
    # The line: "SBATCH options:" is not present on horizon's output
    exp.run_failure_test(
        conf.fixtures_path("failures/slurm-requested-node-not-available.yaml"),
        conf.fixtures_path("failures/"),
        "Batch job submission failed: Requested node configuration is not available",
    )


@pytest.mark.e2e_slurm
def test_bad_slurm_option() -> None:
    # Creates an experiment that uses an invalid slurm option.
    # Only casablanca displays the SBATCH options. Horizon does not upon failure
    # The line: "SBATCH options:" is not present on horizon's output
    exp.run_failure_test(
        conf.fixtures_path("failures/bad-slurm-option.yaml"),
        conf.fixtures_path("failures/"),
        "ERROR: task failed without an associated exit code: sbatch: unrecognized option",
    )


@pytest.mark.e2e_slurm_internet_connected_cluster
def test_docker_login() -> None:
    # Creates an experiment that references a valid docker image,
    # but it fails to download due to the lack of a docker login.
    # As of writing, FOUNDENG-87 is still in progress. The error will instead be:
    # FATAL:   Unable to handle docker://ilumb/mylolcow uri:
    # failed to get checksum for docker://ilumb/mylolcow:
    # loading registries configuration: reading registries.conf.d:
    # lstat /root/.config/containers/registries.conf.d: permission denied
    exp.run_failure_test(
        conf.fixtures_path("failures/docker-login-failure.yaml"),
        conf.fixtures_path("failures/"),
        "FATAL:   Unable to pull docker://ilumb/mylolcow: conveyor failed to get: "
        + "Error reading manifest latest in docker.io/ilumb/mylolcow: errors: "
        + "denied: requested access to the resource is denied",
    )


# Not possible right now due to incomplete state of circleci runner
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
