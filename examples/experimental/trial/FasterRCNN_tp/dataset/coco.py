# -*- coding: utf-8 -*-

import json
import os
import time

import boto3
import numpy as np
from google.cloud import storage

import tqdm
from config import config as cfg
from .dataset import DatasetRegistry, DatasetSplit
from determined.util import download_gcs_blob_with_backoff
from tensorpack.utils import logger
from tensorpack.utils.timer import timed_operation

__all__ = ["register_coco"]


class COCODetection(DatasetSplit):
    # handle the weird (but standard) split of train and val
    _INSTANCE_TO_BASEDIR = {"valminusminival2014": "val2014", "minival2014": "val2014"}

    """
    Mapping from the incontinuous COCO category id to an id in [1, #category]
    For your own coco-format, dataset, change this to an **empty dict**.
    """
    COCO_id_to_category_id = {
        13: 12,
        14: 13,
        15: 14,
        16: 15,
        17: 16,
        18: 17,
        19: 18,
        20: 19,
        21: 20,
        22: 21,
        23: 22,
        24: 23,
        25: 24,
        27: 25,
        28: 26,
        31: 27,
        32: 28,
        33: 29,
        34: 30,
        35: 31,
        36: 32,
        37: 33,
        38: 34,
        39: 35,
        40: 36,
        41: 37,
        42: 38,
        43: 39,
        44: 40,
        46: 41,
        47: 42,
        48: 43,
        49: 44,
        50: 45,
        51: 46,
        52: 47,
        53: 48,
        54: 49,
        55: 50,
        56: 51,
        57: 52,
        58: 53,
        59: 54,
        60: 55,
        61: 56,
        62: 57,
        63: 58,
        64: 59,
        65: 60,
        67: 61,
        70: 62,
        72: 63,
        73: 64,
        74: 65,
        75: 66,
        76: 67,
        77: 68,
        78: 69,
        79: 70,
        80: 71,
        81: 72,
        82: 73,
        84: 74,
        85: 75,
        86: 76,
        87: 77,
        88: 78,
        89: 79,
        90: 80,
    }  # noqa

    """
    80 names for COCO
    For your own coco-format dataset, change this.
    """
    class_names = [
        "person",
        "bicycle",
        "car",
        "motorcycle",
        "airplane",
        "bus",
        "train",
        "truck",
        "boat",
        "traffic light",
        "fire hydrant",
        "stop sign",
        "parking meter",
        "bench",
        "bird",
        "cat",
        "dog",
        "horse",
        "sheep",
        "cow",
        "elephant",
        "bear",
        "zebra",
        "giraffe",
        "backpack",
        "umbrella",
        "handbag",
        "tie",
        "suitcase",
        "frisbee",
        "skis",
        "snowboard",
        "sports ball",
        "kite",
        "baseball bat",
        "baseball glove",
        "skateboard",
        "surfboard",
        "tennis racket",
        "bottle",
        "wine glass",
        "cup",
        "fork",
        "knife",
        "spoon",
        "bowl",
        "banana",
        "apple",
        "sandwich",
        "orange",
        "broccoli",
        "carrot",
        "hot dog",
        "pizza",
        "donut",
        "cake",
        "chair",
        "couch",
        "potted plant",
        "bed",
        "dining table",
        "toilet",
        "tv",
        "laptop",
        "mouse",
        "remote",
        "keyboard",
        "cell phone",
        "microwave",
        "oven",
        "toaster",
        "sink",
        "refrigerator",
        "book",
        "clock",
        "vase",
        "scissors",
        "teddy bear",
        "hair drier",
        "toothbrush",
    ]  # noqa
    cfg.DATA.CLASS_NAMES = ["BG"] + class_names

    def __init__(self, basedir, split, is_aws, is_gcs):
        """
        Args:
            basedir (str): root of the dataset which contains the subdirectories for each split and annotations
            split (str): the name of the split, e.g. "train2017".
                The split has to match an annotation file in "annotations/" and a directory of images.
            is_aws (bool): is the dataset in AWS
            is_gcs (bool): is the dataset in GCS

        Examples:
            For a directory of this structure:

            DIR/
              annotations/
                instances_XX.json
                instances_YY.json
              XX/
              YY/

            use `COCODetection(DIR, 'XX')` and `COCODetection(DIR, 'YY')`
        """
        self.is_aws = is_aws
        self.is_gcs = is_gcs
        if is_aws or is_gcs:
            annotation_file = "annotations/instances_{}.json".format(split)
            self._imgdir = self._INSTANCE_TO_BASEDIR.get(split, split)
        else:
            basedir = os.path.expanduser(basedir)
            self._imgdir = os.path.realpath(
                os.path.join(basedir, self._INSTANCE_TO_BASEDIR.get(split, split))
            )
            assert os.path.isdir(self._imgdir), "{} is not a directory!".format(self._imgdir)
            annotation_file = os.path.join(basedir, "annotations/instances_{}.json".format(split))
            assert os.path.isfile(annotation_file), annotation_file

        from pycocotools.coco import COCO

        if is_aws:
            self.coco = COCO()
            print("loading annotations into memory...")
            s3 = boto3.resource("s3")
            s3_object = s3.meta.client.get_object(
                Bucket="determined-ai-coco-dataset", Key=annotation_file
            )
            self.coco.dataset = json.loads(s3_object["Body"].read())
            assert (
                type(self.coco.dataset) == dict
            ), "annotation file format {} not supported".format(type(self.coco.dataset))
            self.coco.createIndex()
        elif is_gcs:
            self.coco = COCO()
            print("loading annotations into memory...")
            tic = time.time()
            c = storage.Client.create_anonymous_client()
            bucket = c.get_bucket("determined-ai-coco-dataset")
            blob = bucket.blob(annotation_file)
            s = download_gcs_blob_with_backoff(blob)
            self.coco.dataset = json.loads(s)
            assert (
                type(self.coco.dataset) == dict
            ), "annotation file format {} not supported".format(type(self.coco.dataset))
            print("Done (t={:0.2f}s)".format(time.time() - tic))
            self.coco.createIndex()
        else:
            self.coco = COCO(annotation_file)
        self.annotation_file = annotation_file
        logger.info("Instances loaded from {}.".format(annotation_file))

    # https://github.com/cocodataset/cocoapi/blob/master/PythonAPI/pycocoEvalDemo.ipynb
    def print_coco_metrics(self, json_file):
        """
        Args:
            json_file (str): path to the results json file in coco format
        Returns:
            dict: the evaluation metrics
        """
        from pycocotools.cocoeval import COCOeval

        ret = {}
        cocoDt = self.coco.loadRes(json_file)
        cocoEval = COCOeval(self.coco, cocoDt, "bbox")
        cocoEval.evaluate()
        cocoEval.accumulate()
        cocoEval.summarize()
        fields = ["IoU=0.5:0.95", "IoU=0.5", "IoU=0.75", "small", "medium", "large"]
        for k in range(6):
            ret["mAP(bbox)/" + fields[k]] = cocoEval.stats[k]

        json_obj = json.load(open(json_file))
        if len(json_obj) > 0 and "segmentation" in json_obj[0]:
            cocoEval = COCOeval(self.coco, cocoDt, "segm")
            cocoEval.evaluate()
            cocoEval.accumulate()
            cocoEval.summarize()
            for k in range(6):
                ret["mAP(segm)/" + fields[k]] = cocoEval.stats[k]
        return ret

    def load(self, add_gt=True, add_mask=False):
        """
        Args:
            add_gt: whether to add ground truth bounding box annotations to the dicts
            add_mask: whether to also add ground truth mask

        Returns:
            a list of dict, each has keys including:
                'image_id', 'file_name',
                and (if add_gt is True) 'boxes', 'class', 'is_crowd', and optionally
                'segmentation'.
        """
        if add_mask:
            assert add_gt
        with timed_operation(
            "Load Groundtruth Boxes for {}".format(os.path.basename(self.annotation_file))
        ):
            img_ids = self.coco.getImgIds()
            img_ids.sort()
            # list of dict, each has keys: height,width,id,file_name
            imgs = self.coco.loadImgs(img_ids)

            for img in tqdm.tqdm(imgs):
                img["image_id"] = img.pop("id")
                self._use_absolute_file_name(img)
                if add_gt:
                    self._add_detection_gt(img, add_mask)
            return imgs

    def _use_absolute_file_name(self, img):
        """
        Change relative filename to abosolute file name.
        """
        if self.is_aws or self.is_gcs:
            img["file_name"] = "{}/{}".format(self._imgdir, img["file_name"])
        else:
            img["file_name"] = os.path.join(self._imgdir, img["file_name"])
            assert os.path.isfile(img["file_name"]), img["file_name"]

    def _add_detection_gt(self, img, add_mask):
        """
        Add 'boxes', 'class', 'is_crowd' of this image to the dict, used by detection.
        If add_mask is True, also add 'segmentation' in coco poly format.
        """
        # ann_ids = self.coco.getAnnIds(imgIds=img['image_id'])
        # objs = self.coco.loadAnns(ann_ids)
        objs = self.coco.imgToAnns[
            img["image_id"]
        ]  # equivalent but faster than the above two lines
        if "minival" not in self.annotation_file:
            # TODO better to check across the entire json, rather than per-image
            ann_ids = [ann["id"] for ann in objs]
            assert len(set(ann_ids)) == len(
                ann_ids
            ), "Annotation ids in '{}' are not unique!".format(self.annotation_file)

        # clean-up boxes
        valid_objs = []
        width = img.pop("width")
        height = img.pop("height")
        for objid, obj in enumerate(objs):
            if obj.get("ignore", 0) == 1:
                continue
            x1, y1, w, h = obj["bbox"]
            # bbox is originally in float
            # x1/y1 means upper-left corner and w/h means true w/h. This can be verified by segmentation pixels.
            # But we do make an assumption here that (0.0, 0.0) is upper-left corner of the first pixel

            x1 = np.clip(float(x1), 0, width)
            y1 = np.clip(float(y1), 0, height)
            w = np.clip(float(x1 + w), 0, width) - x1
            h = np.clip(float(y1 + h), 0, height) - y1
            # Require non-zero seg area and more than 1x1 box size
            if obj["area"] > 1 and w > 0 and h > 0 and w * h >= 4:
                obj["bbox"] = [x1, y1, x1 + w, y1 + h]
                valid_objs.append(obj)

                if add_mask:
                    segs = obj["segmentation"]
                    if not isinstance(segs, list):
                        assert obj["iscrowd"] == 1
                        obj["segmentation"] = None
                    else:
                        valid_segs = [
                            np.asarray(p).reshape(-1, 2).astype("float32")
                            for p in segs
                            if len(p) >= 6
                        ]
                        if len(valid_segs) == 0:
                            logger.error(
                                "Object {} in image {} has no valid polygons!".format(
                                    objid, img["file_name"]
                                )
                            )
                        elif len(valid_segs) < len(segs):
                            logger.warn(
                                "Object {} in image {} has invalid polygons!".format(
                                    objid, img["file_name"]
                                )
                            )

                        obj["segmentation"] = valid_segs

        # all geometrically-valid boxes are returned
        boxes = np.asarray([obj["bbox"] for obj in valid_objs], dtype="float32")  # (n, 4)
        cls = np.asarray(
            [
                self.COCO_id_to_category_id.get(obj["category_id"], obj["category_id"])
                for obj in valid_objs
            ],
            dtype="int32",
        )  # (n,)
        is_crowd = np.asarray([obj["iscrowd"] for obj in valid_objs], dtype="int8")

        # add the keys
        img["boxes"] = boxes  # nx4
        if len(cls):
            assert cls.min() > 0, "Category id in COCO format must > 0!"
        img["class"] = cls  # n, always >0
        img["is_crowd"] = is_crowd  # n,
        if add_mask:
            # also required to be float32
            img["segmentation"] = [obj["segmentation"] for obj in valid_objs]

    def training_roidbs(self):
        return self.load(add_gt=True, add_mask=cfg.MODE_MASK)

    def inference_roidbs(self):
        return self.load(add_gt=False)

    def eval_inference_results(self, results, output):
        continuous_id_to_COCO_id = {v: k for k, v in self.COCO_id_to_category_id.items()}
        for res in results:
            # convert to COCO's incontinuous category id
            if res["category_id"] in continuous_id_to_COCO_id:
                res["category_id"] = continuous_id_to_COCO_id[res["category_id"]]
            # COCO expects results in xywh format
            box = res["bbox"]
            box[2] -= box[0]
            box[3] -= box[1]
            res["bbox"] = [round(float(x), 3) for x in box]

        assert output is not None, "COCO evaluation requires an output file!"
        with open(output, "w") as f:
            json.dump(results, f)
        if len(results):
            # sometimes may crash if the results are empty?
            return self.print_coco_metrics(output)
        else:
            return {}


def register_coco(basedir, is_aws, is_gcs):
    """
    Add COCO datasets like "coco_train201x" to the registry,
    so you can refer to them with names in `cfg.DATA.TRAIN/VAL`.
    """
    for split in [
        "train2017",
        "val2017",
        "train2014",
        "val2014",
        "valminusminival2014",
        "minival2014",
    ]:
        DatasetRegistry.register(
            "coco_" + split, lambda x=split: COCODetection(basedir, x, is_aws, is_gcs),
        )


if __name__ == "__main__":
    basedir = "~/data/coco"
    c = COCODetection(basedir, "train2014", True)
    roidb = c.load(add_gt=True, add_mask=True)
    print("#Images:", len(roidb))
