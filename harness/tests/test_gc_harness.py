import os
from pathlib import Path
from typing import Any, Dict, List

import pytest
import simplejson

from determined import util
from determined.common import storage
from determined.exec.gc_checkpoints import delete_checkpoints
from tests.storage import util as storage_util


@pytest.fixture()
def manager(tmp_path: Path) -> storage.StorageManager:
    return storage.SharedFSStorageManager(str(tmp_path))


@pytest.fixture(params=[0, 1, 5])
def to_delete(request: Any, manager: storage.StorageManager) -> List[Dict[str, Any]]:
    metadata = []
    for _ in range(request.param):
        with manager.store_path() as (storage_id, path):
            storage_util.create_checkpoint(path)
            metadata.append(storage.StorageMetadata(storage_id, manager._list_directory(path)))

    assert len(os.listdir(manager._base_path)) == request.param
    return [simplejson.loads(util.json_encode(m)) for m in metadata]


def test_delete_checkpoints(
    manager: storage.StorageManager, to_delete: List[Dict[str, Any]]
) -> None:
    delete_checkpoints(manager, to_delete, dry_run=False)
    assert len(os.listdir(manager._base_path)) == 0


def test_dry_run(manager: storage.StorageManager, to_delete: List[Dict[str, Any]]) -> None:
    delete_checkpoints(manager, to_delete, dry_run=True)
    assert len(os.listdir(manager._base_path)) == len(to_delete)
