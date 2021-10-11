import os

import pytest

from determined.common import storage
from determined.common.check import CheckFailedError
from determined.common.storage import StorageManager


def test_unknown_type() -> None:
    config = {"type": "unknown"}
    with pytest.raises(TypeError, match="Unknown storage type: unknown"):
        storage.build(config)


def test_missing_type() -> None:
    with pytest.raises(CheckFailedError, match="Missing 'type' parameter"):
        storage.build({})


def test_illegal_type() -> None:
    config = {"type": 4}
    with pytest.raises(CheckFailedError, match="must be a string"):
        storage.build(config)


def test_list_directory() -> None:
    root = os.path.join(os.path.dirname(__file__), "fixtures")

    assert set(StorageManager._list_directory(root)) == {
        "root.txt",
        "nested/",
        "nested/nested.txt",
        "nested/another.txt",
    }


def test_list_directory_on_file() -> None:
    root = os.path.join(os.path.dirname(__file__), "fixtures", "root.txt")
    assert os.path.exists(root)
    with pytest.raises(CheckFailedError, match="must be an extant directory"):
        StorageManager._list_directory(root)


def test_list_nonexistent_directory() -> None:
    root = "./non-existent-directory"
    assert not os.path.exists(root)
    with pytest.raises(CheckFailedError, match="must be an extant directory"):
        StorageManager._list_directory(root)
