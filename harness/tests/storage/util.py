import os
from typing import Callable, Optional

import pytest

from determined import errors
from determined.common import storage

EXPECTED_FILES = {
    "root.txt": "root file",
    "subdir/": None,
    "subdir/file.txt": "nested file",
    "empty_dir/": None,
}


def create_checkpoint(checkpoint_dir: str) -> None:
    """Create a new checkpoint."""
    os.makedirs(checkpoint_dir, exist_ok=False)
    for file, content in EXPECTED_FILES.items():
        file = os.path.join(checkpoint_dir, file)
        os.makedirs(os.path.dirname(file), exist_ok=True)
        if content is None:
            continue
        with open(file, "w") as fp:
            fp.write(content)


def validate_checkpoint(checkpoint_dir: str) -> None:
    """Make sure an existing checkpoint looks correct."""
    assert os.path.exists(checkpoint_dir)
    files_found = set(storage.StorageManager._list_directory(checkpoint_dir))
    assert files_found == set(EXPECTED_FILES.keys())
    for found in files_found:
        path = os.path.join(checkpoint_dir, found)
        if EXPECTED_FILES[found] is None:
            assert os.path.isdir(path)
        else:
            assert os.path.isfile(path)
            with open(path) as f:
                assert f.read() == EXPECTED_FILES[found]


def run_storage_lifecycle_test(
    manager: storage.StorageManager,
    post_delete_cb: Optional[Callable] = None,
) -> None:
    checkpoints = []
    for _ in range(5):
        with manager.store_path() as (storage_id, path):
            create_checkpoint(path)
            checkpoints.append(storage_id)

    for storage_id in checkpoints:
        # Load checkpoint.
        with manager.restore_path(storage_id) as path:
            validate_checkpoint(path)
        # Delete.
        manager.delete(storage_id)
        # Ensure it is gone.
        with pytest.raises(errors.CheckpointNotFound):
            with manager.restore_path(storage_id) as path:
                pass
        # Allow for backend-specific inspection.
        if post_delete_cb is not None:
            post_delete_cb(storage_id)
