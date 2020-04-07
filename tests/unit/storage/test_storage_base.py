import os

import pytest

from determined_common import storage
from determined_common.check import CheckFailedError
from determined_common.storage import StorageManager


class NoopStorageManager(StorageManager):
    def __init__(self, base_path: str, required: str, optional: str = "default") -> None:
        super().__init__(base_path)
        self.required = required
        self.optional = optional

    @classmethod
    def identifier(cls) -> str:
        return "noop"


storage._STORAGE_MANAGERS["noop"] = NoopStorageManager


def test_getting_manager_instance() -> None:
    config = {"type": "noop", "base_path": "test", "required": "value"}
    manager = storage.build(config)
    assert isinstance(manager, NoopStorageManager)
    assert manager.required == "value"
    assert manager.optional == "default"


def test_setting_optional_variable() -> None:
    config = {"type": "noop", "base_path": "test", "required": "value", "optional": "test"}
    manager = storage.build(config)
    assert isinstance(manager, NoopStorageManager)
    assert manager.required == "value"
    assert manager.optional == "test"


def test_unexpected_params() -> None:
    config = {"type": "noop", "base_path": "test", "require": "value", "optional": "test"}
    with pytest.raises(TypeError, match="unexpected keyword argument " "'require'"):
        storage.build(config)


def test_missing_required_variable() -> None:
    config = {"type": "noop", "base_path": "test"}
    with pytest.raises(TypeError, match="missing 1 required positional " "argument: 'required'"):
        storage.build(config)


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
