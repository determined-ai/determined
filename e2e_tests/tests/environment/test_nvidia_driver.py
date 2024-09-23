import os
import re

import pytest

from tests import api_utils
from tests import command as cmd


@pytest.mark.e2e_gpu
@pytest.mark.gpu_required
@pytest.mark.skipif(
    os.environ.get("CIRCLE_JOB") == "test-e2e-gke-single-gpu",
    reason="gke machine image does not contain nvidia-fabricmanager",
)
def test_nvidia_drivers_version_matching() -> None:
    sess = api_utils.user_session()

    with cmd.interactive_command(sess, ["shell", "start"]) as shell:
        shell.stdin.write(b"nvidia-smi\n")
        shell.stdin.write(b"nv-fabricmanager -v\n")
        # Exit the shell, so we can read output below until EOF instead of timeout
        shell.stdin.write(b"exit\n")
        shell.stdin.close()

        lines = ""
        for line in shell.stdout:
            lines += line

        m = re.search(r"Driver Version: ([\d.]+)", lines)
        if not m:
            pytest.fail(f"Did not find Nvidia driver version in shell output.\n {lines}\n")
        driver_version = m.group(1)

        m = re.search(r"Fabric Manager version is[\s:]*([\d.]+)", lines)
        if not m:
            pytest.fail(f"Did not find fabric manager version in shell output.\n {lines}\n")
        fabric_manager_version = m.group(1)

        assert (
            driver_version == fabric_manager_version
        ), f"nvidia driver {driver_version} doesn't match fabric manager {fabric_manager_version}"
