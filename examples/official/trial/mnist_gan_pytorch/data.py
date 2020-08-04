import logging
import os
import shutil
import urllib.parse
from typing import Any, Dict

import requests

from torchvision import datasets, transforms


def get_dataset(data_dir: str, train: bool) -> Any:
    return datasets.MNIST(
        data_dir,
        train=train,
        transform=transforms.Compose(
            [
                transforms.ToTensor(),
                transforms.Normalize((0.5,), (0.5,)),
            ]
        ),
    )


def download_dataset(download_directory: str, data_config: Dict[str, Any]) -> str:
    url = data_config["url"]
    url_path = urllib.parse.urlparse(url).path
    basename = url_path.rsplit("/", 1)[1]

    download_directory = os.path.join(download_directory, "MNIST")
    os.makedirs(download_directory, exist_ok=True)
    filepath = os.path.join(download_directory, basename)
    if not os.path.exists(filepath):
        logging.info("Downloading {} to {}".format(url, filepath))

        r = requests.get(url, stream=True)
        with open(filepath, "wb") as f:
            for chunk in r.iter_content(chunk_size=8192):
                if chunk:
                    f.write(chunk)

    shutil.unpack_archive(filepath, download_directory)

    return os.path.dirname(download_directory)
