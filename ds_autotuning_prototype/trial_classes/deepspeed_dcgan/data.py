import contextlib
import os
from typing import cast

import filelock
import torch
import torchvision.datasets as dset
import torchvision.transforms as transforms
from attrdict import AttrDict

CHANNELS_BY_DATASET = {
    "imagenet": 3,
    "folder": 3,
    "lfw": 3,
    "lsun": 3,
    "cifar10": 3,
    "mnist": 1,
    "fake": 3,
    "celeba": 3,
}


def get_dataset(data_config: AttrDict) -> torch.utils.data.Dataset:
    if data_config.dataroot is None and str(data_config.dataset).lower() != "fake":
        raise ValueError('`dataroot` parameter is required for dataset "%s"' % data_config.dataset)
    if data_config.dataroot is None:
        context = contextlib.nullcontext()
    else:
        # Ensure that only one local process attempts to download/validate datasets at once.
        context = filelock.FileLock(os.path.join(data_config.dataroot, ".lock"))
    with context:
        if data_config.dataset in ["imagenet", "folder", "lfw"]:
            # folder dataset
            dataset = dset.ImageFolder(
                root=data_config.dataroot,
                transform=transforms.Compose(
                    [
                        transforms.Resize(data_config.image_size),
                        transforms.CenterCrop(data_config.image_size),
                        transforms.ToTensor(),
                        transforms.Normalize((0.5, 0.5, 0.5), (0.5, 0.5, 0.5)),
                    ]
                ),
            )
        elif data_config.dataset == "lsun":
            classes = [c + "_train" for c in data_config.classes.split(",")]
            dataset = dset.LSUN(
                root=data_config.dataroot,
                classes=classes,
                transform=transforms.Compose(
                    [
                        transforms.Resize(data_config.image_size),
                        transforms.CenterCrop(data_config.image_size),
                        transforms.ToTensor(),
                        transforms.Normalize((0.5, 0.5, 0.5), (0.5, 0.5, 0.5)),
                    ]
                ),
            )
        elif data_config.dataset == "cifar10":
            dataset = dset.CIFAR10(
                root=data_config.dataroot,
                download=True,
                transform=transforms.Compose(
                    [
                        transforms.Resize(data_config.image_size),
                        transforms.ToTensor(),
                        transforms.Normalize((0.5, 0.5, 0.5), (0.5, 0.5, 0.5)),
                    ]
                ),
            )
        elif data_config.dataset == "mnist":
            dataset = dset.MNIST(
                root=data_config.dataroot,
                download=True,
                transform=transforms.Compose(
                    [
                        transforms.Resize(data_config.image_size),
                        transforms.ToTensor(),
                        transforms.Normalize((0.5,), (0.5,)),
                    ]
                ),
            )
        elif data_config.dataset == "fake":
            dataset = dset.FakeData(
                image_size=(3, data_config.image_size, data_config.image_size),
                transform=transforms.ToTensor(),
            )
        elif data_config.dataset == "celeba":
            dataset = dset.ImageFolder(
                root=data_config.dataroot,
                transform=transforms.Compose(
                    [
                        transforms.Resize(data_config.image_size),
                        transforms.CenterCrop(data_config.image_size),
                        transforms.ToTensor(),
                        transforms.Normalize((0.5, 0.5, 0.5), (0.5, 0.5, 0.5)),
                    ]
                ),
            )
        else:
            raise Exception(f"Unknown dataset {data_config.dataset}")
    return cast(torch.utils.data.Dataset, dataset)
