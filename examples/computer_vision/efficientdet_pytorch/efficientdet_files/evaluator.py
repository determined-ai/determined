"""
This is based on efficientdet's evaluator.py https://github.com/rwightman/efficientdet-pytorch/blob/678bae1597eb083e05b033ee3eb585877282279a/effdet/evaluator.py

This altered version removes the required distributed code because Determined's custom reducer will handle all distributed training.
"""

import abc
import json
import logging
import time

# FIXME experimenting with speedups for OpenImages eval, it's slow
# import pyximport; py_importer, pyx_importer = pyximport.install(pyimport=True)
import effdet.evaluation.detection_evaluator as tfm_eval
import numpy as np
import torch
from pycocotools.cocoeval import COCOeval

from .utils import FakeParser

# pyximport.uninstall(py_importer, pyx_importer)


class Evaluator:
    def __init__(self, pred_yxyx=False):
        self.pred_yxyx = pred_yxyx
        self.img_indices = []
        self.predictions = []

    def add_predictions(self, detections, target):
        img_indices = target["img_idx"]

        detections = detections.cpu().numpy()
        img_indices = img_indices.cpu().numpy()
        for img_idx, img_dets in zip(img_indices, detections):
            self.img_indices.append(img_idx)
            self.predictions.append(img_dets)

    def _coco_predictions(self):
        # generate coco-style predictions
        coco_predictions = []
        coco_ids = []
        for img_idx, img_dets in zip(self.img_indices, self.predictions):
            img_id = self._dataset.img_ids[img_idx]
            coco_ids.append(img_id)
            if self.pred_yxyx:
                # to xyxy
                img_dets[:, 0:4] = img_dets[:, [1, 0, 3, 2]]
            # to xywh
            img_dets[:, 2] -= img_dets[:, 0]
            img_dets[:, 3] -= img_dets[:, 1]
            for det in img_dets:
                score = float(det[4])
                if score < 0.001:  # stop when below this threshold, scores in descending order
                    break
                coco_det = dict(
                    image_id=int(img_id),
                    bbox=det[0:4].tolist(),
                    score=score,
                    category_id=int(det[5]),
                )
                coco_predictions.append(coco_det)
        return coco_predictions, coco_ids

    @abc.abstractmethod
    def evaluate(self):
        pass


class CocoEvaluator(Evaluator):
    def __init__(self, dataset, pred_yxyx=False):
        super().__init__(pred_yxyx=pred_yxyx)
        self._dataset = dataset.parser
        self.coco_api = dataset.parser.coco

    def reset(self):
        self.img_indices = []
        self.predictions = []

    def evaluate(self):
        coco_predictions, coco_ids = self._coco_predictions()
        json.dump(coco_predictions, open("./temp.json", "w"), indent=4)
        results = self.coco_api.loadRes("./temp.json")
        coco_eval = COCOeval(self.coco_api, results, "bbox")
        coco_eval.params.imgIds = coco_ids  # score only ids we've used
        coco_eval.evaluate()
        coco_eval.accumulate()
        coco_eval.summarize()
        metric = coco_eval.stats[0]  # mAP 0.5-0.95
        print("metric res: ", metric)
        return metric


class TfmEvaluator(Evaluator):
    """TensorFlow Models Evaluator Wrapper"""

    def __init__(
        self,
        dataset,
        distributed=False,
        pred_yxyx=False,
        evaluator_cls=tfm_eval.ObjectDetectionEvaluator,
    ):
        super().__init__(pred_yxyx=pred_yxyx)
        self._evaluator = evaluator_cls(categories=dataset.parser.cat_dicts)
        self._eval_metric_name = self._evaluator._metric_names[0]
        self._dataset = dataset.parser

    def reset(self):
        self._evaluator.clear()
        self.img_indices = []
        self.predictions = []

    def evaluate(self):
        for img_idx, img_dets in zip(self.img_indices, self.predictions):
            gt = self._dataset.get_ann_info(img_idx)
            self._evaluator.add_single_ground_truth_image_info(img_idx, gt)

            bbox = img_dets[:, 0:4] if self.pred_yxyx else img_dets[:, [1, 0, 3, 2]]
            det = dict(bbox=bbox, score=img_dets[:, 4], cls=img_dets[:, 5])
            self._evaluator.add_single_detected_image_info(img_idx, det)

        metrics = self._evaluator.evaluate()
        map_metric = metrics[self._eval_metric_name]
        self.reset()
        return map_metric


class PascalEvaluator(TfmEvaluator):
    def __init__(self, dataset, pred_yxyx=False):
        super().__init__(
            dataset, pred_yxyx=pred_yxyx, evaluator_cls=tfm_eval.PascalDetectionEvaluator
        )


class OpenImagesEvaluator(TfmEvaluator):
    def __init__(self, dataset, pred_yxyx=False):
        super().__init__(
            dataset, pred_yxyx=pred_yxyx, evaluator_cls=tfm_eval.OpenImagesDetectionEvaluator
        )


class FakeEvaluator(Evaluator):
    def __init__(self, dataset, pred_yxyx=False):
        super().__init__(pred_yxyx=pred_yxyx)
        self._dataset = FakeParser()

    def reset(self):
        self.img_indices = []
        self.predictions = []

    def evaluate(self):
        self._dataset.create_fake_img_ids(len(self.img_indices))
        # Confirm coco_preds still works.
        # Skip the rest in CocoEvaluator because there isn't an ann_file
        coco_predictions, coco_ids = self._coco_predictions()

        return 0.001  # return fake metric value


def create_evaluator(name, dataset, pred_yxyx=False, context=None):
    # FIXME support OpenImages Challenge2019 metric w/ image level label consideration
    if context.get_hparam("fake_data"):
        return FakeEvaluator(dataset, pred_yxyx=pred_yxyx)
    elif "coco" in name:
        return CocoEvaluator(dataset, pred_yxyx=pred_yxyx)
    elif "openimages" in name:
        return OpenImagesEvaluator(dataset, pred_yxyx=pred_yxyx)
    else:
        return PascalEvaluator(dataset, pred_yxyx=pred_yxyx)
