import json
import os
import subprocess
import uuid
from typing import List, Optional

import pytest

from determined.common.api import bindings
from tests import api_utils
from tests import config as conf
from tests import ray_utils

EXAMPLES_ROOT = conf.EXAMPLES_PATH / "features" / "unmanaged"


def _run_unmanaged_script(cmd: List, env_to_add: Optional[dict] = None) -> None:
    master_url = conf.make_master_url()
    env = os.environ.copy()
    env["DET_MASTER"] = master_url

    if env_to_add is not None:
        env.update(env_to_add)

    subprocess.run(cmd, env=env, check=True, text=True)


@pytest.mark.e2e_cpu
def test_unmanaged() -> None:
    exp_path = EXAMPLES_ROOT / "1_singleton.py"
    _run_unmanaged_script(["python", exp_path])


@pytest.mark.e2e_cpu
def test_unmanaged_checkpoints() -> None:
    external_id = str(uuid.uuid4())

    exp_path = conf.fixtures_path("unmanaged/checkpointing.py")
    _run_unmanaged_script(["python", exp_path], {"DET_TEST_EXTERNAL_EXP_ID": external_id})

    sess = api_utils.determined_test_session()
    exps = bindings.get_GetExperiments(sess, limit=-1).experiments
    exps = [exp for exp in exps if exp.externalExperimentId == external_id]
    assert len(exps) == 1
    exp_id = exps[0].id

    checkpoints = bindings.get_GetExperimentCheckpoints(session=sess, id=exp_id).checkpoints
    assert len(checkpoints) > 0
    assert all(checkpoint.storageId is not None for checkpoint in checkpoints)


@pytest.mark.e2e_cpu
@pytest.mark.timeout(10)
def test_unmanaged_termination() -> None:
    # Ensure an erroring-out code does not hang due to a background thread.
    exp_path = conf.fixtures_path("unmanaged/error_termination.py")
    with pytest.raises(subprocess.CalledProcessError):
        _run_unmanaged_script(["python", exp_path])


@pytest.mark.e2e_cpu
def test_unmanaged_ray_hp_search() -> None:
    master_url = conf.make_master_url()
    exp_path = EXAMPLES_ROOT / "ray"
    runtime_env = {
        "env_vars": {
            "DET_MASTER": master_url,
        }
    }

    try:
        subprocess.run(["ray", "start", "--head", "--disable-usage-stats"], check=True)
        ray_utils.ray_job_submit(
            exp_path,
            ["python", "ray_hp_search.py"],
            submit_args=[
                "--runtime-env-json",
                json.dumps(runtime_env),
            ],
        )
    finally:
        subprocess.run(["ray", "stop"], check=True)
