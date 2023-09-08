import csv
import io
import pathlib
import re
import subprocess
import time

import pytest
import requests

from determined.common import api
from determined.common.api import bindings
from tests import api_utils
from tests import config as conf
from tests import detproc
from tests import experiment as exp
from tests import ray_utils


def _experiment_task_id(sess: api.Session, exp_id: int) -> str:
    trials = bindings.get_GetExperimentTrials(sess, experimentId=exp_id).trials
    assert len(trials) > 0

    trial = trials[0]
    task_id = trial.taskId
    assert task_id is not None

    return task_id


def _probe_tunnel(proc: "subprocess.Popen[str]", port: int = 8265) -> None:
    max_tunnel_time = 300
    start = time.time()
    ctr = 0
    while time.time() - start < max_tunnel_time:
        try:
            r = requests.get(f"http://localhost:{port}", timeout=5)
            if r.status_code == 200:
                break
        except requests.exceptions.ConnectionError:
            pass
        except requests.exceptions.ReadTimeout:
            pass
        if ctr + 1 % 10 == 0:
            print(f"Tunnel probe pending: {ctr} ticks...")
        time.sleep(1)
        if proc.poll() is not None:
            pytest.fail(f"Tunnel process has exited prematurely, return code: {proc.returncode}")
        ctr += 1
    else:
        pytest.fail(f"Failed to probe the tunnel after {max_tunnel_time} seconds")

    print(f"Tunnel probe done after {ctr} ticks.")


def _ray_job_submit(exp_path: pathlib.Path, port: int = 8265) -> None:
    return ray_utils.ray_job_submit(exp_path, ["python", "ray_job.py"], port=port)


@pytest.mark.e2e_cpu
@pytest.mark.timeout(600)
def test_experiment_proxy_ray_tunnel() -> None:
    sess = api_utils.user_session()
    exp_path = conf.EXAMPLES_PATH / "features" / "ports"
    exp_id = exp.create_experiment(
        sess,
        str(exp_path / "ray_launcher.yaml"),
        str(exp_path),
        ["--config", "max_restarts=0", "--config", "resources.slots=1"],
    )
    try:
        exp.wait_for_experiment_state(sess, exp_id, bindings.experimentv1State.RUNNING)
        task_id = _experiment_task_id(sess, exp_id)

        proc = detproc.Popen(
            sess,
            [
                "python",
                "-m",
                "determined.cli.tunnel",
                "--listener",
                "8265",
                "--auth",
                conf.make_master_url(),
                f"{task_id}:8265",
            ],
            text=True,
        )

        try:
            _probe_tunnel(proc)
            _ray_job_submit(exp_path)
        finally:
            proc.terminate()
            proc.wait(10)
    finally:
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
    sess = api_utils.user_session()
    proc = detproc.run(
        sess,
        [
            "det",
            "experiment",
            "list",
            "--csv",
        ],
        capture_output=True,
        text=True,
        check=True,
    )
    reader = csv.DictReader(io.StringIO(proc.stdout))
    for row in reader:
        if row["name"] == "ray_launcher":
            if row["state"] not in ["CANCELED", "COMPLETED"]:
                exp_id = int(row["ID"])
                bindings.post_KillExperiment(sess, id=exp_id)


@pytest.mark.e2e_cpu
@pytest.mark.timeout(600)
def test_experiment_proxy_ray_publish() -> None:
    sess = api_utils.user_session()
    exp_path = conf.EXAMPLES_PATH / "features" / "ports"
    proc = detproc.Popen(
        sess,
        [
            "det",
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
            exp.wait_for_experiment_state(sess, exp_id, bindings.experimentv1State.RUNNING)
            _probe_tunnel(proc)
            _ray_job_submit(exp_path)
        finally:
            bindings.post_KillExperiment(sess, id=exp_id)
    finally:
        proc.terminate()
        proc.wait(10)
