import os
import shlex
import signal
import subprocess
import time
from typing import Any, Iterator

import pytest

from tests import config as conf
from tests.cluster import abstract_cluster, utils


# ManagedSlurmCluster is an implementation of the abstract class Cluster, to suit a slurm based
# devcluster instance. It is used as part of the e2e slurm tests that require the master to be
# restarted.
class ManagedSlurmCluster(abstract_cluster.Cluster):
    def __init__(self) -> None:
        self.is_circleci_job = False  # DNJ DEBUG os.getenv("CIRCLECI")
        self.gcloud_zone = os.getenv("SLURM_GCLOUD_ZONE")
        self.gcloud_instance_name = os.getenv("SLURM_GCLOUD_INSTANCE_NAME")
        self.gcloud_project = os.getenv("SLURM_GCLOUD_PROJECT")
        self.existing_devcluster_pid = os.getenv("SLURM_DEVCLUSTER_PID")
        print(
            f"DNJ DEBUG fixture start {self.gcloud_zone}, {self.gcloud_instance_name}, {self.gcloud_project}, {self.existing_devcluster_pid}"
        )
        self.dc = None
        self.ssh_cmd = [
            "gcloud",
            "compute",
            "ssh",
            "--zone",
            f"{self.gcloud_zone}",
            f"{self.gcloud_instance_name}",
            "--project",
            f"{self.gcloud_project}",
            "--",
        ]
        return

    def __enter__(self) -> "ManagedSlurmCluster":
        if self.existing_devcluster_pid:
            print(f"DNJ DEBUG killing pid: {self.existing_devcluster_pid}")
            # os.kill(self.existing_devcluster_pid, signal.SIGTERM)
        self._start_devcluster()
        return self

    def __exit__(self, *_: Any) -> None:
        self.kill_master()
        return

    def kill_master(self) -> None:
        print(
            f"DNJ DEBUG killing job {self.gcloud_zone}, {self.gcloud_instance_name}, {self.gcloud_project}"
        )
        if self.is_circleci_job:
            # Use the pre-installed determined master service when running the tests as part of a
            # CircleCI job.
            self._gcloud_ssh("sudo systemctl stop determined-master")
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
                self._gcloud_ssh("sudo systemctl start determined-master")
            else:
                # Use a local instance of the devcluster.
                master_config_file = os.getenv("SLURM_DEVCLUSTER_CONFIG")
                print(f"DNJ DEBUG fixture start {master_config_file}")
                if not master_config_file:
                    raise Exception(
                        "SLURM_DEVCLUSTER_CONFIG is not set. Please set the SLURM_DEVCLUSTER_CONFIG to point "
                        "to the master config file you want to use. Use ./tools/slurmcluster.sh -s "
                        "<machine name> to create a new one."
                    )
                if not os.path.exists(master_config_file):
                    raise Exception(
                        f"Master config file {master_config_file} is missing. Please use "
                        "./tools/slurmcluster.sh -s <machine name> to create one."
                    )
                self.dc = subprocess.Popen(  # type: ignore
                    self.ssh_cmd + ["devcluster", "-c", master_config_file, "--oneshot"],
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

    def _gcloud_ssh(self, cmd: str):
        print(f"DNJ DEBUG GCLOUD SSH GOING WITH COMMAND {cmd}")
        assert self.gcloud_zone, "SLURM_GCLOUD_ZONE must be set for this test!"
        assert self.gcloud_instance_name, "SLURM_GCLOUD_INSTANCE_NAME must be set for this test!"
        assert self.gcloud_project, "SLURM_GCLOUD_PROJECT must be set for this test!"
        out = subprocess.run(
            shlex.split(
                (
                    f"gcloud compute ssh --zone "
                    f'"{self.gcloud_zone}" '
                    f'"{self.gcloud_instance_name}" '
                    f'--project "{self.gcloud_project}" '
                    f"-- {cmd}"
                )
            )
        )
        assert (
            out.returncode == 0
        ), f"Failed gcloud command {cmd}. stdout: {out.stdout}, stderr: {out.stderr}"


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
    if managed_slurm_cluster_session.is_circleci_job:
        # CircleCI job has master running on port 8080
        conf.MASTER_PORT = "8080"
    else:
        # Local instance of devcluster is run on port 8081
        conf.MASTER_PORT = "8081"
    nodeid = request.node.nodeid
    managed_slurm_cluster_session.log_marker(f"pytest [{utils.now_ts()}] {nodeid} setup\n")
    yield managed_slurm_cluster_session
    managed_slurm_cluster_session.log_marker(f"pytest [{utils.now_ts()}] {nodeid} teardown\n")
