import os
from pathlib import Path
from typing import Any, Dict, List

import pytest
import simplejson

from determined import util
from determined.exec.gc_checkpoints import delete_checkpoints
from determined_common import storage
from tests.storage import util as storage_util


@pytest.fixture()  # type: ignore
def config(tmp_path: Path) -> Dict[str, Any]:
    return {
        "checkpoint_storage": {
            "type": "shared_fs",
            "host_path": str(tmp_path),
            "container_path": str(tmp_path),
        }
    }


@pytest.fixture(params=[0, 1, 5])  # type: ignore
def to_delete(request: Any, config: Dict[str, Any]) -> List[Dict[str, Any]]:
    manager = storage.build(config["checkpoint_storage"])
    metadata = []
    for _ in range(request.param):
        with manager.store_path() as (storage_id, path):
            storage_util.create_checkpoint(path)
            metadata.append(storage.StorageMetadata(storage_id, manager._list_directory(path)))

    assert len(os.listdir(manager._base_path)) == request.param
    return [simplejson.loads(util.json_encode(m)) for m in metadata]


def test_delete_checkpoints(config: Dict[str, Any], to_delete: List[Dict[str, Any]]) -> None:
    delete_checkpoints(config, to_delete, validate=False, dry_run=False)

    host_path = config["checkpoint_storage"]["host_path"]
    assert len(os.listdir(host_path)) == 0


def test_dry_run(config: Dict[str, Any], to_delete: List[Dict[str, Any]]) -> None:
    delete_checkpoints(config, to_delete, validate=False, dry_run=True)

    host_path = config["checkpoint_storage"]["host_path"]
    assert len(os.listdir(host_path)) == len(to_delete)


def test_validate_success(config: Dict[str, Any], to_delete: List[Dict[str, Any]]) -> None:
    delete_checkpoints(config, to_delete, validate=True, dry_run=False)

    host_path = config["checkpoint_storage"]["host_path"]
    assert len(os.listdir(host_path)) == 0


def test_validate_failure(config: Dict[str, Any], to_delete: List[Dict[str, Any]]) -> None:
    host_path = config["checkpoint_storage"].pop("host_path")
    with pytest.raises(TypeError):
        delete_checkpoints(config, to_delete, validate=True, dry_run=False)

    assert len(os.listdir(host_path)) == len(to_delete)
