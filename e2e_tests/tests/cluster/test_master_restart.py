import logging
import subprocess
import time
from typing import Iterator

import pytest
import requests

from determined.common import constants
from determined.common.api import authentication
from determined.common.api.bindings import determinedexperimentv1State as EXP_STATE
from tests import command as cmd
from tests import config as conf
from tests import experiment as exp
from tests.cluster.test_users import det_spawn

from .managed_cluster import ManagedCluster
from .utils import (
    command_succeeded,
    get_command_info,
    run_command,
    wait_for_command_state,
    wait_for_task_state,
)

logger = logging.getLogger(__name__)


def _sanity_check(managed_cluster: ManagedCluster) -> None:
    if not managed_cluster.reattach:
        pytest.skip()

    managed_cluster.ensure_agent_ok()


@pytest.fixture
def restartable_managed_cluster(managed_cluster: ManagedCluster) -> Iterator[ManagedCluster]:
    _sanity_check(managed_cluster)

    try:
        yield managed_cluster
        managed_cluster.wait_for_agent_ok(20)
    except Exception:
        managed_cluster.restart_master()
        managed_cluster.restart_agent()
        raise


@pytest.mark.managed_devcluster
def test_master_restart_ok(managed_cluster: ManagedCluster) -> None:
    # - Kill master
    # - Restart master
    # - Schedule something.
    # Do it twice to ensure nothing gets stuck.
    _sanity_check(managed_cluster)

    try:
        for i in range(3):
            print("test_master_restart_ok stage %s start" % i)
            cmd_ids = [run_command(1, slots) for slots in [0, 1]]

            for cmd_id in cmd_ids:
                wait_for_command_state(cmd_id, "TERMINATED", 10)
                assert command_succeeded(cmd_id)

            managed_cluster.kill_master()
            managed_cluster.restart_master()

            print("test_master_restart_ok stage %s done" % i)
    except Exception:
        managed_cluster.restart_master()
        managed_cluster.restart_agent()
        raise
    managed_cluster.restart_agent(wait_for_amnesia=False)
    _sanity_check(managed_cluster)


@pytest.mark.managed_devcluster
@pytest.mark.parametrize("downtime", [0, 20, 60])
def test_master_restart_reattach_recover_experiment(
    managed_cluster: ManagedCluster, downtime: int
) -> None:
    _sanity_check(managed_cluster)

    try:
        exp_id = exp.create_experiment(
            conf.fixtures_path("no_op/single-medium-train-step.yaml"),
            conf.fixtures_path("no_op"),
            None,
        )

        # TODO(ilia): don't wait for progress.
        exp.wait_for_experiment_workload_progress(exp_id)

        if downtime >= 0:
            managed_cluster.kill_master()
            time.sleep(downtime)
            managed_cluster.restart_master()

        exp.wait_for_experiment_state(
            exp_id, EXP_STATE.STATE_COMPLETED, max_wait_secs=downtime + 60
        )
        trials = exp.experiment_trials(exp_id)

        assert len(trials) == 1
        train_wls = exp.workloads_with_training(trials[0].workloads)
        assert len(train_wls) == 5
    except Exception:
        managed_cluster.restart_master()
        managed_cluster.restart_agent()
        raise


@pytest.mark.managed_devcluster
def test_master_restart_kill_works(managed_cluster: ManagedCluster) -> None:
    _sanity_check(managed_cluster)

    try:
        exp_id = exp.create_experiment(
            conf.fixtures_path("no_op/single-many-long-steps.yaml"),
            conf.fixtures_path("no_op"),
            ["--config", "searcher.max_length.batches=10000", "--config", "max_restarts=0"],
        )

        exp.wait_for_experiment_workload_progress(exp_id)

        managed_cluster.kill_master()
        time.sleep(0)
        managed_cluster.restart_master()

        command = ["det", "-m", conf.make_master_url(), "e", "kill", str(exp_id)]
        subprocess.check_call(command)

        exp.wait_for_experiment_state(exp_id, EXP_STATE.STATE_CANCELED, max_wait_secs=10)

        managed_cluster.ensure_agent_ok()
    except Exception:
        managed_cluster.restart_master()
        managed_cluster.restart_agent()


@pytest.mark.managed_devcluster
@pytest.mark.parametrize("slots", [0, 1])
@pytest.mark.parametrize("downtime", [0, 20, 60])
def test_master_restart_cmd(
    restartable_managed_cluster: ManagedCluster, slots: int, downtime: int
) -> None:
    managed_cluster = restartable_managed_cluster

    command_id = run_command(30, slots=slots)
    wait_for_command_state(command_id, "RUNNING", 10)

    if downtime >= 0:
        managed_cluster.kill_master()
        time.sleep(downtime)
        managed_cluster.restart_master()

    wait_for_command_state(command_id, "TERMINATED", 30)
    succeeded = "success" in get_command_info(command_id)["exitStatus"]
    assert succeeded


@pytest.mark.managed_devcluster
@pytest.mark.parametrize("downtime", [5])
def test_master_restart_shell(restartable_managed_cluster: ManagedCluster, downtime: int) -> None:
    managed_cluster = restartable_managed_cluster

    with cmd.interactive_command("shell", "start", "--detach") as shell:
        task_id = shell.task_id

        assert task_id is not None
        wait_for_task_state("shell", task_id, "RUNNING")

        if downtime >= 0:
            managed_cluster.kill_master()
            time.sleep(downtime)
            managed_cluster.restart_master()

        wait_for_task_state("shell", task_id, "RUNNING")

        child = det_spawn(["shell", "open", task_id])
        child.setecho(True)
        child.expect(r".*Permanently added.+([0-9a-f-]{36}).+known hosts\.")
        child.sendline("det user whoami")
        child.expect("You are logged in as user \\'(.*)\\'", timeout=10)
        child.sendline("exit")
        child.read()
        child.wait()
        assert child.exitstatus == 0


def _get_auth_token_for_curl() -> str:
    token = authentication.TokenStore(conf.make_master_url()).get_token(
        constants.DEFAULT_DETERMINED_USER
    )
    assert token is not None
    return token


def _check_web_url(url: str, name: str) -> None:
    token = _get_auth_token_for_curl()
    bad_status_codes = []
    for _ in range(10):
        r = requests.get(url, headers={"Authorization": f"Bearer {token}"}, allow_redirects=True)
        # Sometimes the TB/JL take a bit of time to stand up, returning 502.
        # Sometimes it takes a bit of time for master to register the proxy, returning 404.
        if r.status_code == 502 or r.status_code == 404:
            time.sleep(1)
            bad_status_codes.append(r.status_code)
            continue
        r.raise_for_status()
        html = r.content.decode("utf-8")
        assert name in html  # Brutal.
        break
    else:
        error_msg = ",".join(str(v) for v in bad_status_codes)
        pytest.fail(f"{name} {url} got error codes: {error_msg}")


def _check_notebook_url(url: str) -> None:
    return _check_web_url(url, "JupyterLab")


def _check_tb_url(url: str) -> None:
    return _check_web_url(url, "TensorBoard")


@pytest.mark.managed_devcluster
@pytest.mark.parametrize("downtime", [5])
def test_master_restart_notebook(
    restartable_managed_cluster: ManagedCluster, downtime: int
) -> None:
    managed_cluster = restartable_managed_cluster
    with cmd.interactive_command("notebook", "start", "--detach") as notebook:
        task_id = notebook.task_id
        assert task_id is not None
        wait_for_task_state("notebook", task_id, "RUNNING")
        notebook_url = f"{conf.make_master_url()}proxy/{task_id}/"
        _check_notebook_url(notebook_url)

        if downtime >= 0:
            managed_cluster.kill_master()
            time.sleep(downtime)
            managed_cluster.restart_master()

        _check_notebook_url(notebook_url)

        print("notebook ok")


@pytest.mark.managed_devcluster
@pytest.mark.parametrize("downtime", [5])
def test_master_restart_tensorboard(
    restartable_managed_cluster: ManagedCluster, downtime: int
) -> None:
    managed_cluster = restartable_managed_cluster

    exp_id = exp.create_experiment(
        conf.fixtures_path("no_op/single.yaml"),
        conf.fixtures_path("no_op"),
        None,
    )

    exp.wait_for_experiment_state(exp_id, EXP_STATE.STATE_COMPLETED, max_wait_secs=60)

    with cmd.interactive_command("tensorboard", "start", "--detach", str(exp_id)) as tb:
        task_id = tb.task_id
        assert task_id is not None
        wait_for_task_state("tensorboard", task_id, "RUNNING")

        tb_url = f"{conf.make_master_url()}proxy/{task_id}/"
        _check_tb_url(tb_url)

        if downtime >= 0:
            managed_cluster.kill_master()
            time.sleep(downtime)
            managed_cluster.restart_master()

        _check_tb_url(tb_url)

        print("tensorboard ok")
