import os
from pathlib import Path

from determined.estimator import _scan_checkpoint_directory
from tests.filetree import FileTree


def test_checkpoint_matches_directory_contents(tmp_path: Path) -> None:
    with FileTree(
        tmp_path,
        {
            "checkpoint": """
model_checkpoint_path: "model.ckpt-9"
all_model_checkpoint_paths: "model.ckpt-1"
all_model_checkpoint_paths: "model.ckpt-nonexistent"
all_model_checkpoint_paths: "model.ckpt-9"
""",
            "model.ckpt-1.data-0-of-1": "",
            "model.ckpt-orphan.data-0-of-1": "",
            "model.ckpt-9.data-0-of-1": "",
        },
    ) as tree:
        checkpoints = _scan_checkpoint_directory(str(tree))
        assert len(checkpoints) == 1
        assert [os.path.basename(v) for v in checkpoints[0].state.all_model_checkpoint_paths] == [
            "model.ckpt-orphan",
            "model.ckpt-1",
            "model.ckpt-9",
        ]
