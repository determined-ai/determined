import importlib
import sys
import warnings
from typing import Generator

import pytest


@pytest.fixture
def clear_modules() -> Generator[None, None, None]:
    _clear_modules()
    yield


def _clear_modules() -> None:
    all_module_names = list(sys.modules.keys())
    for module_name in all_module_names:
        if module_name.startswith("determined.common"):
            del sys.modules[module_name]


# TODO (Taylor): Remove this when we actually remove `determined.common.experimental``
@pytest.mark.parametrize(
    "old_module_name",
    [
        "determined.common.experimental.checkpoint",
        "determined.common.experimental.determined",
        "determined.common.experimental.experiment",
        "determined.common.experimental.trial",
        "determined.common.experimental.model",
    ],
)
def test_warnings_happen(old_module_name: str, clear_modules: Generator[None, None, None]) -> None:
    with warnings.catch_warnings(record=True) as w:
        importlib.import_module(old_module_name)
        assert len(w) == 1
        assert issubclass(w[-1].category, FutureWarning)
        assert "deprecated" in str(w[-1].message)


@pytest.mark.parametrize(
    "new_module_name,class_name",
    [
        (
            "determined.experimental",
            "Checkpoint",
        ),
        (
            "determined.experimental",
            "Determined",
        ),
        (
            "determined.experimental",
            "ExperimentReference",
        ),
        (
            "determined.experimental",
            "TrialReference",
        ),
        (
            "determined.experimental",
            "TrialSortBy",
        ),
        (
            "determined.experimental",
            "TrialOrderBy",
        ),
        (
            "determined.experimental",
            "Model",
        ),
        (
            "determined.experimental",
            "ModelOrderBy",
        ),
        (
            "determined.experimental",
            "ModelSortBy",
        ),
    ],
)
def test_warnings_dont_happen(
    new_module_name: str, class_name: str, clear_modules: Generator[None, None, None]
) -> None:
    with warnings.catch_warnings(record=True) as w:
        module = importlib.import_module(new_module_name)
        assert len(w) == 0
        assert getattr(module, class_name) is not None
