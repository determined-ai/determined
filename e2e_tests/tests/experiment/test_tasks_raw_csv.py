import csv
import re
from datetime import datetime, timezone
from io import StringIO

import pytest
import requests

from determined.common import api
from determined.common.api.bindings import experimentv1State
from tests import cluster as clu
from tests import command as cmd
from tests import config as conf
from tests import experiment as exp


# Create a No_Op experiment and Check training/validation times
@pytest.mark.e2e_cpu
def test_experiment_capture() -> None:
    start_time = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")

    experiment_id = exp.create_experiment(
        conf.fixtures_path("no_op/single.yaml"), conf.fixtures_path("no_op")
    )
    exp.wait_for_experiment_state(experiment_id, experimentv1State.STATE_COMPLETED)

    end_time = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    r = api.get(
        conf.make_master_url(),
        f"/resources/allocation/tasks-raw?timestamp_after={start_time}&timestamp_before={end_time}",
    )
    assert r.status_code == requests.codes.ok, r.text

    # Check if a trial entry exists for experiment that just ran
    reader = csv.DictReader(StringIO(r.text))
    matches = [row for row in reader if int(row["experiment_id"]) == experiment_id]
    assert len(matches) >= 1


@pytest.mark.e2e_cpu
def test_notebook_capture() -> None:
    start_time = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")

    task_id = None
    with cmd.interactive_command("notebook", "start") as notebook:
        task_id = notebook.task_id

        for line in notebook.stdout:
            if re.search("Jupyter Notebook .*is running at", line) is not None:
                return
    assert task_id is not None

    end_time = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    r = api.get(
        conf.make_master_url(),
        f"/resources/allocation/tasks-raw?timestamp_after={start_time}&timestamp_before={end_time}",
    )
    assert r.status_code == requests.codes.ok, r.text

    assert re.search(f"{task_id},NOTEBOOK", r.text) is not None


# Create a No_Op Experiment/Tensorboard & Confirm Tensorboard task is captured
@pytest.mark.e2e_cpu
def test_tensorboard_experiment_capture() -> None:
    start_time = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")

    experiment_id = exp.create_experiment(
        conf.fixtures_path("no_op/single.yaml"), conf.fixtures_path("no_op")
    )

    exp.wait_for_experiment_state(experiment_id, experimentv1State.STATE_COMPLETED)

    task_id = None
    with cmd.interactive_command("tensorboard", "start", "--detach", str(experiment_id)) as tb:
        task_id = tb.task_id
        for line in tb.stdout:
            if "TensorBoard is running at: http" in line:
                break
            if "TensorBoard is awaiting metrics" in line:
                raise AssertionError("Tensorboard did not find metrics")
    assert task_id is not None
    clu.utils.wait_for_task_state("tensorboard", task_id, "TERMINATED")

    end_time = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    r = api.get(
        conf.make_master_url(),
        f"/resources/allocation/tasks-raw?timestamp_after={start_time}&timestamp_before={end_time}",
    )
    assert r.status_code == requests.codes.ok, r.text

    # Confirm Experiment is captured and valid
    reader = csv.DictReader(StringIO(r.text))
    matches = [row for row in reader if int(row["experiment_id"]) == experiment_id]
    assert len(matches) >= 1

    # Confirm Tensorboard task is captured
    assert re.search(f"{task_id},TENSORBOARD", r.text) is not None
