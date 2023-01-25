"""
Data utility functions for creating the dataset and dataloader for use with MMDetTrial.
"""
import functools
import logging
import math
import os
from typing import Any, Iterator, Tuple

import filelock
import mmcv
import mmcv.parallel
import mmdet.datasets
import numpy as np
import torch
import torch.utils.data as torch_data

import determined.pytorch as det_torch


class GroupSampler(torch.utils.data.Sampler):
    """
    Modifies DistributedGroupSampler from
    https://github.com/open-mmlab/mmdetection/blob/master/mmdet/datasets/samplers/group_sampler.py
    to work with our Dataloader which automatically handles sharding for distributed training.
    """

    def __init__(
        self,
        dataset: torch_data.Dataset,
        samples_per_gpu: int,
        num_replicas: int,
    ):
        """
        This sampler will generate indices such that each batch will belong to the same group.
        For example, if batch size is `b`, samples 1 to b will be one group, samples b+1 to 2b
        another group, etc. Hence, we effectively have len(dataset)/batch_size group indices which
        get shuffled every epoch.

        Arguments:
            dataset: dataset that has a flag attribute to indicate group member for each sample.
            samples_per_gpu: number of samples per slot.
            num_replicas: number of processes participating in distributed training.
        """
        self.dataset = dataset
        self.samples_per_gpu = samples_per_gpu
        self.num_replicas = num_replicas

        assert hasattr(self.dataset, "flag")
        self.flag = self.dataset.flag  # type: ignore
        self.group_sizes = np.bincount(self.flag)

        self.num_samples = 0
        for size in self.group_sizes:
            self.num_samples += (
                int(math.ceil(size * 1.0 / self.samples_per_gpu / self.num_replicas))
                * self.samples_per_gpu
            )
        self.total_size = self.num_samples * self.num_replicas

    def __iter__(self) -> Iterator[Any]:
        indices = []
        for i, size in enumerate(self.group_sizes):
            if size > 0:
                indice = np.where(self.flag == i)[0]
                assert len(indice) == size
                indice = indice[list(torch.randperm(int(size)))].tolist()
                extra = int(
                    math.ceil(size * 1.0 / self.samples_per_gpu / self.num_replicas)
                ) * self.samples_per_gpu * self.num_replicas - len(indice)
                # pad indice
                tmp = indice.copy()
                for _ in range(extra // size):
                    indice.extend(tmp)
                indice.extend(tmp[: extra % size])
                indices.extend(indice)

        assert len(indices) == self.total_size

        indices = [
            indices[j]
            for i in list(torch.randperm(len(indices) // self.samples_per_gpu))
            for j in range(i * self.samples_per_gpu, (i + 1) * self.samples_per_gpu)
        ]

        return iter(indices)

    def __len__(self) -> int:
        return self.total_size


def maybe_download_ann_file(cfg: mmcv.Config) -> None:
    """
    mmdetection expects the annotation files to be available in the disk at a specific directory
    to initialize a dataset.  However, the annotation file is usually not available locally when a
    cloud backend is used. We will try to download the annotation file if it exists from the cloud
    if the backend is gcp or s3.

    Arguments:
        cfg: mmcv.Config with dataset specifications.
    """
    if "dataset" in cfg:
        dataset = cfg.dataset
    else:
        dataset = cfg
    ann_dir = "/".join(dataset.ann_file.split("/")[0:-1])
    os.makedirs(ann_dir, exist_ok=True)
    lock = filelock.FileLock(dataset.ann_file + ".lock")

    with lock:
        if not os.path.isfile(dataset.ann_file):
            try:
                assert (
                    dataset.pipeline[0].type == "LoadImageFromFile"
                ), "First step of dataset.pipeline is not LoadImageFromFile."
                file_client_args = dataset.pipeline[0].file_client_args
                file_client = mmcv.FileClient(**file_client_args)
                ann_bytes = file_client.get(dataset.ann_file)
                logging.info(
                    f'Downloading annotation file using {file_client_args["backend"]} backend.'
                )
                with open(dataset.ann_file, "wb") as f:
                    f.write(ann_bytes)
            except Exception as e:
                logging.error(
                    f"Could not download missing annotation file.  Encountered {e}."
                    f"Please make sure it is available at the following path {dataset.ann_file}."
                )


class DatasetWithIndex(torch.utils.data.Dataset):
    """
    The way Determined shards a dataset for distributed training and then gathers predictions in
    custom reducers does not maintain dataset ordering.  Here, we include the index in the dataset
    so that predictions can be sorted correctly at evaluation time.
    """

    def __init__(self, dataset: torch.utils.data.Dataset):
        self.dataset = dataset

    def __getattr__(self, item: Any) -> Any:
        return getattr(self.dataset, item)

    def __getitem__(self, idx: int) -> Any:
        sample = self.dataset[idx]
        if "idx" not in sample["img_metas"][0].data:
            sample["img_metas"][0].data["idx"] = idx
        return sample

    def __len__(self) -> int:
        return self.dataset.__len__()  # type: ignore


def build_dataloader(
    cfg: mmcv.Config,
    split: "str",
    context: det_torch.PyTorchTrialContext,
    shuffle: bool,
) -> Tuple[torch_data.Dataset, det_torch.DataLoader]:
    """
    Build the dataset and dataloader according to cfg and sampler parameters.

    Arguments:
        cfg: mmcv.Config with dataset specifications.
        split: one of train, val, or test. If val or test, annotations are not loaded.
        context: PyTorchTrialContext with seed info used to seed the dataloader workers.
        shuffle: whether to shuffle indices for data loading.
    Returns:
        dataset and dataloader
    """
    assert split in ["train", "val", "test"], "argument split must be one of train, val, or test."
    num_samples_per_gpu = context.get_per_slot_batch_size()
    num_replicas = context.distributed.get_size()
    num_workers = cfg.workers_per_gpu
    test_mode = False if split == "train" else True

    cfg = eval(f"cfg.{split}")
    maybe_download_ann_file(cfg)

    dataset = mmdet.datasets.build_dataset(cfg, {"test_mode": test_mode})
    if test_mode:
        dataset = DatasetWithIndex(dataset)
    sampler = GroupSampler(dataset, num_samples_per_gpu, num_replicas) if shuffle else None

    return dataset, det_torch.DataLoader(
        dataset,
        batch_size=num_samples_per_gpu,
        num_workers=num_workers,
        sampler=sampler,
        collate_fn=functools.partial(mmcv.parallel.collate, samples_per_gpu=num_samples_per_gpu),
        pin_memory=False,
        worker_init_fn=functools.partial(
            mmdet.datasets.builder.worker_init_fn,
            seed=context.get_trial_seed(),
            rank=context.distributed.get_rank(),
            num_workers=num_workers,
        ),
    )
