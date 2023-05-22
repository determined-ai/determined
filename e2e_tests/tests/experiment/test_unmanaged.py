import os
import subprocess

import pytest

from tests import config as conf


@pytest.mark.e2e_cpu
def test_unmanaged() -> None:
    master_url = conf.make_master_url()
    exp_path = conf.EXAMPLES_PATH / "features" / "unmanaged" / "unmanaged_2_hp_search.py"
    env = os.environ.copy()
    env["DET_MASTER"] = master_url
    subprocess.run(["python", exp_path], env=env, check=True)
