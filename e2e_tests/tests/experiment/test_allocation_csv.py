import csv
import datetime
import io
import re
import sys
import uuid
from typing import Optional

import pytest
import requests

from determined.common.api import bindings
from determined.experimental import client
from tests import api_utils
from tests import command as cmd
from tests.cluster import utils as cluster_utils
from tests.experiment import noop

API_URL = "/resources/allocation/allocations-csv?"


def timestamp_with_offset(delta: int) -> str:
    dt = datetime.datetime.now(datetime.timezone.utc) + datetime.timedelta(seconds=delta)
    return dt.strftime("%Y-%m-%dT%H:%M:%SZ")


def validate_trial_csv_rows(
    raw_text: str, experiment_id: int, workspace_name: Optional[str]
) -> None:
    rows = list(csv.DictReader(io.StringIO(raw_text)))
    experiment_matches = list(filter(lambda row: row["experiment_id"] == str(experiment_id), rows))
    assert len(experiment_matches) >= 1, f"could not find any rows for experiment {experiment_id}"

    if workspace_name is None:
        return
    workspace_matches = list(filter(lambda row: row["workspace_name"] == str(workspace_name), rows))
    assert len(workspace_matches) >= 1, f"could not find any rows for workspace {workspace_name}"


# Create a noop experiment and Check training/validation times
@pytest.mark.e2e_cpu
def test_experiment_capture() -> None:
    sess = api_utils.admin_session()
    w1 = bindings.post_PostWorkspace(
        sess,
        body=bindings.v1PostWorkspaceRequest(
            name=f"workspace-{uuid.uuid4().hex[:8]}",
            agentUserGroup=bindings.v1AgentUserGroup(agentGid=1000, agentGroup="det"),
        ),
    ).workspace
    p1 = bindings.post_PostProject(
        sess,
        body=bindings.v1PostProjectRequest(
            name=f"project-{uuid.uuid4().hex[:8]}",
            workspaceId=w1.id,
        ),
        workspaceId=w1.id,
    ).project

    # Avoid any rounding or inclusion errors.
    start_time = timestamp_with_offset(-2)

    exp_ref = noop.create_experiment(sess, project_id=p1.id)
    assert exp_ref.wait(interval=0.01) == client.ExperimentState.COMPLETED

    # Avoid any rounding or inclusion errors.
    end_time = timestamp_with_offset(+2)
    # Check if an entry exists for experiment that just ran
    r = sess.get(f"{API_URL}timestamp_after={start_time}&timestamp_before={end_time}")
    assert r.status_code == requests.codes.ok, r.text
    validate_trial_csv_rows(r.text, exp_ref.id, w1.name)

    # Move project to new workspace,
    # and confirm that allocation csv still persists the old workspace id
    w2 = bindings.post_PostWorkspace(
        sess,
        body=bindings.v1PostWorkspaceRequest(
            name=f"workspace-{uuid.uuid4().hex[:8]}",
            agentUserGroup=bindings.v1AgentUserGroup(agentGid=1000, agentGroup="det"),
        ),
    ).workspace
    bindings.post_MoveProject(
        sess,
        projectId=p1.id,
        body=bindings.v1MoveProjectRequest(
            destinationWorkspaceId=w2.id,
            projectId=p1.id,
        ),
    )

    r = sess.get(f"{API_URL}timestamp_after={start_time}&timestamp_before={end_time}")
    assert r.status_code == requests.codes.ok, r.text
    validate_trial_csv_rows(r.text, exp_ref.id, w1.name)

    # Delete the experiment, and confirm that allocation csv still persists the experiment id
    bindings.delete_DeleteExperiment(session=sess, experimentId=exp_ref.id)

    r = sess.get(f"{API_URL}timestamp_after={start_time}&timestamp_before={end_time}")
    assert r.status_code == requests.codes.ok, r.text
    validate_trial_csv_rows(r.text, exp_ref.id, w1.name)

    # Clean up test workspaces
    bindings.delete_DeleteWorkspace(session=sess, id=w1.id)
    bindings.delete_DeleteWorkspace(session=sess, id=w2.id)


@pytest.mark.e2e_cpu
def test_notebook_capture() -> None:
    sess = api_utils.admin_session()
    # Avoid any rounding or inclusion errors.
    start_time = timestamp_with_offset(-2)

    task_id = None
    with cmd.interactive_command(sess, ["notebook", "start"]) as notebook:
        task_id = notebook.task_id

        for line in notebook.stdout:
            if re.search("Jupyter Notebook .*is running at", line) is not None:
                break

    assert task_id is not None

    # Avoid any rounding or inclusion errors.
    end_time = timestamp_with_offset(+2)
    end_time = datetime.datetime.now(datetime.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    sess = api_utils.admin_session()
    url = f"{API_URL}timestamp_after={start_time}&timestamp_before={end_time}"
    r = sess.get(url)
    assert r.status_code == requests.codes.ok, r.text

    if re.search(f"{task_id}.*,NOTEBOOK", r.text) is None:
        msg = f"did not find task_id={task_id} @ {url} in output:\n{r.text}"
        print(msg, file=sys.stderr)
        raise ValueError(msg)

    workspace = cluster_utils.get_task_info(sess, "notebook", task_id).get("workspaceName", None)
    assert workspace is not None
    assert re.search(f"{workspace},,", r.text) is not None


# Create a noop Experiment/Tensorboard & Confirm Tensorboard task is captured
@pytest.mark.e2e_cpu
def test_tensorboard_experiment_capture() -> None:
    sess = api_utils.admin_session()
    # Avoid any rounding or inclusion errors.
    start_time = timestamp_with_offset(-2)

    exp_ref = noop.create_experiment(sess, [noop.Report({"x": 1})])
    assert exp_ref.wait(interval=0.01) == client.ExperimentState.COMPLETED

    with cmd.interactive_command(
        sess,
        ["tensorboard", "start", "--detach", str(exp_ref.id)],
    ) as tb:
        assert tb.task_id
        cluster_utils.wait_for_task_state(sess, "tensorboard", tb.task_id, "RUNNING")
    cluster_utils.wait_for_task_state(sess, "tensorboard", tb.task_id, "TERMINATED")

    # Avoid any rounding or inclusion errors.
    end_time = timestamp_with_offset(+2)
    url = f"{API_URL}timestamp_after={start_time}&timestamp_before={end_time}"
    r = sess.get(url)
    assert r.status_code == requests.codes.ok, r.text

    # Confirm Experiment is captured and valid
    reader = csv.DictReader(io.StringIO(r.text))
    matches = list(filter(lambda row: row["experiment_id"] == str(exp_ref.id), reader))
    assert len(matches) >= 1

    # Confirm Tensorboard task is captured
    if re.search(f"{tb.task_id}.*,TENSORBOARD", r.text) is None:
        msg = f"did not find task_id={tb.task_id} @ {url} in output:\n{r.text}"
        print(msg, file=sys.stderr)
        raise ValueError(msg)

    workspace = cluster_utils.get_task_info(sess, "tensorboard", tb.task_id).get(
        "workspaceName", None
    )
    assert workspace is not None
    assert re.search(f"{workspace},,", r.text) is not None


# Create a command and confirm that the task is captured.
@pytest.mark.e2e_cpu
def test_cmd_capture() -> None:
    sess = api_utils.admin_session()
    # Avoid any rounding or inclusion errors.
    start_time = timestamp_with_offset(-2)

    task_id = None
    with cmd.interactive_command(sess, ["cmd", "run", "sleep 10s"]) as sleep_cmd:
        task_id = sleep_cmd.task_id

        for line in sleep_cmd.stdout:
            if re.search("Resources for Command .*have started", line) is not None:
                break

    assert task_id is not None

    # Avoid any rounding or inclusion errors.
    end_time = timestamp_with_offset(+2)

    sess = api_utils.admin_session()
    url = f"{API_URL}timestamp_after={start_time}&timestamp_before={end_time}"
    r = sess.get(url)
    assert r.status_code == requests.codes.ok, r.text

    if re.search(f"{task_id}.*,COMMAND", r.text) is None:
        msg = f"did not find task_id={task_id} @ {url} in output:\n{r.text}"
        print(msg, file=sys.stderr)
        raise ValueError(msg)

    workspace = cluster_utils.get_task_info(sess, "command", task_id).get("workspaceName", None)
    assert workspace is not None
    assert re.search(f"{workspace},,", r.text) is not None
