"""
"""
import copy
from collections import defaultdict
from typing import Any, Dict, Sequence, Union
from attrdict import AttrDict
import sys
import os
import numpy as np

sys.path.insert(0, '/')
sys.path.insert(0, '/detr')

import torch

# DETR imports
import detr.util.misc as utils
from detr.datasets import build_dataset, get_coco_api_from_dataset
from detr.datasets.coco_eval import CocoEvaluator, create_common_coco_eval
from detr.models import build_model

from data import unwrap_collate_fn
from determined.pytorch import (
    DataLoader,
    LRScheduler,
    PyTorchTrial,
    PyTorchTrialContext,
)


TorchData = Union[Dict[str, torch.Tensor], Sequence[torch.Tensor], torch.Tensor]


class DETRTrial(PyTorchTrial):
    def __init__(self, context: PyTorchTrialContext) -> None:
        self.context = context
        self.hparams = AttrDict(self.context.get_hparams())

        # Wrap the model.
        model, self.criterion, self.postprocessors = build_model(self.hparams)
        self.model = self.context.wrap_model(model)

        n_parameters = sum(p.numel() for p in model.parameters() if p.requires_grad)
        print("number of params:", n_parameters)

        param_dicts = [
            {
                "params": [
                    p
                    for n, p in self.model.named_parameters()
                    if "backbone" not in n and p.requires_grad
                ]
            },
            {
                "params": [
                    p
                    for n, p in self.model.named_parameters()
                    if "backbone" in n and p.requires_grad
                ],
                "lr": self.hparams.lr_backbone,
            },
        ]
        self.optimizer = self.context.wrap_optimizer(
            torch.optim.AdamW(
                param_dicts, lr=self.hparams.lr, weight_decay=self.hparams.weight_decay
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

        # Build datasets
        self.dataset_train = build_dataset(image_set="train", args=self.hparams)
        self.dataset_val = build_dataset(image_set="val", args=self.hparams)

        # Set up evaluator
        self.base_ds = get_coco_api_from_dataset(self.dataset_val)
        self.iou_types = tuple(
            k for k in ("segm", "bbox") if k in self.postprocessors.keys()
        )

    def build_training_data_loader(self) -> DataLoader:
        return DataLoader(
            self.dataset_train,
            batch_size=self.context.get_per_slot_batch_size(),
            collate_fn=unwrap_collate_fn,
            num_workers=self.hparams.num_workers,
        )

    def build_validation_data_loader(self) -> DataLoader:
        return DataLoader(
            self.dataset_val,
            batch_size=self.context.get_per_slot_batch_size(),
            collate_fn=unwrap_collate_fn,
            num_workers=self.hparams.num_workers,
        )

    def train_batch(
        self, batch: TorchData, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        samples, targets = batch
        samples = utils.NestedTensor(samples["tensors"], samples["mask"])
        outputs = self.model(samples)
        loss_dict = self.criterion(outputs, targets)
        weight_dict = self.criterion.weight_dict
        losses = sum(
            loss_dict[k] * weight_dict[k] for k in loss_dict.keys() if k in weight_dict
        )
        self.context.backward(losses)
        self.context.step_optimizer(self.optimizer)

        # Compute losses for logging
        loss_dict_scaled = {
            f"{k}_scaled": v * weight_dict[k]
            for k, v in loss_dict.items()
            if k in weight_dict
        }
        loss_dict["sum_unscaled"] = sum(loss_dict.values())
        loss_dict["sum_scaled"] = sum(loss_dict_scaled.values())
        loss_dict.update(loss_dict_scaled)

        loss_dict["loss"] = losses

        return loss_dict

    def evaluate_full_dataset(
        self, data_loader: torch.utils.data.DataLoader
    ) -> Dict[str, Any]:
        # This is slow, need to have custom reducer to suppport multi-GPU eval.
        coco_evaluator = CocoEvaluator(self.base_ds, self.iou_types)
        results = {}
        loss_dict_aggregated = defaultdict(int)
        for i, batch in enumerate(data_loader):
            if i < 10:
                samples, targets = self.context.to_device(batch)
                samples = utils.NestedTensor(samples["tensors"], samples["mask"])

                outputs = self.model(samples)
                loss_dict = self.criterion(outputs, targets)
                weight_dict = self.criterion.weight_dict

                # Compute losses for logging
                loss_dict_scaled = {
                    f"{k}_scaled": v * weight_dict[k]
                    for k, v in loss_dict.items()
                    if k in weight_dict
                }
                loss_dict["sum_unscaled"] = sum(loss_dict.values())
                loss_dict["sum_scaled"] = sum(loss_dict_scaled.values())
                loss_dict.update(loss_dict_scaled)

                for k in loss_dict:
                    loss_dict_aggregated[k] += loss_dict[k]

                orig_target_sizes = torch.stack([t["orig_size"] for t in targets], dim=0)
                res = self.postprocessors["bbox"](outputs, orig_target_sizes)
                results.update({
                    target["image_id"].item(): output
                    for target, output in zip(targets, res)
                })

        for k in loss_dict_aggregated:
            loss_dict_aggregated[k] /= i + 1

        coco_evaluator.update(results)
        for iou_type in coco_evaluator.iou_types:
            coco_eval = coco_evaluator.coco_eval[iou_type]
            coco_evaluator.eval_imgs[iou_type] = np.concatenate(coco_evaluator.eval_imgs[iou_type], 2)
            create_common_coco_eval(coco_eval, coco_evaluator.img_ids, coco_evaluator.eval_imgs[iou_type])
        coco_evaluator.accumulate()
        coco_evaluator.summarize()

        coco_stats = coco_evaluator.coco_eval["bbox"].stats.tolist()

        loss_dict_aggregated["mAP_50"] = coco_stats[0]
        loss_dict_aggregated["mAP_75"] = coco_stats[1]
        loss_dict_aggregated["mAP_small"] = coco_stats[2]
        loss_dict_aggregated["mAP_medium"] = coco_stats[3]
        loss_dict_aggregated["mAP_large"] = coco_stats[4]
        return loss_dict_aggregated

if __name__=="__main__":
    context = PyTorchTrialContext.from_config('./const.yaml')
    trial = DETRTrial(context)
    val_dataloader = trial.build_validation_data_loader().get_data_loader()
    trial.evaluate_full_dataset(val_dataloader)

