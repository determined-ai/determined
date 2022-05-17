import dataclasses
from io import BytesIO, StringIO
from typing import Any, Dict, Tuple, Union
import os

import attrdict
from determined.util import download_gcs_blob_with_backoff
import filelock
from google.cloud import storage
from PIL import Image as PILImage
from timm.data import create_transform
import torch.nn as nn
from torch.utils.data import Dataset
import torchvision
import torchvision.transforms as T

Stat = Union[Tuple[float], Tuple[float, float, float]]


@dataclasses.dataclass
class DatasetMetadata:
    num_classes: int
    img_size: int
    in_chans: int
    mean: Stat
    std: Stat

    def to_dict(self) -> Dict[str, Union[int, Stat]]:
        return dataclasses.asdict(self)


DATASET_METADATA_BY_NAME = {
    "mnist": DatasetMetadata(
        num_classes=10, img_size=28, in_chans=1, mean=(0.1307,), std=(0.3081,)
    ),
    "cifar10": DatasetMetadata(
        num_classes=10,
        img_size=32,
        in_chans=3,
        mean=(0.4914, 0.4822, 0.4465),
        std=(0.2470, 0.2435, 0.2616),
    ),
    "imagenet": DatasetMetadata(
        num_classes=1000,
        img_size=224,
        in_chans=3,
        mean=(0.485, 0.456, 0.406),
        std=(0.229, 0.224, 0.225),
    ),
}


class GCSImageNetStreamDataset(Dataset):
    """Streams ImageNet images from Google Cloud Storage into memory. Adapted from byol example."""

    def __init__(
        self,
        data_config: attrdict.AttrDict,
        train: bool,
        transform: nn.Module,
    ) -> None:
        """
        Args:
            data_config (attrdict.AttrDict): AttrDict containing 'gcs_bucket', 'gcs_train_blob_list_path', and
                'gcs_validation_blob_list_path' keys.
            train (bool): flag for building the training (True) or validation (False) datasets.
            transform (nn.Module): transforms to be applied to the dataset.
        """
        self._transform = transform
        self._storage_client = storage.Client()
        self._bucket = self._storage_client.bucket(data_config.gcs_bucket)
        # When the dataset is first initialized, we'll loop through to catalogue the classes (subdirectories)
        # This step might take a long time.
        self._imgs_paths = []
        self._labels = []
        self._subdir_to_class: Dict[str, int] = {}
        class_count = 0
        if train:
            blob_list_path = data_config.gcs_train_blob_list_path
        else:
            blob_list_path = data_config.gcs_validation_blob_list_path
        blob_list_blob = self._bucket.blob(blob_list_path)
        blob_list_io = StringIO(
            download_gcs_blob_with_backoff(
                blob_list_blob, n_retries=4, max_backoff=2
            ).decode("utf-8")
        )
        blob_list = [s.strip() for s in blob_list_io.readlines()]
        for path in blob_list:
            self._imgs_paths.append(path)
            sub_dir = path.split("/")[-2]
            if sub_dir not in self._subdir_to_class:
                self._subdir_to_class[sub_dir] = class_count
                class_count += 1
            self._labels.append(self._subdir_to_class[sub_dir])
        dataset_str = "training" if train else "validation"
        print(f"The {dataset_str} dataset contains {len(self._imgs_paths)} records.")

    def __len__(self) -> int:
        return len(self._imgs_paths)

    def __getitem__(self, idx: int) -> Tuple[PILImage.Image, int]:
        img_path = self._imgs_paths[idx]
        blob = self._bucket.blob(img_path)
        img_str = download_gcs_blob_with_backoff(blob)
        img_bytes = BytesIO(img_str)
        img = PILImage.open(img_bytes)
        img = img.convert("RGB")
        return self._transform(img), self._labels[idx]


DATASET_DICT = {
    "mnist": torchvision.datasets.MNIST,
    "cifar10": torchvision.datasets.CIFAR10,
    "imagenet": GCSImageNetStreamDataset,
}


def get_dataset(
    data_config: attrdict.AttrDict, train: bool, transform: nn.Module
) -> Dataset:
    """
    Downloads or streams (in the case of ImageNet) the training or validation dataset, and applies `transform`
    to the corresponding images.
    """
    dataset_name = data_config.dataset_name
    dataset = DATASET_DICT[dataset_name]
    if dataset_name == "imagenet":
        # Imagenet data is streamed from GCS directly into memory.
        return dataset(data_config=data_config, train=train, transform=transform)
    else:
        download_dir = data_config.download_dir
        os.makedirs(download_dir, exist_ok=True)
        with filelock.FileLock(os.path.join(download_dir, "lock")):
            return dataset(
                root=download_dir,
                train=train,
                download=True,
                transform=transform,
            )


def build_transform(
    dataset_metadata: Any, transform_config: attrdict.AttrDict, train: bool
) -> nn.Module:
    """Generate transforms via timm's transform factory."""
    return create_transform(
        input_size=dataset_metadata.img_size,
        is_training=train,
        mean=dataset_metadata.mean,
        std=dataset_metadata.std,
        **transform_config,
    )
