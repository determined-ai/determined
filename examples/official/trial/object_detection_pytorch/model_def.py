"""
This is an object detection finetuning example.  We finetune a Faster R-CNN
model pretrained on COCO to detect pedestrians in the relatively small PennFudan
dataset.

Useful References:
    https://docs.determined.ai/latest/reference/api/pytorch.html
    https://www.cis.upenn.edu/~jshi/ped_html/

Based on: https://pytorch.org/tutorials/intermediate/torchvision_tutorial.html
"""
import copy
from typing import Any, Dict, Sequence, Union

import torch
import torchvision
from torch import nn
from torchvision.models.detection import fasterrcnn_resnet50_fpn
from torchvision.models.detection.faster_rcnn import FastRCNNPredictor

from determined.pytorch import DataLoader, LRScheduler, PyTorchTrial, PyTorchTrialContext

from data import download_data, get_transform, collate_fn, PennFudanDataset

TorchData = Union[Dict[str, torch.Tensor], Sequence[torch.Tensor], torch.Tensor]


class ObjectDetectionTrial(PyTorchTrial):
    def __init__(self, context: PyTorchTrialContext) -> None:
        self.context = context

        # Create a unique download directory for each rank so they don't
        # overwrite each other.
        self.download_directory = f"/tmp/data-rank{self.context.distributed.get_rank()}"
        download_data(
            download_directory=self.download_directory, data_config=self.context.get_data_config(),
        )

        dataset = PennFudanDataset(self.download_directory + "/PennFudanPed", get_transform())

        # Split 80/20 into training and validation datasets.
        train_size = int(0.8 * len(dataset))
        test_size = len(dataset) - train_size
        self.dataset_train, self.dataset_val = torch.utils.data.random_split(
            dataset, [train_size, test_size]
        )

        self.model = self.context.Model(fasterrcnn_resnet50_fpn(pretrained=True))

        # Replace the classifier with a new two-class classifier.  There are
        # only two "classes": pedestrian and background.
        num_classes = 2
        in_features = self.model.roi_heads.box_predictor.cls_score.in_features
        self.model.roi_heads.box_predictor = FastRCNNPredictor(in_features, num_classes)

        self.optimizer = self.context.Optimizer(torch.optim.SGD(
            self.model.parameters(),
            lr=self.context.get_hparam("learning_rate"),
            momentum=self.context.get_hparam("momentum"),
            weight_decay=self.context.get_hparam("weight_decay"),
        ))

        self.lr_scheduler = self.context.LRScheduler(
            torch.optim.lr_scheduler.StepLR(self.optimizer, step_size=3, gamma=0.1),
            step_mode=LRScheduler.StepMode.STEP_EVERY_EPOCH
        )

    def build_training_data_loader(self) -> DataLoader:
        return DataLoader(
            self.dataset_train,
            batch_size=self.context.get_per_slot_batch_size(),
            collate_fn=collate_fn,
        )

    def build_validation_data_loader(self) -> DataLoader:
        return DataLoader(
            self.dataset_val,
            batch_size=self.context.get_per_slot_batch_size(),
            collate_fn=collate_fn,
        )

    def train_batch(
        self, batch: TorchData, model: nn.Module, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        images, targets = batch
        loss_dict = self.model(list(images), list(targets))

        self.context.backward(loss_dict["loss_box_reg"])
        self.context.step_optimizer(self.optimizer)

        return {"loss": loss_dict["loss_box_reg"]}

    def evaluate_batch(self, batch: TorchData, model: nn.Module) -> Dict[str, Any]:
        images, targets = batch
        output = self.model(list(images), copy.deepcopy(list(targets)))
        sum_iou = 0
        num_boxes = 0

        # Our eval metric is the average best IoU (across all predicted
        # pedestrian bounding boxes) per target pedestrian.  Given predicted
        # and target bounding boxes, IoU is the area of the intersection over
        # the area of the union.
        for idx, target in enumerate(targets):
            # Filter out overlapping bounding box predictions based on
            # non-maximum suppression (NMS)
            predicted_boxes = output[idx]["boxes"]
            prediction_scores = output[idx]["scores"]
            keep_indices = torchvision.ops.nms(predicted_boxes, prediction_scores, 0.1)
            predicted_boxes = torch.index_select(predicted_boxes, 0, keep_indices)

            # Tally IoU with respect to the ground truth target boxes
            target_boxes = target["boxes"]
            boxes_iou = torchvision.ops.box_iou(target_boxes, predicted_boxes)
            sum_iou += sum(max(iou_result) for iou_result in boxes_iou)
            num_boxes += len(target_boxes)

        return {"val_avg_iou": sum_iou / num_boxes}
