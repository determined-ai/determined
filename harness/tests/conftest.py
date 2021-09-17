import os
from typing import Any, Iterator, List

import pytest
from _pytest.fixtures import SubRequest

from determined import gpu


def pytest_addoption(parser: Any) -> None:
    parser.addoption("--runslow", action="store_true", default=False, help="run slow tests")
    parser.addoption(
        "--require-secrets", action="store_true", help="fail tests when storage access fails"
    )


@pytest.fixture
def require_secrets(request: SubRequest) -> Iterator[bool]:
    yield bool(request.config.getoption("--require-secrets"))


def pytest_collection_modifyitems(config: Any, items: List[Any]) -> None:
    if config.getoption("--runslow"):
        # --runslow given in cli: do not skip slow tests
        return
    skip_slow = pytest.mark.skip(reason="need --runslow option to run")
    for item in items:
        if "slow" in item.keywords:
            item.add_marker(skip_slow)


@pytest.fixture(scope="function")
def expose_gpus() -> Iterator[None]:
    """
    Set the environment variables to mimic what the agent uses to control which
    GPUs will be used in the harness code. Using this fixture will enforce that
    GPU-capable unit tests use the GPU properly when a GPU is available.
    """
    old_use_gpu = os.environ.get("DET_USE_GPU")
    old_visible_devs = os.environ.get("NVIDIA_VISIBLE_DEVICES")

    _, gpu_uuids = gpu.get_gpu_ids_and_uuids()
    if not gpu_uuids:
        os.environ["DET_USE_GPU"] = "0"
        os.environ["NVIDIA_VISIBLE_DEVICES"] = ""
        yield
    else:
        os.environ["DET_USE_GPU"] = "1"
        os.environ["NVIDIA_VISIBLE_DEVICES"] = ",".join(gpu_uuids)
        yield

    # Restore original environment.
    if old_use_gpu is None:
        del os.environ["DET_USE_GPU"]
    else:
        os.environ["DET_USE_GPU"] = old_use_gpu

    if old_visible_devs is None:
        del os.environ["NVIDIA_VISIBLE_DEVICES"]
    else:
        os.environ["NVIDIA_VISIBLE_DEVICES"] = old_visible_devs
