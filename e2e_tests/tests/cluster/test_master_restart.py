import logging
import time

import docker
import pytest
import requests

from determined.common import api
from determined.common.api import bindings
from tests import api_utils
from tests import command as cmd
from tests import config as conf
from tests import detproc
from tests import experiment as exp
from tests.cluster import abstract_cluster, managed_cluster, managed_cluster_k8s, utils
from tests.task import task

logger = logging.getLogger(__name__)


@pytest.mark.managed_devcluster
def test_master_restart_ok(restartable_managed_cluster: managed_cluster.ManagedCluster) -> None:
    _test_master_restart_ok(restartable_managed_cluster)
    restartable_managed_cluster.restart_agent(wait_for_amnesia=False)


@pytest.mark.e2e_k8s
def test_master_restart_ok_k8s(k8s_managed_cluster: managed_cluster_k8s.ManagedK8sCluster) -> None:
    _test_master_restart_ok(k8s_managed_cluster)


def _test_master_restart_ok(managed_cluster: abstract_cluster.Cluster) -> None:
    # - Kill master
    # - Restart master
    # - Schedule something.
    # Do it twice to ensure nothing gets stuck.
    sess = api_utils.user_session()
    try:
        for i in range(3):
            print("test_master_restart_ok stage %s start" % i)
            cmd_ids = [utils.run_command(sess, 1, slots) for slots in [0, 1]]

            for cmd_id in cmd_ids:
                utils.wait_for_command_state(sess, cmd_id, "TERMINATED", 300)
                utils.assert_command_succeeded(sess, cmd_id)

            managed_cluster.kill_master()
            managed_cluster.restart_master()

            print("test_master_restart_ok stage %s done" % i)
    except Exception:
        managed_cluster.restart_master()
        managed_cluster.restart_agent()
        raise


@pytest.mark.managed_devcluster
@pytest.mark.parametrize("downtime", [0, 20, 60])
def test_master_restart_reattach_recover_experiment(
    restartable_managed_cluster: managed_cluster.ManagedCluster,
    downtime: int,
) -> None:
    _test_master_restart_reattach_recover_experiment(restartable_managed_cluster, downtime)


@pytest.mark.e2e_k8s
@pytest.mark.parametrize("downtime", [0, 20, 60])
def test_master_restart_reattach_recover_experiment_k8s(
    k8s_managed_cluster: managed_cluster_k8s.ManagedK8sCluster,
    downtime: int,
) -> None:
    _test_master_restart_reattach_recover_experiment(k8s_managed_cluster, downtime)


@pytest.mark.managed_devcluster
def test_master_restart_generic_task(
    managed_cluster_restarts: managed_cluster.ManagedCluster,
) -> None:
    sess = api_utils.user_session()

    with open(conf.fixtures_path("generic_task/test_config.yaml"), "r") as config_file:
        # Create task
        config_text = config_file.read()

    req = bindings.v1CreateGenericTaskRequest(
        config=config_text,
        contextDirectory=[],
        projectId=None,
        forkedFrom=None,
        parentId=None,
        inheritContext=False,
        noPause=False,
    )
    task_resp = bindings.post_CreateGenericTask(sess, body=req)

    # Wait for task to start
    task.wait_for_task_start(sess, task_resp.taskId)
    managed_cluster_restarts.kill_master()
    managed_cluster_restarts.restart_master()
    task.wait_for_task_state(sess, task_resp.taskId, bindings.v1GenericTaskState.COMPLETED)


@pytest.mark.managed_devcluster
def test_master_restart_generic_task_pause(
    managed_cluster_restarts: managed_cluster.ManagedCluster,
) -> None:
    sess = api_utils.user_session()

    with open(conf.fixtures_path("generic_task/test_config.yaml"), "r") as config_file:
        # Create task
        config_text = config_file.read()

    req = bindings.v1CreateGenericTaskRequest(
        config=config_text,
        contextDirectory=[],
        projectId=None,
        forkedFrom=None,
        parentId=None,
        inheritContext=False,
        noPause=False,
    )
    task_resp = bindings.post_CreateGenericTask(sess, body=req)

    # Wait for task to start
    task.wait_for_task_start(sess, task_resp.taskId)
    # Pause task
    bindings.post_PauseGenericTask(sess, taskId=task_resp.taskId)
    task.wait_for_task_state(sess, task_resp.taskId, bindings.v1GenericTaskState.PAUSED)
    managed_cluster_restarts.kill_master()
    managed_cluster_restarts.restart_master()

    # Unpause task
    bindings.post_UnpauseGenericTask(sess, taskId=task_resp.taskId)
    task.wait_for_task_state(sess, task_resp.taskId, bindings.v1GenericTaskState.COMPLETED)


@pytest.mark.managed_devcluster
def _test_master_restart_reattach_recover_experiment(
    restartable_managed_cluster: abstract_cluster.Cluster, downtime: int
) -> None:
    sess = api_utils.user_session()
    try:
        exp_id = exp.create_experiment(
            sess,
            conf.fixtures_path("no_op/single-medium-train-step.yaml"),
            conf.fixtures_path("no_op"),
            None,
        )

        # TODO(ilia): don't wait for progress.
        exp.wait_for_experiment_workload_progress(sess, exp_id)

        if downtime >= 0:
            restartable_managed_cluster.kill_master()
            time.sleep(downtime)
            restartable_managed_cluster.restart_master()

        exp.wait_for_experiment_state(
            sess, exp_id, bindings.experimentv1State.COMPLETED, max_wait_secs=downtime + 60
        )
        trials = exp.experiment_trials(sess, exp_id)

        assert len(trials) == 1
        train_wls = exp.workloads_with_training(trials[0].workloads)
        assert len(train_wls) == 5
    except Exception:
        restartable_managed_cluster.restart_master()
        restartable_managed_cluster.restart_agent()
        raise


@pytest.mark.managed_devcluster
def test_master_restart_continued_experiment(
    managed_cluster_restarts: managed_cluster.ManagedCluster,
) -> None:
    sess = api_utils.user_session()
    exp_id = exp.create_experiment(
        sess,
        conf.fixtures_path("no_op/single-medium-train-step.yaml"),
        conf.fixtures_path("no_op"),
        None,
    )
    exp.wait_for_experiment_state(sess, exp_id, bindings.experimentv1State.COMPLETED)

    detproc.check_output(
        sess,
        [
            "det",
            "e",
            "continue",
            str(exp_id),
            "--config",
            "searcher.max_length.batches=505",
            "--config",
            "searcher.name=single",
        ],
    )

    managed_cluster_restarts.kill_master()
    managed_cluster_restarts.restart_master()
    exp.wait_for_experiment_state(
        sess, exp_id, bindings.experimentv1State.COMPLETED, max_wait_secs=60
    )

    # We continued the latest task, not the first one.
    experiment_trials = exp.experiment_trials(sess, exp_id)
    assert len(experiment_trials) == 1
    task_ids = experiment_trials[0].trial.taskIds
    assert task_ids is not None
    assert len(task_ids) == 2

    logs = api.task_logs(sess, task_ids[-1])
    assert "resources exited successfully with a zero exit code" in "".join(log.log for log in logs)


@pytest.mark.managed_devcluster
@pytest.mark.parametrize("wait_for_amnesia", [True, False])
def test_master_restart_error_missing_docker_container(
    managed_cluster_restarts: managed_cluster.ManagedCluster,
    wait_for_amnesia: bool,
) -> None:
    sess = api_utils.user_session()
    exp_id = exp.create_experiment(
        sess,
        conf.fixtures_path("core_api/sleep.yaml"),
        conf.fixtures_path("core_api"),
        None,
    )

    try:
        exp.wait_for_experiment_state(sess, exp_id, bindings.experimentv1State.RUNNING)

        client = docker.from_env()
        containers = client.containers.list()

        label = "ai.determined.container.description"
        containers = [c for c in containers if f"exp-{exp_id}" in c.labels.get(label, "")]
        assert len(containers) == 1

        managed_cluster_restarts.kill_agent()
        managed_cluster_restarts.kill_master()
        containers[0].kill()
        managed_cluster_restarts.restart_master()
        managed_cluster_restarts.restart_agent(wait_for_amnesia=wait_for_amnesia)

        exp.wait_for_experiment_state(sess, exp_id, bindings.experimentv1State.RUNNING)
        trials = exp.experiment_trials(sess, exp_id)
        trial_id = trials[0].trial.id

        expected_message = (
            (
                "allocation failed due to agent failure: agent failed while the "
                + "container was running: agent closed with allocated containers"
            )
            if wait_for_amnesia
            else (
                "allocation failed due to restore error: RM failed "
                + "to restore the allocation: container is gone on reattachment"
            )
        )

        for _ in range(30):
            trial_logs = "\n".join(exp.trial_logs(sess, trial_id))
            if expected_message in trial_logs:
                break
            time.sleep(1)
        assert expected_message in trial_logs
    finally:
        detproc.check_call(sess, ["det", "e", "kill", str(exp_id)])
        exp.wait_for_experiment_state(
            sess, exp_id, bindings.experimentv1State.CANCELED, max_wait_secs=20
        )


@pytest.mark.managed_devcluster
def test_master_restart_kill_works_experiment(
    restartable_managed_cluster: managed_cluster.ManagedCluster,
) -> None:
    _test_master_restart_kill_works(restartable_managed_cluster)


@pytest.mark.e2e_k8s
def test_master_restart_kill_works_k8s(
    k8s_managed_cluster: managed_cluster_k8s.ManagedK8sCluster,
) -> None:
    _test_master_restart_kill_works(k8s_managed_cluster)


def _test_master_restart_kill_works(managed_cluster_restarts: abstract_cluster.Cluster) -> None:
    sess = api_utils.user_session()
    try:
        exp_id = exp.create_experiment(
            sess,
            conf.fixtures_path("no_op/single-many-long-steps.yaml"),
            conf.fixtures_path("no_op"),
            ["--config", "searcher.max_length.batches=10000", "--config", "max_restarts=0"],
        )

        exp.wait_for_experiment_workload_progress(sess, exp_id)

        managed_cluster_restarts.kill_master()
        time.sleep(0)
        managed_cluster_restarts.restart_master()

        detproc.check_call(sess, ["det", "e", "kill", str(exp_id)])

        exp.wait_for_experiment_state(
            sess, exp_id, bindings.experimentv1State.CANCELED, max_wait_secs=30
        )

        managed_cluster_restarts.ensure_agent_ok()
    except Exception:
        managed_cluster_restarts.restart_master()
        managed_cluster_restarts.restart_agent()


@pytest.mark.managed_devcluster
@pytest.mark.parametrize("downtime, slots", [(0, 0), (20, 1), (60, 1)])
def test_master_restart_cmd(
    restartable_managed_cluster: managed_cluster.ManagedCluster, slots: int, downtime: int
) -> None:
    _test_master_restart_cmd(restartable_managed_cluster, slots, downtime)


@pytest.mark.e2e_k8s
@pytest.mark.parametrize("slots", [0, 1])
@pytest.mark.parametrize("downtime", [0, 20, 60])
def test_master_restart_cmd_k8s(
    k8s_managed_cluster: managed_cluster_k8s.ManagedK8sCluster, slots: int, downtime: int
) -> None:
    _test_master_restart_cmd(k8s_managed_cluster, slots, downtime)


def _test_master_restart_cmd(
    managed_cluster: abstract_cluster.Cluster, slots: int, downtime: int
) -> None:
    sess = api_utils.user_session()
    command = [
        "det",
        "command",
        "run",
        "-d",
        "--config",
        f"resources.slots={slots}",
        "echo weareready && sleep 30",
    ]
    command_id = detproc.check_output(sess, command).strip()

    # Don't just check ready. We want to make sure the command's sleep has started.
    logs = ""
    for log in api.task_logs(sess, command_id, follow=True):
        if "weareready" in log.log:
            break
        logs += log.log
    else:
        pytest.fail(f"did not get weareready in task logs, logs {logs}")

    if downtime >= 0:
        managed_cluster.kill_master()
        time.sleep(downtime)
        managed_cluster.restart_master()

    utils.wait_for_command_state(sess, command_id, "TERMINATED", 30)
    utils.assert_command_succeeded(sess, command_id)


@pytest.mark.managed_devcluster
@pytest.mark.parametrize("downtime", [5])
def test_master_restart_shell(
    restartable_managed_cluster: managed_cluster.ManagedCluster, downtime: int
) -> None:
    _test_master_restart_shell(restartable_managed_cluster, downtime)


@pytest.mark.e2e_k8s
@pytest.mark.parametrize("downtime", [5])
def test_master_restart_shell_k8s(
    k8s_managed_cluster: managed_cluster_k8s.ManagedK8sCluster, downtime: int
) -> None:
    _test_master_restart_shell(k8s_managed_cluster, downtime)


def _test_master_restart_shell(managed_cluster: abstract_cluster.Cluster, downtime: int) -> None:
    sess = api_utils.user_session()
    with cmd.interactive_command(sess, ["shell", "start", "--detach"]) as shell:
        assert shell.task_id
        # Checking running is not correct here, running != ready for shells.
        api.wait_for_task_ready(sess, shell.task_id)
        pre_restart_queue = detproc.check_json(sess, ["det", "job", "list", "--json"])

        if downtime >= 0:
            managed_cluster.kill_master()
            time.sleep(downtime)
            managed_cluster.restart_master()

        utils.wait_for_task_state(sess, "shell", shell.task_id, "RUNNING")
        post_restart_queue = detproc.check_json(sess, ["det", "job", "list", "--json"])
        assert pre_restart_queue == post_restart_queue

        output = detproc.check_output(
            sess, ["det", "shell", "open", shell.task_id, "det", "user", "whoami"]
        )
        assert "you are logged in as user" in output.lower(), output


def _check_web_url(sess: api.Session, path: str, name: str) -> None:
    token = sess.token
    url = sess.master.rstrip("/") + "/" + path.lstrip("/")
    bad_status_codes = []
    for _ in range(30):
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


def _check_notebook_url(sess: api.Session, path: str) -> None:
    return _check_web_url(sess, path, "JupyterLab")


def _check_tb_url(sess: api.Session, path: str) -> None:
    return _check_web_url(sess, path, "TensorBoard")


@pytest.mark.managed_devcluster
@pytest.mark.parametrize("downtime", [5])
def test_master_restart_notebook(
    restartable_managed_cluster: managed_cluster.ManagedCluster, downtime: int
) -> None:
    return _test_master_restart_notebook(restartable_managed_cluster, downtime)


@pytest.mark.e2e_k8s
@pytest.mark.parametrize("downtime", [5])
def test_master_restart_notebook_k8s(
    k8s_managed_cluster: managed_cluster_k8s.ManagedK8sCluster, downtime: int
) -> None:
    return _test_master_restart_notebook(k8s_managed_cluster, downtime)


def _test_master_restart_notebook(managed_cluster: abstract_cluster.Cluster, downtime: int) -> None:
    sess = api_utils.user_session()
    with cmd.interactive_command(sess, ["notebook", "start", "--detach"]) as notebook:
        task_id = notebook.task_id
        assert task_id is not None
        utils.wait_for_task_state(sess, "notebook", task_id, "RUNNING")
        notebook_path = f"proxy/{task_id}/"
        _check_notebook_url(sess, notebook_path)

        if downtime >= 0:
            managed_cluster.kill_master()
            time.sleep(downtime)
            managed_cluster.restart_master()

        _check_notebook_url(sess, notebook_path)

        print("notebook ok")


@pytest.mark.managed_devcluster
@pytest.mark.parametrize("downtime", [5])
def test_master_restart_tensorboard(
    restartable_managed_cluster: managed_cluster.ManagedCluster, downtime: int
) -> None:
    return _test_master_restart_tensorboard(restartable_managed_cluster, downtime)


@pytest.mark.e2e_k8s
@pytest.mark.parametrize("downtime", [5])
def test_master_restart_tensorboard_k8s(
    k8s_managed_cluster: managed_cluster_k8s.ManagedK8sCluster, downtime: int
) -> None:
    return _test_master_restart_tensorboard(k8s_managed_cluster, downtime)


def _test_master_restart_tensorboard(
    managed_cluster: abstract_cluster.Cluster, downtime: int
) -> None:
    sess = api_utils.user_session()
    exp_id = exp.create_experiment(
        sess,
        conf.fixtures_path("no_op/single-default-ckpt.yaml"),
        conf.fixtures_path("no_op"),
        None,
    )

    exp.wait_for_experiment_state(
        sess, exp_id, bindings.experimentv1State.COMPLETED, max_wait_secs=60
    )

    with cmd.interactive_command(sess, ["tensorboard", "start", "--detach", str(exp_id)]) as tb:
        task_id = tb.task_id
        assert task_id is not None
        utils.wait_for_task_state(sess, "tensorboard", task_id, "RUNNING")

        tb_path = f"proxy/{task_id}/"
        _check_tb_url(sess, tb_path)

        if downtime >= 0:
            managed_cluster.kill_master()
            time.sleep(downtime)
            managed_cluster.restart_master()

        _check_tb_url(sess, tb_path)

        print("tensorboard ok")


@pytest.mark.managed_devcluster
def test_agent_devices_change(restartable_managed_cluster: managed_cluster.ManagedCluster) -> None:
    admin = api_utils.admin_session()
    try:
        restartable_managed_cluster.kill_agent()
        restartable_managed_cluster.dc.restart_stage("agent10")

        for _i in range(5):
            agent_data = managed_cluster.get_agent_data(admin)
            if len(agent_data) == 0:
                # Agent has exploded and been wiped due to device mismatch, as expected.
                break
        else:
            pytest.fail(
                f"agent with different devices is still present after {_i} ticks: {agent_data}"
            )
    finally:
        restartable_managed_cluster.dc.kill_stage("agent10")
        restartable_managed_cluster.restart_agent()


@pytest.mark.e2e_k8s
def test_master_restart_with_queued(
    k8s_managed_cluster: managed_cluster_k8s.ManagedK8sCluster,
) -> None:
    sess = api_utils.user_session()
    agent_data = managed_cluster.get_agent_data(api_utils.admin_session())
    slots = sum([a["num_slots"] for a in agent_data])

    running_command_id = utils.run_command(sess, 120, slots)
    utils.wait_for_command_state(sess, running_command_id, "RUNNING", 30)

    queued_command_id = utils.run_command(sess, 60, slots)
    utils.wait_for_command_state(sess, queued_command_id, "QUEUED", 30)

    job_list = detproc.check_json(sess, ["det", "job", "list", "--json"])["jobs"]

    k8s_managed_cluster.kill_master()
    k8s_managed_cluster.restart_master()

    post_restart_job_list = detproc.check_json(sess, ["det", "job", "list", "--json"])["jobs"]

    assert job_list == post_restart_job_list

    for cmd_id in [running_command_id, queued_command_id]:
        utils.wait_for_command_state(sess, cmd_id, "TERMINATED", 90)
        utils.assert_command_succeeded(sess, cmd_id)
