import os
import subprocess
import time
from shlex import split as sh_split
from typing import Any, Iterator

import pytest

from tests import config as conf

from .abstract_cluster import Cluster
from .utils import now_ts


# ManagedSlurmCluster is an implementation of the abstract class Cluster, to suit a slurm based
# devcluster instance. It is used as part of the e2e slurm tests that require the master to be
# restarted.
class ManagedSlurmCluster(Cluster):
    def __init__(self) -> None:
        self.is_circleci_job = os.getenv("IS_CIRCLECI_JOB")
        self.dc = None
        return

    def __enter__(self) -> "ManagedSlurmCluster":
        self._start_devcluster()
        return self

    def __exit__(self, *_: Any) -> None:
        self.kill_master()
        return

    def kill_master(self) -> None:
        if self.is_circleci_job:
            # Use the pre-installed determined master service when running the tests as part of a
            # CircleCI job.
            subprocess.run(sh_split("sudo systemctl stop determined-master"))
        else:
            # Use the local instance of devcluster.
            if self.dc:
                self.dc.kill()
            self.dc = None
        time.sleep(10)

    def restart_master(self) -> None:
        try:
            self.kill_master()
            self._start_devcluster()
        except Exception as e:
            print(e)
            raise

    def _start_devcluster(self) -> None:
        try:
            if self.is_circleci_job:
                # Use the pre-installed determined master service when running the tests as part
                # of a CircleCI job.
                subprocess.run(sh_split("sudo systemctl start determined-master"))
            else:
                # Use a local instance of the devcluster.
                master_config_file = os.getenv("MASTER_CONFIG_FILE")
                if not master_config_file:
                    raise Exception(
                        "MASTER_CONFIG_FILE is not set. Please set the MASTER_CONFIG_FILE to point "
                        "to the master config file you want to use. Use ./tools/slurmcluster.sh -s "
                        "<machine name> to create a new one."
                    )
                if not os.path.exists(master_config_file):
                    raise Exception(
                        f"Master config file {master_config_file} is missing. Please use "
                        "./tools/slurmcluster.sh -s <machine name> to create one."
                    )
                self.dc = subprocess.Popen(  # type: ignore
                    ["devcluster", "-c", master_config_file, "--oneshot"],
                    cwd="..",
                )
            time.sleep(30)
        except Exception as e:
            print(e)
            raise

    def ensure_agent_ok(self) -> None:
        pass

    def restart_agent(self, wait_for_amnesia: bool = True, wait_for_agent: bool = True) -> None:
        pass


# Create a pytest fixture that returns a ManagedSlurmCluster instance and set it's scope equal as
# session (active for entire duration of the pytest command execution).
@pytest.fixture(scope="session")
def managed_slurm_cluster_session(request: Any) -> Iterator[ManagedSlurmCluster]:
    with ManagedSlurmCluster() as msc:
        yield msc


# Create a pytest fixture that returns a fixture of kind managed_slurm_cluster_session, defined
# above. Additionally, log the timestamp and the nodeid (pytest identifier for each test) before
# and after every test.
@pytest.fixture
def managed_slurm_cluster_restarts(
    managed_slurm_cluster_session: ManagedSlurmCluster, request: Any
) -> Iterator[ManagedSlurmCluster]:
    if os.getenv("IS_CIRCLECI_JOB"):
        # CircleCI job has master running on port 8080
        conf.MASTER_PORT = "8080"
    else:
        # Local instance of devcluster is run on port 8081
        conf.MASTER_PORT = "8081"
    nodeid = request.node.nodeid
    managed_slurm_cluster_session.log_marker(f"pytest [{now_ts()}] {nodeid} setup\n")
    yield managed_slurm_cluster_session
    managed_slurm_cluster_session.log_marker(f"pytest [{now_ts()}] {nodeid} teardown\n")
