import json
import os
import subprocess
from typing import List

import pytest

from tests import config as conf
from tests import ray_utils

EXAMPLES_ROOT = conf.EXAMPLES_PATH / "features" / "unmanaged"


def _run_unmanaged_script(cmd: List) -> None:
    master_url = conf.make_master_url()
    env = os.environ.copy()
    env["DET_MASTER"] = master_url
    subprocess.run(cmd, env=env, check=True)


@pytest.mark.e2e_cpu
def test_unmanaged() -> None:
    exp_path = EXAMPLES_ROOT / "1_singleton.py"
    _run_unmanaged_script(["python", exp_path])


@pytest.mark.e2e_cpu
@pytest.mark.timeout(10)
def test_unmanaged_termination() -> None:
    # Ensure an erroring-out code does not hang due to a background thread.
    exp_path = EXAMPLES_ROOT / "1_error.py"
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
