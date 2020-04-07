import os

from determined_common.storage import Storable


class StorableFixture(Storable):
    def __init__(self) -> None:
        self.expected_files = {
            "root.txt": "root file",
            "subdir/": None,
            "subdir/file.txt": "nested file",
        }

    def save(self, checkpoint_dir: str) -> None:
        assert not os.path.exists(checkpoint_dir)
        os.makedirs(checkpoint_dir, exist_ok=False)
        for file, content in self.expected_files.items():
            if content is None:
                continue
            file = os.path.join(checkpoint_dir, file)
            os.makedirs(os.path.dirname(file), exist_ok=True)
            with open(file, "w") as fp:
                fp.write(content)

    def load(self, checkpoint_dir: str) -> None:
        assert os.path.exists(checkpoint_dir)
        for file, content in self.expected_files.items():
            if content is None:
                continue
            file = os.path.join(checkpoint_dir, file)
            with open(file, "r") as fp:
                assert fp.read() == content
