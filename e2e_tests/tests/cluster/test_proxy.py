import csv
import os
import pathlib
import re
import subprocess
import time
from io import StringIO

import pytest
import requests

from determined.common.api import bindings
from tests import api_utils
from tests import config as conf
from tests import experiment as exp


def _experiment_task_id(exp_id: int) -> str:
    sess = api_utils.determined_test_session()
    trials = bindings.get_GetExperimentTrials(sess, experimentId=exp_id).trials
    assert len(trials) > 0

    trial = trials[0]
    task_id = trial.taskId
    assert task_id is not None

    return task_id


def _probe_tunnel(proc: "subprocess.Popen[str]") -> None:
    max_tunnel_ticks = 300
    for i in range(max_tunnel_ticks):
        try:
            r = requests.get("http://localhost:8265", timeout=5)
            if r.status_code == 200:
                break
        except requests.exceptions.ConnectionError:
            pass
        except requests.exceptions.ReadTimeout:
            pass
        if i + 1 % 10 == 0:
            print(f"Tunnel probe pending: {i} ticks...")
        time.sleep(1)
        if proc.poll() is not None:
            pytest.fail(f"Tunnel process has exited prematurely, return code: {proc.returncode}")
    else:
        pytest.fail(f"Failed to probe the tunnel in {max_tunnel_ticks} ticks")

    print(f"Tunnel probe done after {i} ticks.")


def _ray_job_submit(exp_path: pathlib.Path) -> None:
    env = os.environ.copy()
    env["RAY_ADDRESS"] = "http://localhost:8265"
    subprocess.run(
        [
            "ray",
            "job",
            "submit",
            "--working-dir",
            str(exp_path),
            "--",
            "python",
            "ray_job.py",
        ],
        check=True,
        env=env,
    )


@pytest.mark.e2e_cpu
def test_experiment_proxy_ray_tunnel() -> None:
    exp_path = conf.EXAMPLES_PATH / "features" / "ports"
    exp_id = exp.create_experiment(
        str(exp_path / "ray_launcher.yaml"),
        str(exp_path),
        ["--config", "max_restarts=0", "--config", "resources.slots=1"],
    )
    try:
        exp.wait_for_experiment_state(exp_id, bindings.experimentv1State.STATE_RUNNING)
        task_id = _experiment_task_id(exp_id)

        proc = subprocess.Popen(
            [
                "python",
                "-m",
                "determined.cli.tunnel",
                "--listener",
                "8265",
                conf.make_master_url(),
                f"{task_id}:8265",
            ],
            text=True,
        )

        try:
            _probe_tunnel(proc)
            _ray_job_submit(exp_path)
        finally:
            proc.kill()
    finally:
        sess = api_utils.determined_test_session()
        bindings.post_KillExperiment(sess, id=exp_id)


def _parse_exp_id(proc: "subprocess.Popen[str]") -> int:
    assert proc.stdout is not None
    for line in iter(proc.stdout.readline, ""):
        if proc.poll() is not None:
            pytest.fail(
                f"Unexpected `det e create` failure before receiving an experiment id, "
                f"return code: f{proc.returncode}"
            )
        m = re.search(r"Created experiment (\d+)\n", line)
        if m is not None:
            return int(m.group(1))
    pytest.fail("Failed to find experiment id in `det e create` output")


def _kill_all_ray_experiments() -> None:
    proc = subprocess.run(
        [
            "det",
            "-m",
            conf.make_master_url(),
            "experiment",
            "list",
            "--csv",
        ],
        capture_output=True,
        text=True,
        check=True,
    )
    reader = csv.DictReader(StringIO(proc.stdout))
    sess = api_utils.determined_test_session()
    for row in reader:
        if row["name"] == "ray_launcher":
            if row["state"] not in ["CANCELED", "COMPLETED"]:
                exp_id = int(row["ID"])
                bindings.post_KillExperiment(sess, id=exp_id)


@pytest.mark.e2e_cpu
def test_experiment_proxy_ray_publish() -> None:
    exp_path = conf.EXAMPLES_PATH / "features" / "ports"
    proc = subprocess.Popen(
        [
            "det",
            "-m",
            conf.make_master_url(),
            "experiment",
            "create",
            str(exp_path / "ray_launcher.yaml"),
            str(exp_path),
            "--config",
            "max_restarts=0",
            "--config",
            "resources.slots=1",
            "-f",
            "-p",
            "8265",
        ],
        stdout=subprocess.PIPE,
        text=True,
    )

    try:
        try:
            exp_id = _parse_exp_id(proc)
        except Exception:
            _kill_all_ray_experiments()
            raise

        try:
            exp.wait_for_experiment_state(exp_id, bindings.experimentv1State.STATE_RUNNING)
            _probe_tunnel(proc)
            _ray_job_submit(exp_path)
        finally:
            sess = api_utils.determined_test_session()
            bindings.post_KillExperiment(sess, id=exp_id)
    finally:
        proc.kill()
