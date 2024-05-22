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


def _experiment_task_id(sess: api.Session, exp_id: int) -> str:
    trials = bindings.get_GetExperimentTrials(sess, experimentId=exp_id).trials
    assert len(trials) > 0

    trial = trials[0]
    task_id = trial.taskId
    assert task_id is not None

    return task_id


def setup_exp_proxy() -> None:
    sess = api_utils.user_session()
    exp_path = pathlib.Path("/Users/hmd/projects/da/experiments/ports-proxy")
    exp_id = exp.create_experiment(
        sess,
        str(exp_path / "config.yaml"),
        str(exp_path),
        ["--config", "max_restarts=0", "--config", "resources.slots=0"],
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
                "8888",
                "--auth",
                conf.make_master_url(),
                f"{task_id}:8888",
            ],
            text=True,
        )
        print("Tunnel process started", exp_id, task_id, proc.pid)

        try:
            time.sleep(3600)
        finally:
            proc.terminate()
            proc.wait(10)
    finally:
        bindings.post_KillExperiment(sess, id=exp_id)


setup_exp_proxy()
