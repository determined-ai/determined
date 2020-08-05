"""
This example uses the distributed training aspect of Determined
to quickly and efficiently train a state-of-the-art architecture
for ImageNet found by a leading NAS method called GAEA:
https://arxiv.org/abs/2004.07802

We will add swish activation and squeeze-and-excite modules in this
model to further improve upon the published 24.0 test error on imagenet.

We assume that you already have imagenet downloaded and the train and test
directories set up.
"""

from collections import namedtuple
from typing import Any, Dict

import torch
import torchvision.transforms as transforms
from torch import nn

from data import ImageNetDataset
from determined.pytorch import (
    DataLoader,
    LRScheduler,
    PyTorchTrial,
    reset_parameters,
    PyTorchCallback,
    ClipGradsL2Norm,
)
from model import NetworkImageNet
from utils import (
    RandAugment,
    CrossEntropyLabelSmooth,
    Cutout,
    HSwish,
    Swish,
    accuracy,
    AvgrageMeter,
    EMAWrapper,
)
from lr_schedulers import *

Genotype = namedtuple("Genotype", "normal normal_concat reduce reduce_concat")

activation_map = {"relu": nn.ReLU, "swish": Swish, "hswish": HSwish}


class ImageNetTrial(PyTorchTrial):
    def __init__(self, context: PyTorchTrialContext) -> None:
        self.context = context
        self.data_config = context.get_data_config()
        self.criterion = CrossEntropyLabelSmooth(
            context.get_hparam("num_classes"),  # num classes
            context.get_hparam("label_smoothing_rate"),
        )
        self.last_epoch_idx = -1

        genotype = Genotype(
            normal=[
                ("skip_connect", 1),
                ("skip_connect", 0),
                ("sep_conv_3x3", 2),
                ("sep_conv_3x3", 1),
                ("sep_conv_5x5", 2),
                ("sep_conv_3x3", 0),
                ("sep_conv_5x5", 3),
                ("sep_conv_5x5", 2),
            ],
            normal_concat=range(2, 6),
            reduce=[
                ("max_pool_3x3", 1),
                ("sep_conv_3x3", 0),
                ("sep_conv_5x5", 1),
                ("dil_conv_5x5", 2),
                ("sep_conv_3x3", 1),
                ("sep_conv_3x3", 3),
                ("sep_conv_5x5", 1),
                ("max_pool_3x3", 2),
            ],
            reduce_concat=range(2, 6),
        )
        activation_function = activation_map[self.context.get_hparam("activation")]

        self.model = self.context.wrap_model(NetworkImageNet(
            genotype,
            activation_function,
            self.context.get_hparam("init_channels"),
            self.context.get_hparam("num_classes"),
            self.context.get_hparam("layers"),
            auxiliary=self.context.get_hparam("auxiliary"),
            do_SE=self.context.get_hparam("do_SE"),
        ))

        self.optimizer = self.context.wrap_optimizer(torch.optim.SGD(
            self.model.parameters(),
            lr=self.context.get_hparam("learning_rate"),
            momentum=self.context.get_hparam("momentum"),
            weight_decay=self.context.get_hparam("weight_decay"),
        ))

        self.lr_scheduler = self.context.wrap_lrscheduler(
            torch.optim.lr_scheduler.CosineAnnealingLR(
                self.optimizer, self.context.get_hparam("cosine_annealing_epochs")
            ),
            step_mode = LRScheduler.StepMode.MANUAL_STEP
        )

    def build_training_data_loader(self) -> DataLoader:
        bucket_name = self.data_config["bucket_name"]
        normalize = transforms.Normalize(
            mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225]
        )
        train_transforms = transforms.Compose(
            [
                transforms.RandomResizedCrop(224),
                transforms.RandomHorizontalFlip(),
                transforms.ColorJitter(
                    brightness=0.4, contrast=0.4, saturation=0.4, hue=0.2
                ),
                transforms.ToTensor(),
                normalize,
            ]
        )
        if self.context.get_hparam("cutout"):
            train_transforms.transforms.append(
                Cutout(self.context.get_hparam("cutout_length"))
            )
        if self.context.get_hparam("randaugment"):
            train_transforms.transforms.insert(0, RandAugment())

        train_data = ImageNetDataset(
            "train",
            bucket_name,
            streaming=self.data_config["streaming"],
            data_download_dir=self.data_config["data_download_dir"],
            transform=train_transforms,
        )

        train_queue = DataLoader(
            train_data,
            batch_size=self.context.get_per_slot_batch_size(),
            shuffle=True,
            pin_memory=True,
            num_workers=self.data_config["num_workers_train"],
        )
        return train_queue

    def build_validation_data_loader(self) -> DataLoader:
        bucket_name = self.data_config["bucket_name"]
        normalize = transforms.Normalize(
            mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225]
        )

        valid_data = ImageNetDataset(
            "validation",
            bucket_name,
            streaming=self.data_config["streaming"],
            data_download_dir=self.data_config["data_download_dir"],
            transform=transforms.Compose(
                [
                    transforms.Resize(256),
                    transforms.CenterCrop(224),
                    transforms.ToTensor(),
                    normalize,
                ]
            ),
        )

        valid_queue = DataLoader(
            valid_data,
            batch_size=self.context.get_per_slot_batch_size(),
            shuffle=False,
            pin_memory=True,
            num_workers=self.data_config["num_workers_val"],
        )
        return valid_queue

    def build_model(self) -> nn.Module:
        genotype = Genotype(
            normal=[
                ("skip_connect", 1),
                ("skip_connect", 0),
                ("sep_conv_3x3", 2),
                ("sep_conv_3x3", 1),
                ("sep_conv_5x5", 2),
                ("sep_conv_3x3", 0),
                ("sep_conv_5x5", 3),
                ("sep_conv_5x5", 2),
            ],
            normal_concat=range(2, 6),
            reduce=[
                ("max_pool_3x3", 1),
                ("sep_conv_3x3", 0),
                ("sep_conv_5x5", 1),
                ("dil_conv_5x5", 2),
                ("sep_conv_3x3", 1),
                ("sep_conv_3x3", 3),
                ("sep_conv_5x5", 1),
                ("max_pool_3x3", 2),
            ],
            reduce_concat=range(2, 6),
        )
        activation_function = activation_map[self.context.get_hparam("activation")]

        model = NetworkImageNet(
            genotype,
            activation_function,
            self.context.get_hparam("init_channels"),
            self.context.get_hparam("num_classes"),
            self.context.get_hparam("layers"),
            auxiliary=self.context.get_hparam("auxiliary"),
            do_SE=self.context.get_hparam("do_SE"),
            drop_path_prob=self.context.get_hparam("drop_path_prob"),
            drop_prob=self.context.get_hparam("drop_prob"),
        )

        ema_model = EMAWrapper(self.context.get_hparam("ema_decay"), model)

        return ema_model

    def optimizer(self, model: nn.Module) -> torch.optim.Optimizer:  # type: ignore
        return torch.optim.SGD(
            model.parameters(),
            lr=self.context.get_hparam("learning_rate"),
            momentum=self.context.get_hparam("momentum"),
            weight_decay=self.context.get_hparam("weight_decay"),
        )

    def create_lr_scheduler(self, optimizer):
        if self.context.get_hparam("lr_scheduler") == "cosine":
            scheduler_cls = WarmupWrapper(torch.optim.lr_scheduler.CosineAnnealingLR)
            self.scheduler = scheduler_cls(
                self.context.get_hparam("warmup_epochs"),
                optimizer,
                self.context.get_hparam("lr_epochs"),
            )
        elif self.context.get_hparam("lr_scheduler") == "linear":
            scheduler_cls = WarmupWrapper(LinearLRScheduler)
            self.scheduler = scheduler_cls(
                self.context.get_hparam("warmup_epochs"),
                optimizer,
                self.context.get_hparam("lr_epochs"),
                self.context.get_hparam("warmup_epochs"),
            )
        elif self.context.get_hparam("lr_scheduler") == "efficientnet":
            scheduler_cls = WarmupWrapper(EfficientNetScheduler)
            self.scheduler = scheduler_cls(
                self.context.get_hparam("warmup_epochs"),
                optimizer,
                self.context.get_hparam("lr_gamma"),
                self.context.get_hparam("lr_decay_every"),
            )
        else:
            raise NotImplementedError
        step_mode = LRScheduler.StepMode.STEP_EVERY_EPOCH
        self._optimizer = self.context.get_optimizer()

        return LRScheduler(self.scheduler, step_mode=step_mode)

    def train_batch(
        self, batch: Any, model: nn.Module, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:

        # Update EMA vars
        model.update_ema()

        if batch_idx == 0 or self.last_epoch_idx < epoch_idx:
            current_lr = self.scheduler.get_last_lr()[0]
            print("Epoch: {} lr {}".format(epoch_idx, current_lr))
        self.last_epoch_idx = epoch_idx

        input, target = batch

        logits, logits_aux = self.model(input)
        loss = self.criterion(logits, target)
        if self.context.get_hparam("auxiliary"):
            loss_aux = self.criterion(logits_aux, target)
            loss += self.context.get_hparam("auxiliary_weight") * loss_aux
        top1, top5 = accuracy(logits, target, topk=(1, 5))

        self.context.backward(loss)
        self.context.step_optimizer(self.optimizer)

        return {"loss": loss, "top1_accuracy": top1, "top5_accuracy": top5}

    def evaluate_batch(self, batch: Any) -> Dict[str, Any]:
        input, target = batch
        logits, _ = self.model(input)
        loss = self.criterion(logits, target)
        top1, top5 = accuracy(logits, target, topk=(1, 5))

        model.restore_ema()

        input, target = batch
        logits, _ = model(input)
        ema_loss = self.criterion(logits, target)
        ema_top1, ema_top5 = accuracy(logits, target, topk=(1, 5))

        model.restore_latest()

        return {"loss": loss, "top1_accuracy": top1, "top5_accuracy": top5,
                "ema_loss": ema_loss, "top1_ema": ema_top1, "top5_ema": ema_top5}

    def build_callbacks(self) -> Dict[str, PyTorchCallback]:
        return {
            "clip_grads": ClipGradsL2Norm(
                self.context.get_hparam("clip_gradients_l2_norm")
            )
        }
