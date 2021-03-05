from collections import defaultdict
from typing import Any, Dict, Sequence, Union
from attrdict import AttrDict
import numpy as np
import sys
import os
import copy
import time

sys.path.append("./ddetr")

import torch

from determined.pytorch import (
    DataLoader,
    LRScheduler,
    MetricReducer,
    PyTorchTrial,
    PyTorchTrialContext,
)
from determined.experimental import Determined

# Deformable DETR imports
import ddetr.util.misc as utils
from ddetr.datasets.coco_eval import CocoEvaluator, create_common_coco_eval
from model import build_model

# Experiment dir imports
from data import unwrap_collate_fn, build_dataset
from data_utils import download_coco_from_source


TorchData = Union[Dict[str, torch.Tensor], Sequence[torch.Tensor], torch.Tensor]


def match_name_keywords(n, name_keywords):
    out = False
    for b in name_keywords:
        if b in n:
            out = True
            break
    return out


class COCOReducer(MetricReducer):
    def __init__(self, base_ds, iou_types, cat_ids=[]):
        self.base_ds = base_ds
        self.iou_types = iou_types
        self.cat_ids = cat_ids
        self.reset()

    def reset(self):
        self.results = []

    def update(self, result):
        self.results.extend(result)

    def per_slot_reduce(self):
        return self.results

    def cross_slot_reduce(self, per_slot_metrics):
        coco_evaluator = CocoEvaluator(self.base_ds, self.iou_types)
        if len(self.cat_ids):
            for iou_type in self.iou_types:
                coco_evaluator.coco_eval[iou_type].params.catIds = self.cat_ids
        for results in per_slot_metrics:
            results_dict = {r[0]: r[1] for r in results}
            coco_evaluator.update(results_dict)

        for iou_type in coco_evaluator.iou_types:
            coco_eval = coco_evaluator.coco_eval[iou_type]
            coco_evaluator.eval_imgs[iou_type] = np.concatenate(
                coco_evaluator.eval_imgs[iou_type], 2
            )
            coco_eval.evalImgs = list(coco_evaluator.eval_imgs[iou_type].flatten())
            coco_eval.params.imgIds = list(coco_evaluator.img_ids)
            coco_eval._paramsEval = copy.deepcopy(coco_eval.params)
        coco_evaluator.accumulate()
        coco_evaluator.summarize()

        coco_stats = coco_evaluator.coco_eval["bbox"].stats.tolist()

        loss_dict = {}
        loss_dict["mAP"] = coco_stats[0]
        loss_dict["mAP_50"] = coco_stats[1]
        loss_dict["mAP_75"] = coco_stats[2]
        loss_dict["mAP_small"] = coco_stats[3]
        loss_dict["mAP_medium"] = coco_stats[4]
        loss_dict["mAP_large"] = coco_stats[5]
        return loss_dict


class DeformableDETRTrial(PyTorchTrial):
    def __init__(self, context: PyTorchTrialContext) -> None:
        self.context = context
        self.hparams = AttrDict(self.context.get_hparams())

        # If backend is local download data in rank 0 slot.
        if self.hparams.backend == "local":
            if self.context.distributed.get_local_rank() == 0:
                if not all(
                    [
                        os.path.isdir(os.path.join(self.hparams.data_dir, d))
                        for d in ["train2017", "val2017"]
                    ]
                ):
                    download_coco_from_source(self.hparams.data_dir)
            else:
                # Other slots wait until rank 0 is done downloading, which will
                # correspond to the head writing a done.txt file.
                while not os.path.isfile(
                    os.path.join(self.hparams.data_dir, "done.txt")
                ):
                    time.sleep(10)

        self.cat_ids = []

        # Build the model and configure postprocessors for evaluation.
        model, self.criterion, self.postprocessors = build_model(
            self.hparams, world_size=self.context.distributed.get_size()
        )

        # Load pretrained weights downloaded in the startup-hook.sh from
        # the original repo.
        if "warmstart" in self.hparams and self.hparams.warmstart:
            checkpoint = torch.load("model.ckpt")
            ckpt = checkpoint["model"]
            # Remove class weights if finetuning.
            if "cat_ids" in self.hparams and len(self.hparams.cat_ids):
                delete_keys = [k for k in ckpt if "class_embed" in k]
                for k in delete_keys:
                    del ckpt[k]
            model.load_state_dict(ckpt, strict=False)

        self.model = self.context.wrap_model(model)

        n_parameters = sum(
            p.numel() for p in self.model.parameters() if p.requires_grad
        )
        print("number of params:", n_parameters)
        param_dicts = [
            {
                "params": [
                    p
                    for n, p in self.model.named_parameters()
                    if not match_name_keywords(n, self.hparams.lr_backbone_names)
                    and not match_name_keywords(n, self.hparams.lr_linear_proj_names)
                    and p.requires_grad
                ],
                "lr": self.hparams.lr,
            },
            {
                "params": [
                    p
                    for n, p in self.model.named_parameters()
                    if match_name_keywords(n, self.hparams.lr_backbone_names)
                    and p.requires_grad
                ],
                "lr": self.hparams.lr_backbone,
            },
            {
                "params": [
                    p
                    for n, p in self.model.named_parameters()
                    if match_name_keywords(n, self.hparams.lr_linear_proj_names)
                    and p.requires_grad
                ],
                "lr": self.hparams.lr * self.hparams.lr_linear_proj_mult,
            },
        ]

        if self.hparams.sgd:
            self.optimizer = self.context.wrap_optimizer(
                torch.optim.SGD(
                    param_dicts,
                    lr=self.hparams.lr,
                    momentum=0.9,
                    weight_decay=self.hparams.weight_decay,
                )
            )
        else:
            self.optimizer = self.context.wrap_optimizer(
                torch.optim.AdamW(
                    param_dicts,
                    lr=self.hparams.lr,
                    weight_decay=self.hparams.weight_decay,
                )
            )

        # Wrap the LR scheduler.
        self.lr_scheduler = self.context.wrap_lr_scheduler(
            torch.optim.lr_scheduler.StepLR(self.optimizer, self.hparams.lr_drop),
            step_mode=LRScheduler.StepMode.STEP_EVERY_EPOCH,
        )

        self.clip_grads_fn = (
            lambda x: torch.nn.utils.clip_grad_norm_(x, self.hparams.clip_max_norm)
            if self.hparams.clip_max_norm > 0
            else None
        )

    def build_training_data_loader(self) -> DataLoader:
        dataset_train = build_dataset(image_set="train", args=self.hparams)
        return DataLoader(
            dataset_train,
            batch_size=self.context.get_per_slot_batch_size(),
            collate_fn=unwrap_collate_fn,
            num_workers=self.hparams.num_workers,
            shuffle=True,
        )

    def build_validation_data_loader(self) -> DataLoader:
        dataset_val = build_dataset(image_set="val", args=self.hparams)
        if "cat_ids" in self.hparams:
            self.cat_ids = self.hparams.cat_ids
            self.catIdtoCls = dataset_val.catIdtoCls
        # Set up evaluator
        self.base_ds = dataset_val.coco
        iou_types = tuple(
            k for k in ("segm", "bbox") if k in self.postprocessors.keys()
        )
        self.reducer = self.context.experimental.wrap_reducer(
            COCOReducer(self.base_ds, iou_types, self.cat_ids),
            for_training=False,
            for_validation=True,
        )

        return DataLoader(
            dataset_val,
            batch_size=self.context.get_per_slot_batch_size(),
            collate_fn=unwrap_collate_fn,
            num_workers=1,
            shuffle=False,
        )

    def train_batch(
        self, batch: TorchData, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        samples, targets = batch
        outputs = self.model(samples)
        loss_dict = self.criterion(outputs, targets)
        weight_dict = self.criterion.weight_dict
        losses = sum(
            loss_dict[k] * weight_dict[k] for k in loss_dict.keys() if k in weight_dict
        )
        self.context.backward(losses)
        self.context.step_optimizer(self.optimizer, clip_grads=self.clip_grads_fn)

        # Compute losses for logging
        loss_dict["sum_unscaled"] = sum(loss_dict.values())
        loss_dict["loss"] = losses

        return loss_dict

    def evaluate_batch(self, batch):
        samples, targets = batch

        outputs = self.model(samples)
        loss_dict = self.criterion(outputs, targets, eval=True)

        # Compute losses for logging
        loss_dict["sum_unscaled"] = sum(loss_dict.values())

        orig_target_sizes = torch.stack([t["orig_size"] for t in targets], dim=0)
        res = self.postprocessors["bbox"](outputs, orig_target_sizes)
        res = [{k: v.cpu() for k, v in r.items()} for r in res]
        if len(self.cat_ids):
            for row in res:
                row["labels"] = torch.tensor(
                    [self.cat_ids[l.item()] for l in row["labels"]], dtype=torch.int64
                )
        result = [
            (target["image_id"].item(), output) for target, output in zip(targets, res)
        ]
        self.reducer.update(result)
        return loss_dict
