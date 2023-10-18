import pathlib
from typing import Any

import filelock
from torchvision import datasets, transforms


def get_dataset(data_dir: str, train: bool) -> Any:
    data_path = pathlib.Path(data_dir)
    data_path.mkdir(parents=True, exist_ok=True)

    # Use a file lock so that only one worker on each node downloads.
    with filelock.FileLock(data_path / "lock"):
        return datasets.MNIST(
            root=str(data_dir),
            train=train,
            transform=transforms.Compose(
                [
                    transforms.ToTensor(),
                    # These are the precomputed mean and standard deviation of the
                    # MNIST data; this normalizes the data to have zero mean and unit
                    # standard deviation.
                    transforms.Normalize((0.1307,), (0.3081,)),
                ]
            ),
            download=True,
        )
