import os
import random
from dataclasses import dataclass
from enum import Enum, auto
from io import BytesIO, StringIO
from typing import Callable, Dict, List, Tuple

import torch
import torch.nn as nn
import torchvision.transforms as T
from attrdict import AttrDict
from filelock import FileLock
from google.cloud import storage
from PIL import Image as PILImage
from torch.utils.data import Dataset
from torchvision.datasets import CIFAR10, STL10

from determined.util import download_gcs_blob_with_backoff


def load_image(path: str) -> PILImage.Image:
    # Helper function from https://pytorch.org/docs/stable/_modules/torchvision/datasets/folder.html#ImageFolder
    with open(path, "rb") as f:
        img = PILImage.open(f)
        return img.convert("RGB")


class JointDataset(Dataset):
    """
    Combines multiple image datasets yielding (img, label) into one yielding (img, label, dataset_idx).

    Used for simultaneously evaluating on multiple distinct datasets in the `evaluate_batch` loop.
    A custom reducer is then used to report metrics on them separately
    """

    def __init__(self, datasets: List[Dataset], dataset_names: List[str]):
        self.datasets = datasets
        self.dataset_names = dataset_names
        assert len(datasets) == len(dataset_names)
        self.lens = [len(dataset) for dataset in datasets]

    def __getitem__(self, idx: int) -> Tuple[torch.Tensor, int, int]:
        d_idx = 0
        for i, d_len in enumerate(self.lens):
            if idx >= d_len:
                idx -= d_len
                d_idx += 1
        img, label = self.datasets[d_idx][idx]
        return (img, label, d_idx)

    def __len__(self) -> int:
        return sum(self.lens)


class TransformDataset(Dataset):
    """
    Wrap dataset in a transform.

    Used for torchvision Datasets when we don't know the transform at time of construction.
    """

    def __init__(self, dataset: Dataset, transform: Callable):
        self.dataset = dataset
        self.transform = transform

    def __getitem__(self, idx: int) -> Tuple[torch.Tensor, int]:
        sample, target = self.dataset[idx]
        return self.transform(sample), target

    def __len__(self) -> int:
        return len(self.dataset)


class DoubleTransformDataset(Dataset):
    """
    Returns result of applying two separate transforms to the given dataset.
    """

    def __init__(self, dataset: Dataset, transform1: Callable, transform2: Callable) -> None:
        self.dataset = dataset
        self.transform1 = transform1
        self.transform2 = transform2

    def __getitem__(self, idx: int) -> Tuple[torch.Tensor, torch.Tensor, int]:
        sample, target = self.dataset[idx]
        return self.transform1(sample), self.transform2(sample), target

    def __len__(self) -> int:
        return len(self.dataset)


class GCSImageFolder(Dataset):
    """
    Stream an image dataset from Google Cloud Storage, stored in torch ImageFolder format.
    Adapted from gaea_pytorch example.
    """

    def __init__(
        self,
        bucket_name: str,
        blob_list_path: str,
        streaming: bool = True,
        data_download_dir: str = None,
    ) -> None:
        """
        Args:
            bucket_name: GCS bucket name, without gs:// prefix.
            blob_list_path: Path within the bucket to a file containing a newline separated list of blob locations.
                            Generate using the generate_blob_list.py script and upload to bucket.
            streaming: Flag for whether to always stream data. If False, will pull data once and then store on disk.
            data_download_dir: Location to store data if streaming is False.
        """
        self._bucket_name = bucket_name
        self._target_dir = data_download_dir
        # Streaming always downloads image from GCP regardless of whether it
        # has been downloaded before.
        # When streaming is false, we will save the downloaded image to disk and
        # check whether the image is available before sending a download request
        # to the GCP bucket.
        self._streaming = streaming
        self._storage_client = storage.Client()
        self._bucket = self._storage_client.bucket(bucket_name)
        # When the dataset is first initialized, we'll loop through to catalogue the classes (subdirectories)
        # This step might take a long time.
        self._imgs_paths = []
        self._labels = []
        self._subdir_to_class: Dict[str, int] = {}
        class_count = 0
        blob_list_blob = self._bucket.blob(blob_list_path)
        blob_list_io = StringIO(
            download_gcs_blob_with_backoff(blob_list_blob, n_retries=4, max_backoff=2).decode(
                "utf-8"
            )
        )
        blob_list = [s.strip() for s in blob_list_io.readlines()]
        for path in blob_list:
            self._imgs_paths.append(path)
            sub_dir = path.split("/")[-2]
            if sub_dir not in self._subdir_to_class:
                self._subdir_to_class[sub_dir] = class_count
                class_count += 1
            self._labels.append(self._subdir_to_class[sub_dir])
        print("There are {} records in dataset.".format(len(self._imgs_paths)))

    def __len__(self) -> int:
        return len(self._imgs_paths)

    def __getitem__(self, idx: int) -> Tuple[PILImage.Image, int]:
        img_path = self._imgs_paths[idx]
        blob = self._bucket.blob(img_path)
        if self._streaming:
            img_str = download_gcs_blob_with_backoff(blob)
        else:
            assert self._target_dir is not None, "Must pass download directory if not streaming."
            target_path = os.path.join(self._target_dir, img_path)
            if not os.path.exists(target_path):
                os.makedirs(target_path, exist_ok=True)
                img_str = download_gcs_blob_with_backoff(blob)
                with open(target_path, "wb") as f:
                    f.write(img_str)
            else:
                with open(target_path, "rb") as f:
                    img_str = f.read()
        img_bytes = BytesIO(img_str)
        img = PILImage.open(img_bytes)
        img = img.convert("RGB")
        return img, self._labels[idx]


class DatasetSplit(Enum):
    """
    TRAIN: Can be unlabeled, used for training self-supervised network.
    CLS_TRAIN: Requires labels, used for training classifier.
    CLS_VALIDATION: Requires labels, used for selecting optimal classifier learning rate.
    TEST: Requires labels, used for final reported accuracy.
    """

    TRAIN = auto()
    CLS_TRAIN = auto()
    CLS_VALIDATION = auto()
    TEST = auto()


@dataclass
class DatasetMetadata:
    mean: Tuple[float, float, float]
    std: Tuple[float, float, float]
    num_classes: int


DATASET_METADATA_BY_NAME = {
    "cifar10": DatasetMetadata(
        mean=(0.4914, 0.4822, 0.4465),
        std=(0.2023, 0.1994, 0.2010),
        num_classes=10,
    ),
    "stl10": DatasetMetadata(
        mean=(0.4914, 0.4822, 0.4465),
        std=(0.2023, 0.1994, 0.2010),
        num_classes=10,
    ),
    "imagenet": DatasetMetadata(
        mean=(0.485, 0.456, 0.406),
        std=(0.229, 0.224, 0.225),
        num_classes=1000,
    ),
}


def build_training_transform(
    settings: AttrDict,
    mean: Tuple[float, float, float],
    std: Tuple[float, float, float],
) -> nn.Module:
    return T.Compose(
        [
            T.RandomResizedCrop(
                settings.random_crop_size,
                scale=(settings.random_crop_min_scale, 1.0),
                interpolation=T.InterpolationMode.BICUBIC,
            ),
            T.RandomHorizontalFlip(settings.random_hflip_prob),
            T.RandomApply(
                [
                    T.ColorJitter(
                        settings.color_jitter_brightness,
                        settings.color_jitter_contrast,
                        settings.color_jitter_saturation,
                        settings.color_jitter_hue,
                    )
                ],
                p=settings.color_jitter_prob,
            ),
            T.RandomGrayscale(p=settings.grayscale_prob),
            T.RandomApply(
                [
                    T.GaussianBlur(
                        settings.gaussian_blur_kernel_size,
                        (
                            settings.gaussian_blur_min_std,
                            settings.gaussian_blur_max_std,
                        ),
                    )
                ],
                p=settings.gaussian_blur_prob,
            ),
            T.RandomSolarize(0.5, p=settings.solarization_prob),
            T.ToTensor(),
            T.Normalize(mean=mean, std=std),
        ]
    )


def build_evaluation_transform(
    settings: AttrDict,
    mean: Tuple[float, float, float],
    std: Tuple[float, float, float],
) -> nn.Module:
    return T.Compose(
        [
            T.Resize(settings.resize_short_edge),
            T.CenterCrop(settings.center_crop_size),
            T.ToTensor(),
            T.Normalize(mean=mean, std=std),
        ]
    )


def split_supervised_dataset(
    data_config: AttrDict,
    split: DatasetSplit,
    build_train: Callable[[], Dataset],
    build_val: Callable[[], Dataset],
) -> Dataset:
    """
    Takes a pre-existing train/val split supervised dataset and subdivides into TRAIN, CLS_TRAIN, CLS_VALIDATION,
    and TEST
    """
    if split in [
        DatasetSplit.TRAIN,
        DatasetSplit.CLS_TRAIN,
        DatasetSplit.CLS_VALIDATION,
    ]:
        train_dataset = build_train()
        indices = list(range(len(train_dataset)))
        random.seed(0)
        random.shuffle(indices)
        val_size = data_config.validation_subset_size
        val_indices = indices[:val_size]
        train_indices = indices[val_size:]
        if split in [DatasetSplit.TRAIN, DatasetSplit.CLS_TRAIN]:
            return torch.utils.data.Subset(train_dataset, train_indices)
        elif split == DatasetSplit.CLS_VALIDATION:
            return torch.utils.data.Subset(train_dataset, val_indices)
        else:
            # Unreachable
            raise Exception()
    elif split == DatasetSplit.TEST:
        return build_val()
    else:
        # Unreachable
        raise Exception()


def build_imagenet(data_config: AttrDict, download_dir: str, split: DatasetSplit) -> Dataset:
    # We assume ImageNet is already on disk, as e.g. a mounted NFS drive.
    def build_train() -> Dataset:
        # return ImageFolder(os.path.join(download_dir, "train"))
        return GCSImageFolder(
            data_config.gcs_bucket,
            data_config.gcs_train_blob_list_path,
            streaming=True,
        )

    def build_val() -> Dataset:
        return GCSImageFolder(
            data_config.gcs_bucket,
            data_config.gcs_validation_blob_list_path,
            streaming=True,
        )

    return split_supervised_dataset(data_config, split, build_train, build_val)


def download_and_build_cifar10(
    data_config: AttrDict, download_dir: str, split: DatasetSplit
) -> Dataset:
    def build_train() -> Dataset:
        return CIFAR10(download_dir, train=True, download=True)

    def build_val() -> Dataset:
        return CIFAR10(download_dir, train=False, download=True)

    return split_supervised_dataset(data_config, split, build_train, build_val)


def download_and_build_stl10(
    data_config: AttrDict, download_dir: str, split: DatasetSplit
) -> Dataset:
    # For the validation set, we split off part of the test set, since training set is mostly unlabeled.
    # This is also done in
    # https://generallyintelligent.ai/blog/2020-08-24-understanding-self-supervised-contrastive-learning/
    # so our results should be comparable.
    if split == DatasetSplit.TRAIN:
        return STL10(download_dir, split="train+unlabeled", download=True)
    elif split == DatasetSplit.CLS_TRAIN:
        # Can't use unlabeled images for classifier training.
        return STL10(download_dir, split="train", download=True)
    else:
        test_dataset = STL10(download_dir, split="test", download=True)
        indices = list(range(len(test_dataset)))
        random.seed(0)
        random.shuffle(indices)
        val_size = data_config.validation_subset_size
        val_indices = indices[:val_size]
        test_indices = indices[val_size:]
        if split == DatasetSplit.TEST:
            return torch.utils.data.Subset(test_dataset, test_indices)
        elif split == DatasetSplit.CLS_VALIDATION:
            return torch.utils.data.Subset(test_dataset, val_indices)
        else:
            # Unreachable
            raise Exception()


DATASET_BUILD_MAP = {
    "cifar10": download_and_build_cifar10,
    "imagenet": build_imagenet,
    "stl10": download_and_build_stl10,
}


def build_dataset(data_config: AttrDict, split: DatasetSplit) -> Dataset:
    """
    Returns specified Dataset object.

    Downloads CIFAR10 / STL10 if necessary.  Will not download ImageNet -- assumed to be on disk.
    """
    name = data_config.dataset_name
    download_dir = data_config.download_dir
    if name in DATASET_BUILD_MAP:
        # Lock so that only one process attempts download at a time.
        os.makedirs(download_dir, exist_ok=True)
        with FileLock(os.path.join(download_dir, "download.lock")):
            dataset = DATASET_BUILD_MAP[name](data_config, download_dir, split)
        if split == DatasetSplit.TRAIN:
            return DoubleTransformDataset(
                dataset,
                data_config.train_transform_fn1,
                data_config.train_transform_fn2,
            )
        elif split == DatasetSplit.CLS_TRAIN:
            return TransformDataset(dataset, data_config.train_transform_fn1)
        else:
            return TransformDataset(dataset, data_config.eval_transform_fn)
    else:
        raise Exception(
            f'"{name}" is not a supported dataset.  Datasets supported are {list(DATASET_BUILD_MAP.keys())}.'
        )
