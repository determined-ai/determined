import os

import pytest

@pytest.mark.e2e_gpu
def test_nvidia_driver_versions() -> None:
    driver_version = os.popen("nvidia-smi | grep -Po 'Driver Version: \K[0-9]*\.[0-9]*\.[0-9]*'").read()
    fabric_manager_version = os.popen("nv-fabricmanager -v | grep -Po '[0-9]*\.[0-9]*\.[0-9]*'").read()

    assert driver_version == fabric_manager_version, f"nvidia driver {driver_version} doesn't match fabric manager {fabric_manager_version}"