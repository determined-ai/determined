# Copyright (c) Facebook, Inc. and its affiliates. All Rights Reserved
"""
COCO dataset which returns image_id for evaluation.
Mostly copy-paste from https://github.com/pytorch/vision/blob/13b35ff/references/detection/coco_utils.py
"""
import os
from io import BytesIO

from google.cloud import storage
from determined.util import download_gcs_blob_with_backoff

import torchvision
from PIL import Image

from detr.util.misc import nested_tensor_from_tensor_list
from detr.datasets.coco import ConvertCocoPolysToMask, make_coco_transforms


def unwrap_collate_fn(batch):
    batch = list(zip(*batch))
    batch[0] = nested_tensor_from_tensor_list(batch[0])
    batch[0] = {"tensors": batch[0].tensors, "mask": batch[0].mask}
    return tuple(batch)


class GCSBackend:
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


class FakeBackend:
    def __init__(self):
        self.data = None

    def get(self, filepath):
        if self.data is None:
            with open("imgs/train_curves.png", "rb") as f:
                img_str = f.read()
            self.data = img_str
        return self.data


class CocoDetection(torchvision.datasets.CocoDetection):
    def __init__(self, bucket_name, img_folder, ann_file, transforms, return_masks):
        super(CocoDetection, self).__init__(bucket_name, ann_file)
        self.img_folder = img_folder
        self._transforms = transforms
        self.prepare = ConvertCocoPolysToMask(return_masks)
        if bucket_name is None:
            self.backend = FakeBackend()
        else:
            self.backend = GCSBackend(bucket_name)

    def __getitem__(self, idx):
        coco = self.coco
        img_id = self.ids[idx]
        ann_ids = coco.getAnnIds(imgIds=img_id)
        target = coco.loadAnns(ann_ids)
        path = coco.loadImgs(img_id)[0]["file_name"]
        img_bytes = BytesIO(self.backend.get(os.path.join(self.img_folder, path)))

        img = Image.open(img_bytes).convert("RGB")

        image_id = self.ids[idx]
        target = {"image_id": image_id, "annotations": target}
        img, target = self.prepare(img, target)
        if self._transforms is not None:
            img, target = self._transforms(img, target)
        return img, target


def build_dataset(image_set, args):
    root = args.bucket_name
    mode = "instances"
    PATHS = {
        "train": (f"{root}/train2017", f"/tmp/{mode}_train2017.json"),
        "val": (f"{root}/val2017", f"/tmp/{mode}_val2017.json"),
    }

    img_folder, ann_file = PATHS[image_set]
    dataset = CocoDetection(
        args.bucket_name,
        img_folder,
        ann_file,
        transforms=make_coco_transforms(image_set),
        return_masks=args.masks,
    )
    return dataset
