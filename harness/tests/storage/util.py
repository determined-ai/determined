import os

from determined.common import storage

EXPECTED_FILES = {
    "root.txt": "root file",
    "subdir/": None,
    "subdir/file.txt": "nested file",
}


def create_checkpoint(checkpoint_dir: str) -> None:
    """Create a new checkpoint."""
    os.makedirs(checkpoint_dir, exist_ok=False)
    for file, content in EXPECTED_FILES.items():
        if content is None:
            continue
        file = os.path.join(checkpoint_dir, file)
        os.makedirs(os.path.dirname(file), exist_ok=True)
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
