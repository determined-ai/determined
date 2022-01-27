from attrdict import AttrDict
from dataclasses import dataclass
from enum import auto, Enum
import os
import random
from typing import Callable, List, Tuple

import torch
import torch.nn as nn
from torchvision.datasets import CIFAR10, ImageFolder, STL10
import torchvision.transforms as T
from torch.utils.data import Dataset


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

    def __init__(
        self, dataset: Dataset, transform1: Callable, transform2: Callable
    ) -> None:
        self.dataset = dataset
        self.transform1 = transform1
        self.transform2 = transform2

    def __getitem__(self, idx: int) -> Tuple[torch.Tensor, torch.Tensor, int]:
        sample, target = self.dataset[idx]
        a, b, c = self.transform1(sample), self.transform2(sample), target
        if torch.any(torch.isnan(b)):
            print(f"NaN detected.  Original sample idx {idx}.")
            print("Sample:")
            print(sample)
        return a, b, c

    def __len__(self) -> int:
        return len(self.dataset)


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
    # TODO: Add resize and center crop for ImageNet.
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
        random.seed(0)
        indices = list(range(len(train_dataset)))
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


def build_imagenet(
    data_config: AttrDict, download_dir: str, split: DatasetSplit
) -> Dataset:
    # We assume ImageNet is already on disk, as e.g. a mounted NFS drive.
    def build_train() -> Dataset:
        return ImageFolder(os.path.join(download_dir, "train"))

    def build_val() -> Dataset:
        return ImageFolder(os.path.join(download_dir, "validation"))

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
        random.seed(0)
        indices = list(range(len(test_dataset)))
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


def build_dataset(
    data_config: AttrDict, download_dir: str, split: DatasetSplit
) -> Dataset:
    """
    Returns specified Dataset object.

    Downloads CIFAR10 / STL10 if necessary.  Relies on pre-downloaded ImageNet specified on disk.
    """
    name = data_config.dataset_name
    if name in DATASET_BUILD_MAP:
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
