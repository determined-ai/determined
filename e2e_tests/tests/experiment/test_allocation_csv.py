import csv
import datetime
import io
import re

import pytest
import requests

from determined.common.api import bindings
from tests import api_utils
from tests import command as cmd
from tests import config as conf
from tests import experiment as exp
from tests.cluster import utils as cluster_utils

API_URL = "/resources/allocation/allocations-csv?"


# Create a No_Op experiment and Check training/validation times
@pytest.mark.e2e_cpu
def test_experiment_capture() -> None:
    sess = api_utils.user_session()
    start_time = datetime.datetime.now(datetime.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")

    experiment_id = exp.create_experiment(
        sess, conf.fixtures_path("no_op/single.yaml"), conf.fixtures_path("no_op")
    )
    exp.wait_for_experiment_state(sess, experiment_id, bindings.experimentv1State.COMPLETED)

    end_time = datetime.datetime.now(datetime.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    r = sess.get(f"{API_URL}timestamp_after={start_time}&timestamp_before={end_time}")
    assert r.status_code == requests.codes.ok, r.text

    # Check if an entry exists for experiment that just ran
    reader = csv.DictReader(io.StringIO(r.text))
    matches = [row for row in reader if row["experiment_id"] == str(experiment_id)]
    assert len(matches) >= 1, f"could not find any rows for experiment {experiment_id}"


@pytest.mark.e2e_cpu
def test_notebook_capture() -> None:
    sess = api_utils.user_session()
    start_time = datetime.datetime.now(datetime.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")

    task_id = None
    with cmd.interactive_command(sess, ["notebook", "start"]) as notebook:
        task_id = notebook.task_id

        for line in notebook.stdout:
            if re.search("Jupyter Notebook .*is running at", line) is not None:
                break

    assert task_id is not None

    end_time = datetime.datetime.now(datetime.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    sess = api_utils.user_session()
    r = sess.get(f"{API_URL}timestamp_after={start_time}&timestamp_before={end_time}")
    assert r.status_code == requests.codes.ok, r.text

    assert re.search(f"{task_id}.*,NOTEBOOK", r.text) is not None

    workspace = cluster_utils.get_task_info(sess, "notebook", task_id).get("workspaceName", None)
    assert workspace is not None
    assert re.search(f"{workspace},,", r.text) is not None


# Create a No_Op Experiment/Tensorboard & Confirm Tensorboard task is captured
@pytest.mark.e2e_cpu
def test_tensorboard_experiment_capture() -> None:
    sess = api_utils.user_session()
    start_time = datetime.datetime.now(datetime.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")

    experiment_id = exp.create_experiment(
        sess, conf.fixtures_path("no_op/single.yaml"), conf.fixtures_path("no_op")
    )

    exp.wait_for_experiment_state(sess, experiment_id, bindings.experimentv1State.COMPLETED)

    with cmd.interactive_command(
        sess,
        ["tensorboard", "start", "--detach", str(experiment_id)],
    ) as tb:
        assert tb.task_id
        cluster_utils.wait_for_task_state(sess, "tensorboard", tb.task_id, "RUNNING")
    cluster_utils.wait_for_task_state(sess, "tensorboard", tb.task_id, "TERMINATED")

    # Ensure that end_time captures tensorboard
    end_time = (
        datetime.datetime.now(datetime.timezone.utc) + datetime.timedelta(minutes=1)
    ).strftime("%Y-%m-%dT%H:%M:%SZ")
    r = sess.get(f"{API_URL}timestamp_after={start_time}&timestamp_before={end_time}")
    assert r.status_code == requests.codes.ok, r.text

    # Confirm Experiment is captured and valid
    reader = csv.DictReader(io.StringIO(r.text))
    matches = [row for row in reader if row["experiment_id"] == str(experiment_id)]
    assert len(matches) >= 1

    # Confirm Tensorboard task is captured
    assert re.search(f"{tb.task_id}.*,TENSORBOARD", r.text) is not None

    workspace = cluster_utils.get_task_info(sess, "tensorboard", tb.task_id).get(
        "workspaceName", None
    )
    assert workspace is not None
    assert re.search(f"{workspace},,", r.text) is not None


# Create a command and confirm that the task is captured.
@pytest.mark.e2e_cpu
def test_cmd_capture() -> None:
    sess = api_utils.user_session()
    start_time = datetime.datetime.now(datetime.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")

    task_id = None
    with cmd.interactive_command(sess, ["cmd", "run", "sleep 10s"]) as sleep_cmd:
        task_id = sleep_cmd.task_id

        for line in sleep_cmd.stdout:
            if re.search("Resources for Command .*have started", line) is not None:
                break

    assert task_id is not None

    end_time = datetime.datetime.now(datetime.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    sess = api_utils.user_session()
    r = sess.get(f"{API_URL}timestamp_after={start_time}&timestamp_before={end_time}")
    assert r.status_code == requests.codes.ok, r.text

    assert re.search(f"{task_id}.*,COMMAND", r.text) is not None

    workspace = cluster_utils.get_task_info(sess, "command", task_id).get("workspaceName", None)
    assert workspace is not None
    assert re.search(f"{workspace},,", r.text) is not None
