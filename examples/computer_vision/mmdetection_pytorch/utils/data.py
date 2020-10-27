import math
from functools import partial

import numpy as np
import torch
from torch.utils.data import Sampler

from google.cloud import storage

from mmcv.fileio import BaseStorageBackend, FileClient
from .collate import collate
from mmcv.parallel.data_container import DataContainer
from mmdet.datasets import build_dataset
from mmcv.utils.config import Config, ConfigDict

from determined.util import download_gcs_blob_with_backoff
from determined.pytorch import DataLoader

# from torch.utils.data import DataLoader


class GCSBackend(BaseStorageBackend):
    def __init__(self, bucket_name):
        self._storage_client = storage.Client(project="determined-ai")
        self._bucket = self._storage_client.bucket(bucket_name)

    def convert_filepath(self, filepath):
        tokens = filepath.split("/")
        directory = tokens[-2]
        filename = tokens[-1]
        return "{}/{}".format(directory, filename)

    def get(self, filepath):
        filepath = self.convert_filepath(filepath)
        blob = self._bucket.blob(filepath)
        img_str = download_gcs_blob_with_backoff(blob)
        return img_str

    def get_text(self, filepath):
        return NotImplementedError


FileClient.register_backend("gcs", GCSBackend)


class FakeBackend(BaseStorageBackend):
    def __init__(self):
        self.data = None

    def get(self, filepath):
        if self.data is None:
            with open("imgs/train_metrics.png", "rb") as f:
                img_str = f.read()
            self.data = img_str
        return self.data

    def get_text(self, filepath):
        return NotImplementedError


FileClient.register_backend("fake", FakeBackend)


class MyGroupSampler(Sampler):
    """
    Modifies DistributedGroupSampler from https://github.com/open-mmlab/mmdetection/blob/master/mmdet/datasets/samplers/group_sampler.py
    to work with our Dataloader which automatically handles sharding for distributed training.

    Arguments:
        dataset: Dataset used for sampling.
        num_replicas (optional): Number of processes participating in
            distributed training.
    """

    def __init__(
        self,
        dataset,
        samples_per_gpu,
        num_replicas,
    ):
        self.dataset = dataset
        self.samples_per_gpu = samples_per_gpu
        self.num_replicas = num_replicas

        assert hasattr(self.dataset, "flag")
        self.flag = self.dataset.flag
        self.group_sizes = np.bincount(self.flag)

        self.num_samples = 0
        for i, j in enumerate(self.group_sizes):
            self.num_samples += (
                int(
                    math.ceil(
                        self.group_sizes[i]
                        * 1.0
                        / self.samples_per_gpu
                        / self.num_replicas
                    )
                )
                * self.samples_per_gpu
            )
        self.total_size = self.num_samples * self.num_replicas

    def __iter__(self):
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

        # new_indices = indices.copy()
        # for r in range(self.num_replicas):
        #    offset = self.num_samples * r
        #    for s in range(self.num_samples):
        #        new_indices[s*self.num_replicas + r] = indices[offset + s]
        # return iter(new_indices)

    def __len__(self):
        return self.total_size


def build_dataloader(cfg, num_samples_per_gpu, num_replicas, num_workers, shuffle):
    dataset = build_dataset(cfg)
    sampler = (
        MyGroupSampler(dataset, num_samples_per_gpu, num_replicas) if shuffle else None
    )
    # may need to look into collate_fn for distributed data and init_fn for seeding
    return dataset, DataLoader(
        dataset,
        batch_size=num_samples_per_gpu,
        num_workers=num_workers,
        sampler=sampler,
        collate_fn=partial(collate, samples_per_gpu=num_samples_per_gpu),
        pin_memory=False,
    )


def sub_backend(backend, cfg):
    """
    Replace default backend for getting files with GCSBackend which downloads
    a file from a GCS bucket.
    """
    if type(cfg) in [Config, ConfigDict]:
        backend_cfg = {
            "gcs": {"backend": "gcs", "bucket_name": "determined-ai-coco-dataset"},
            "fake": {"backend": "fake"},
        }

        if "type" in cfg and cfg["type"] == "LoadImageFromFile":
            cfg["file_client_args"] = backend_cfg[backend]
        else:
            for k in cfg:
                sub_backend(backend, cfg[k])
    else:
        if isinstance(cfg, list):
            for i in cfg:
                sub_backend(backend, i)


def decontainer(data):
    """
    Flatten DataContainer objects used by original MMDetection library.
    """
    if type(data) in [list, tuple]:
        data = [decontainer(d) for d in data]
    elif isinstance(data, dict):
        data = {k: decontainer(data[k]) for k in data}
    elif isinstance(data, DataContainer):
        data = data.data[0]
    return data


if __name__ == "__main__":
    # Test backend
    backend = GCSBackend("determined-ai-coco-dataset")
    img_bytes = backend.get("annotations2017/instances_val2017.json")
    import os

    with open("/tmp/instances_val2017.json", "wb") as f:
        f.write(img_bytes)
    print("done")

    # Test dataloader
    from mmcv import Config
    from mmdet.models import build_detector
    from mmcv.runner import load_checkpoint

    cfg = Config.fromfile("configs/retinanet/retinanet_r50_fpn_1x_coco.py")
    sub_backend("gcs", cfg)
    cfg.data.val.ann_file = "/tmp/instances_val2017.json"
    cfg.data.val.test_mode = True

    model = build_detector(cfg.model, train_cfg=cfg.train_cfg, test_cfg=cfg.test_cfg)
    checkpoint = load_checkpoint(
        model, "./retinanet_r50_fpn_1x_coco_20200130-c2398f9e.pth"
    )
    model.cuda()
    model.eval()

    dataset, data_loader = build_dataloader(cfg.data.val, 1, 1, 8, False)

    # from mmdet.core import encode_mask_results
    # results = []
    # for i, batch in enumerate(data_loader):
    #    batch = decontainer(batch)
    #    batch['img'][0] = batch['img'][0].cuda()
    #    with torch.no_grad():
    #        result = model(return_loss=False, rescale=True, **batch)
    #    if isinstance(result[0], tuple):
    #        result = [(bbox_results, encode_mask_results(mask_results))
    #                  for bbox_results, mask_results in result]
    #    results.extend(result)

    # eval_kwargs = cfg.evaluation
    # for key in ['interval', 'tmpdir', 'start', 'gpu_collect']:
    #    eval_kwargs.pop(key, None)
    # metrics = data_loader.dataset.evaluate(results, **eval_kwargs)
    # print(metrics)
