# -*- coding: utf-8 -*-
# File: data.py

import copy
import itertools

import boto3
import numpy as np
from google.cloud import storage
from tabulate import tabulate
from termcolor import colored

import cv2
from common import (
    CustomResize,
    DataFromListOfDict,
    box_to_point8,
    filter_boxes_inside_shape,
    np_iou,
    point8_to_box,
    segmentation_to_mask,
)
from config import config as cfg
from dataset import DatasetRegistry
from determined.util import download_gcs_blob_with_backoff
from tensorpack.dataflow import (
    DataFromList,
    MapData,
    MapDataComponent,
    MultiProcessMapData,
    MultiThreadMapData,
    TestDataSpeed,
    imgaug,
)
from tensorpack.utils import logger
from tensorpack.utils.argtools import log_once, memoized
from utils.np_box_ops import area as np_area
from utils.np_box_ops import ioa as np_ioa

# import tensorpack.utils.viz as tpviz


class MalformedData(BaseException):
    pass


def print_class_histogram(roidbs):
    """
    Args:
        roidbs (list[dict]): the same format as the output of `training_roidbs`.
    """
    # labels are in [1, NUM_CATEGORY], hence +2 for bins
    hist_bins = np.arange(cfg.DATA.NUM_CATEGORY + 2)

    # Histogram of ground-truth objects
    gt_hist = np.zeros((cfg.DATA.NUM_CATEGORY + 1,), dtype=np.int)
    for entry in roidbs:
        # filter crowd?
        gt_inds = np.where((entry["class"] > 0) & (entry["is_crowd"] == 0))[0]
        gt_classes = entry["class"][gt_inds]
        gt_hist += np.histogram(gt_classes, bins=hist_bins)[0]
    data = [[cfg.DATA.CLASS_NAMES[i], v] for i, v in enumerate(gt_hist)]
    data.append(["total", sum(x[1] for x in data)])
    # the first line is BG
    table = tabulate(data[1:], headers=["class", "#box"], tablefmt="pipe")
    logger.info("Ground-Truth Boxes:\n" + colored(table, "cyan"))


@memoized
def get_all_anchors(*, stride, sizes, ratios, max_size):
    """
    Get all anchors in the largest possible image, shifted, floatbox
    Args:
        stride (int): the stride of anchors.
        sizes (tuple[int]): the sizes (sqrt area) of anchors
        ratios (tuple[int]): the aspect ratios of anchors
        max_size (int): maximum size of input image

    Returns:
        anchors: SxSxNUM_ANCHORx4, where S == ceil(MAX_SIZE/STRIDE), floatbox
        The layout in the NUM_ANCHOR dim is NUM_RATIO x NUM_SIZE.

    """
    # Generates a NAx4 matrix of anchor boxes in (x1, y1, x2, y2) format. Anchors
    # are centered on 0, have sqrt areas equal to the specified sizes, and aspect ratios as given.
    anchors = []
    for sz in sizes:
        for ratio in ratios:
            w = np.sqrt(sz * sz / ratio)
            h = ratio * w
            anchors.append([-w, -h, w, h])
    cell_anchors = np.asarray(anchors) * 0.5

    field_size = int(np.ceil(max_size / stride))
    shifts = (np.arange(0, field_size) * stride).astype("float32")
    shift_x, shift_y = np.meshgrid(shifts, shifts)
    shift_x = shift_x.flatten()
    shift_y = shift_y.flatten()
    shifts = np.vstack((shift_x, shift_y, shift_x, shift_y)).transpose()
    # Kx4, K = field_size * field_size
    K = shifts.shape[0]

    A = cell_anchors.shape[0]
    field_of_anchors = cell_anchors.reshape((1, A, 4)) + shifts.reshape((1, K, 4)).transpose(
        (1, 0, 2)
    )
    field_of_anchors = field_of_anchors.reshape((field_size, field_size, A, 4))
    # FSxFSxAx4
    # Many rounding happens inside the anchor code anyway
    # assert np.all(field_of_anchors == field_of_anchors.astype('int32'))
    field_of_anchors = field_of_anchors.astype("float32")
    return field_of_anchors


@memoized
def get_all_anchors_fpn(*, strides, sizes, ratios, max_size):
    """
    Returns:
        [anchors]: each anchors is a SxSx NUM_ANCHOR_RATIOS x4 array.
    """
    assert len(strides) == len(sizes)
    foas = []
    for stride, size in zip(strides, sizes):
        foa = get_all_anchors(stride=stride, sizes=(size,), ratios=ratios, max_size=max_size)
        foas.append(foa)
    return foas


class TrainingDataPreprocessor:
    """
    The mapper to preprocess the input data for training.

    Since the mapping may run in other processes, we write a new class and
    explicitly pass cfg to it, in the spirit of "explicitly pass resources to subprocess".
    """

    def __init__(self, cfg, is_aws, is_gcs):
        self.cfg = cfg
        self.aug = imgaug.AugmentorList(
            [
                CustomResize(cfg.PREPROC.TRAIN_SHORT_EDGE_SIZE, cfg.PREPROC.MAX_SIZE),
                imgaug.Flip(horiz=True),
            ]
        )
        self.is_aws = is_aws
        self.is_gcs = is_gcs
        if self.is_aws:
            self.s3 = boto3.resource("s3")
        elif self.is_gcs:
            self.storage_client = storage.Client.create_anonymous_client()
            self.bucket = self.storage_client.get_bucket("determined-ai-coco-dataset")

    def __call__(self, roidb):
        fname, boxes, klass, is_crowd = (
            roidb["file_name"],
            roidb["boxes"],
            roidb["class"],
            roidb["is_crowd"],
        )
        boxes = np.copy(boxes)
        if self.is_aws:
            s3_object = self.s3.meta.client.get_object(
                Bucket="determined-ai-coco-dataset", Key=fname
            )
            im = cv2.imdecode(
                np.asarray(bytearray(s3_object["Body"].read()), dtype=np.uint8), cv2.IMREAD_COLOR,
            )
        elif self.is_gcs:
            blob = self.bucket.blob(fname)
            s = download_gcs_blob_with_backoff(blob)
            im = cv2.imdecode(np.asarray(bytearray(s), dtype=np.uint8), cv2.IMREAD_COLOR)
        else:
            im = cv2.imread(fname, cv2.IMREAD_COLOR)
        assert im is not None, fname
        im = im.astype("float32")
        height, width = im.shape[:2]
        # assume floatbox as input
        assert boxes.dtype == np.float32, "Loader has to return floating point boxes!"

        if not self.cfg.DATA.ABSOLUTE_COORD:
            boxes[:, 0::2] *= width
            boxes[:, 1::2] *= height

        # augmentation:
        im, params = self.aug.augment_return_params(im)
        points = box_to_point8(boxes)
        points = self.aug.augment_coords(points, params)
        boxes = point8_to_box(points)
        assert np.min(np_area(boxes)) > 0, "Some boxes have zero area!"

        ret = {"image": im}
        # Add rpn data to dataflow:
        try:
            if self.cfg.MODE_FPN:
                multilevel_anchor_inputs = self.get_multilevel_rpn_anchor_input(im, boxes, is_crowd)
                for i, (anchor_labels, anchor_boxes) in enumerate(multilevel_anchor_inputs):
                    ret["anchor_labels_lvl{}".format(i + 2)] = anchor_labels
                    ret["anchor_boxes_lvl{}".format(i + 2)] = anchor_boxes
            else:
                ret["anchor_labels"], ret["anchor_boxes"] = self.get_rpn_anchor_input(
                    im, boxes, is_crowd
                )

            boxes = boxes[is_crowd == 0]  # skip crowd boxes in training target
            klass = klass[is_crowd == 0]
            ret["gt_boxes"] = boxes
            ret["gt_labels"] = klass
            if not len(boxes):
                raise MalformedData("No valid gt_boxes!")
        except MalformedData as e:
            log_once("Input {} is filtered for training: {}".format(fname, str(e)), "warn")
            return None

        if self.cfg.MODE_MASK:
            # augmentation will modify the polys in-place
            segmentation = copy.deepcopy(roidb["segmentation"])
            segmentation = [segmentation[k] for k in range(len(segmentation)) if not is_crowd[k]]
            assert len(segmentation) == len(boxes)

            # Apply augmentation on polygon coordinates.
            # And produce one image-sized binary mask per box.
            masks = []
            width_height = np.asarray([width, height], dtype=np.float32)
            gt_mask_width = int(
                np.ceil(im.shape[1] / 8.0) * 8
            )  # pad to 8 in order to pack mask into bits
            for polys in segmentation:
                if not self.cfg.DATA.ABSOLUTE_COORD:
                    polys = [p * width_height for p in polys]
                polys = [self.aug.augment_coords(p, params) for p in polys]
                masks.append(segmentation_to_mask(polys, im.shape[0], gt_mask_width))
            masks = np.asarray(masks, dtype="uint8")  # values in {0, 1}
            masks = np.packbits(masks, axis=-1)
            ret["gt_masks_packed"] = masks

            # from viz import draw_annotation, draw_mask
            # viz = draw_annotation(im, boxes, klass)
            # for mask in masks:
            #     viz = draw_mask(viz, mask)
            # tpviz.interactive_imshow(viz)
        return ret

    def get_rpn_anchor_input(self, im, boxes, is_crowd):
        """
        Args:
            im: an image
            boxes: nx4, floatbox, gt. shoudn't be changed
            is_crowd: n,

        Returns:
            The anchor labels and target boxes for each pixel in the featuremap.
            fm_labels: fHxfWxNA
            fm_boxes: fHxfWxNAx4
            NA will be NUM_ANCHOR_SIZES x NUM_ANCHOR_RATIOS
        """
        boxes = boxes.copy()
        all_anchors = np.copy(
            get_all_anchors(
                stride=self.cfg.RPN.ANCHOR_STRIDE,
                sizes=self.cfg.RPN.ANCHOR_SIZES,
                ratios=self.cfg.RPN.ANCHOR_RATIOS,
                max_size=self.cfg.PREPROC.MAX_SIZE,
            )
        )
        # fHxfWxAx4 -> (-1, 4)
        featuremap_anchors_flatten = all_anchors.reshape((-1, 4))

        # only use anchors inside the image
        inside_ind, inside_anchors = filter_boxes_inside_shape(
            featuremap_anchors_flatten, im.shape[:2]
        )
        # obtain anchor labels and their corresponding gt boxes
        anchor_labels, anchor_gt_boxes = self.get_anchor_labels(
            inside_anchors, boxes[is_crowd == 0], boxes[is_crowd == 1]
        )

        # Fill them back to original size: fHxfWx1, fHxfWx4
        num_anchor = self.cfg.RPN.NUM_ANCHOR
        anchorH, anchorW = all_anchors.shape[:2]
        featuremap_labels = -np.ones((anchorH * anchorW * num_anchor,), dtype="int32")
        featuremap_labels[inside_ind] = anchor_labels
        featuremap_labels = featuremap_labels.reshape((anchorH, anchorW, num_anchor))
        featuremap_boxes = np.zeros((anchorH * anchorW * num_anchor, 4), dtype="float32")
        featuremap_boxes[inside_ind, :] = anchor_gt_boxes
        featuremap_boxes = featuremap_boxes.reshape((anchorH, anchorW, num_anchor, 4))
        return featuremap_labels, featuremap_boxes

    def get_multilevel_rpn_anchor_input(self, im, boxes, is_crowd):
        """
        Args:
            im: an image
            boxes: nx4, floatbox, gt. shoudn't be changed
            is_crowd: n,

        Returns:
            [(fm_labels, fm_boxes)]: Returns a tuple for each FPN level.
            Each tuple contains the anchor labels and target boxes for each pixel in the featuremap.

            fm_labels: fHxfWx NUM_ANCHOR_RATIOS
            fm_boxes: fHxfWx NUM_ANCHOR_RATIOS x4
        """
        boxes = boxes.copy()
        anchors_per_level = get_all_anchors_fpn(
            strides=self.cfg.FPN.ANCHOR_STRIDES,
            sizes=self.cfg.RPN.ANCHOR_SIZES,
            ratios=self.cfg.RPN.ANCHOR_RATIOS,
            max_size=self.cfg.PREPROC.MAX_SIZE,
        )
        flatten_anchors_per_level = [k.reshape((-1, 4)) for k in anchors_per_level]
        all_anchors_flatten = np.concatenate(flatten_anchors_per_level, axis=0)

        inside_ind, inside_anchors = filter_boxes_inside_shape(all_anchors_flatten, im.shape[:2])
        anchor_labels, anchor_gt_boxes = self.get_anchor_labels(
            inside_anchors, boxes[is_crowd == 0], boxes[is_crowd == 1]
        )

        # map back to all_anchors, then split to each level
        num_all_anchors = all_anchors_flatten.shape[0]
        all_labels = -np.ones((num_all_anchors,), dtype="int32")
        all_labels[inside_ind] = anchor_labels
        all_boxes = np.zeros((num_all_anchors, 4), dtype="float32")
        all_boxes[inside_ind] = anchor_gt_boxes

        start = 0
        multilevel_inputs = []
        for level_anchor in anchors_per_level:
            assert level_anchor.shape[2] == len(self.cfg.RPN.ANCHOR_RATIOS)
            anchor_shape = level_anchor.shape[:3]  # fHxfWxNUM_ANCHOR_RATIOS
            num_anchor_this_level = np.prod(anchor_shape)
            end = start + num_anchor_this_level
            multilevel_inputs.append(
                (
                    all_labels[start:end].reshape(anchor_shape),
                    all_boxes[start:end, :].reshape(anchor_shape + (4,)),
                )
            )
            start = end
        assert end == num_all_anchors, "{} != {}".format(end, num_all_anchors)
        return multilevel_inputs

    def get_anchor_labels(self, anchors, gt_boxes, crowd_boxes):
        """
        Label each anchor as fg/bg/ignore.
        Args:
            anchors: Ax4 float
            gt_boxes: Bx4 float, non-crowd
            crowd_boxes: Cx4 float

        Returns:
            anchor_labels: (A,) int. Each element is {-1, 0, 1}
            anchor_boxes: Ax4. Contains the target gt_box for each anchor when the anchor is fg.
        """
        # This function will modify labels and return the filtered inds
        def filter_box_label(labels, value, max_num):
            curr_inds = np.where(labels == value)[0]
            if len(curr_inds) > max_num:
                disable_inds = np.random.choice(
                    curr_inds, size=(len(curr_inds) - max_num), replace=False
                )
                labels[disable_inds] = -1  # ignore them
                curr_inds = np.where(labels == value)[0]
            return curr_inds

        NA, NB = len(anchors), len(gt_boxes)
        assert NB > 0  # empty images should have been filtered already
        box_ious = np_iou(anchors, gt_boxes)  # NA x NB
        ious_argmax_per_anchor = box_ious.argmax(axis=1)  # NA,
        ious_max_per_anchor = box_ious.max(axis=1)
        ious_max_per_gt = np.amax(box_ious, axis=0, keepdims=True)  # 1xNB
        # for each gt, find all those anchors (including ties) that has the max ious with it
        anchors_with_max_iou_per_gt = np.where(box_ious == ious_max_per_gt)[0]

        # Setting NA labels: 1--fg 0--bg -1--ignore
        anchor_labels = -np.ones((NA,), dtype="int32")  # NA,

        # the order of setting neg/pos labels matter
        anchor_labels[anchors_with_max_iou_per_gt] = 1
        anchor_labels[ious_max_per_anchor >= self.cfg.RPN.POSITIVE_ANCHOR_THRESH] = 1
        anchor_labels[ious_max_per_anchor < self.cfg.RPN.NEGATIVE_ANCHOR_THRESH] = 0

        # label all non-ignore candidate boxes which overlap crowd as ignore
        if crowd_boxes.size > 0:
            cand_inds = np.where(anchor_labels >= 0)[0]
            cand_anchors = anchors[cand_inds]
            ioas = np_ioa(crowd_boxes, cand_anchors)
            overlap_with_crowd = cand_inds[ioas.max(axis=0) > self.cfg.RPN.CROWD_OVERLAP_THRESH]
            anchor_labels[overlap_with_crowd] = -1

        # Subsample fg labels: ignore some fg if fg is too many
        target_num_fg = int(self.cfg.RPN.BATCH_PER_IM * self.cfg.RPN.FG_RATIO)
        fg_inds = filter_box_label(anchor_labels, 1, target_num_fg)
        # Keep an image even if there is no foreground anchors
        # if len(fg_inds) == 0:
        #     raise MalformedData("No valid foreground for RPN!")

        # Subsample bg labels. num_bg is not allowed to be too many
        old_num_bg = np.sum(anchor_labels == 0)
        if old_num_bg == 0:
            # No valid bg in this image, skip.
            raise MalformedData("No valid background for RPN!")
        target_num_bg = self.cfg.RPN.BATCH_PER_IM - len(fg_inds)
        filter_box_label(anchor_labels, 0, target_num_bg)  # ignore return values

        # Set anchor boxes: the best gt_box for each fg anchor
        anchor_boxes = np.zeros((NA, 4), dtype="float32")
        fg_boxes = gt_boxes[ious_argmax_per_anchor[fg_inds], :]
        anchor_boxes[fg_inds, :] = fg_boxes
        # assert len(fg_inds) + np.sum(anchor_labels == 0) == self.cfg.RPN.BATCH_PER_IM
        return anchor_labels, anchor_boxes


def get_train_dataflow(is_aws, is_gcs):
    """
    Return a training dataflow. Each datapoint consists of the following:

    An image: (h, w, 3),

    1 or more pairs of (anchor_labels, anchor_boxes):
    anchor_labels: (h', w', NA)
    anchor_boxes: (h', w', NA, 4)

    gt_boxes: (N, 4)
    gt_labels: (N,)

    If MODE_MASK, gt_masks: (N, h, w)
    """

    roidbs = list(
        itertools.chain.from_iterable(
            DatasetRegistry.get(x).training_roidbs() for x in cfg.DATA.TRAIN
        )
    )
    print_class_histogram(roidbs)

    # Valid training images should have at least one fg box.
    # But this filter shall not be applied for testing.
    num = len(roidbs)
    roidbs = list(filter(lambda img: len(img["boxes"][img["is_crowd"] == 0]) > 0, roidbs))
    logger.info(
        "Filtered {} images which contain no non-crowd groudtruth boxes. Total #images for training: {}".format(
            num - len(roidbs), len(roidbs)
        )
    )

    ds = DataFromList(roidbs, shuffle=True)

    preprocess = TrainingDataPreprocessor(cfg, is_aws, is_gcs)

    if cfg.DATA.NUM_WORKERS > 0:
        if cfg.TRAINER == "horovod":
            buffer_size = (
                cfg.DATA.NUM_WORKERS * 20
            )  # one dataflow for each process, therefore don't need large buffer
            ds = MultiThreadMapData(ds, cfg.DATA.NUM_WORKERS, preprocess, buffer_size=buffer_size)
            # MPI does not like fork()
        else:
            buffer_size = cfg.DATA.NUM_WORKERS * 20
            ds = MultiProcessMapData(ds, cfg.DATA.NUM_WORKERS, preprocess, buffer_size=buffer_size)
    else:
        ds = MapData(ds, preprocess)
    return ds


def get_eval_dataflow(name, is_aws, is_gcs, shard=0, num_shards=1):
    """
    Args:
        name (str): name of the dataset to evaluate
        shard, num_shards: to get subset of evaluation data
    """
    roidbs = DatasetRegistry.get(name).inference_roidbs()
    logger.info("Found {} images for inference.".format(len(roidbs)))

    num_imgs = len(roidbs)
    img_per_shard = num_imgs // num_shards
    img_range = (
        shard * img_per_shard,
        (shard + 1) * img_per_shard if shard + 1 < num_shards else num_imgs,
    )

    # no filter for training
    ds = DataFromListOfDict(roidbs[img_range[0] : img_range[1]], ["file_name", "image_id"])

    if is_aws:
        s3 = boto3.resource("s3")
    elif is_gcs:
        c = storage.Client.create_anonymous_client()
        bucket = c.get_bucket("determined-ai-coco-dataset")

    def f(fname):
        if is_aws:
            s3_object = s3.meta.client.get_object(Bucket="determined-ai-coco-dataset", Key=fname)
            im = cv2.imdecode(
                np.asarray(bytearray(s3_object["Body"].read()), dtype=np.uint8), cv2.IMREAD_COLOR,
            )
        elif is_gcs:
            blob = bucket.blob(fname)
            s = download_gcs_blob_with_backoff(blob)
            im = cv2.imdecode(np.asarray(bytearray(s), dtype=np.uint8), cv2.IMREAD_COLOR)
        else:
            im = cv2.imread(fname, cv2.IMREAD_COLOR)
        assert im is not None, fname
        return im

    ds = MapDataComponent(ds, f, 0)
    # Evaluation itself may be multi-threaded, therefore don't add prefetch here.
    return ds


if __name__ == "__main__":
    import os
    from tensorpack.dataflow import PrintData

    cfg.DATA.BASEDIR = os.path.expanduser("~/data/coco")
    ds = get_train_dataflow(True)
    ds = PrintData(ds, 100)
    TestDataSpeed(ds, 50000).start()
    ds.reset_state()
    for k in ds:
        pass
