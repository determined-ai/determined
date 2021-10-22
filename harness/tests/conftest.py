from typing import Any, Iterator, List

import pytest
from _pytest.fixtures import SubRequest


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
