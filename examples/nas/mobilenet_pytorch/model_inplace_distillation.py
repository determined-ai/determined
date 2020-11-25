"""
"""

from collections import namedtuple
from typing import Any, Dict

import torch
import torchvision.transforms as transforms
import torch.nn.functional as F

from data import ImageNetDataset
from determined.pytorch import (
    DataLoader,
    LRScheduler,
    PyTorchTrial,
    PyTorchTrialContext,
    PyTorchCallback,
    ClipGradsL2Norm,
)

from ofa.elastic_nn.networks import OFAMobileNetV3
from ofa.utils import download_url
from ofa.imagenet_codebase.utils import cross_entropy_loss_with_soft_target

from utils import (
    RandAugment,
    CrossEntropyLabelSmooth,
    Cutout,
    accuracy,
)
from lr_schedulers import *


class OFATrial(PyTorchTrial):
    def __init__(self, context: PyTorchTrialContext) -> None:
        self.context = context
        self.data_config = context.get_data_config()
        self.hparams = context.get_hparams()
        self.criterion = CrossEntropyLabelSmooth(
            context.get_hparam("n_classes"),  # num classes
            context.get_hparam("label_smoothing_rate"),
        )
        self.last_epoch_idx = -1

        self.supernet = self.context.wrap_model(
            OFAMobileNetV3(
                n_classes=self.hparams["n_classes"],
                bn_param=(self.hparams["bn_momentum"], self.hparams["bn_eps"]),
                dropout_rate=self.hparams["dropout"],
                base_stage_width="proxyless",
                width_mult_list=[1.0],
                ks_list=[3, 5, 7],
                expand_ratio_list=[3, 4, 6],
                depth_list=[2, 3, 4],
            )
        )

        # Configure supernet according to architecture config.
        self.teacher_arch = {
            'kernel_sizes': []
            'depths': []
            'widths': []
        }
        self.student_arch = {
            'kernel_sizes': []
            'depths': []
            'widths': []
        }
        for module, n_blocks in enumerate([4] * 5):
            self.student_arch['depths'].append(
                self.hparams["b{}_depth".format(module + 1)]
            )
            self.teacher_arch['depths'].append(4)
            for block in range(n_blocks):
                self.student_arch['kernel_sizes'].append(
                    self.hparams["b{}{}_ks".format(module + 1, block + 1)]
                )
                self.teacher_arch['kernel_sizes'].append(7)
                self.student_arch['widths'].append(
                    self.hparams["b{}{}_expand".format(module + 1, block + 1)]
                )
                self.teacher_arch['widths'].append(6)

        print('kernel_sizes', self.kernel_sizes)
        print('depths', self.depths)
        print('widths', self.widths)
        print('block_group_info', self.supernet.block_group_info)
        print("n_blocks", len(self.supernet.blocks))

        for n in self.context.models:
            n.init_model(self.hparams["init_policy"])

        self.supernet.set_active_subnet(
            1.0, 
            self.student_arch['kernel_sizes'], 
            self.student_arch['widths'], 
            self.student_arch['depths']
        )

        self.optimizer = self.context.wrap_optimizer(
            torch.optim.SGD(
                self.supernet.parameters(),
                lr=self.context.get_hparam("learning_rate"),
                momentum=self.context.get_hparam("momentum"),
                weight_decay=self.context.get_hparam("weight_decay"),
            )
        )

        self.lr_scheduler = self.context.wrap_lr_scheduler(
            self.build_lr_scheduler_from_config(self.optimizer),
            step_mode=LRScheduler.StepMode.STEP_EVERY_EPOCH,
        )

    def build_lr_scheduler_from_config(self, optimizer):
        if self.context.get_hparam("lr_scheduler") == "cosine":
            scheduler_cls = WarmupWrapper(torch.optim.lr_scheduler.CosineAnnealingLR)
            scheduler = scheduler_cls(
                self.context.get_hparam("warmup_epochs"),
                optimizer,
                self.context.get_hparam("lr_epochs"),
            )
        elif self.context.get_hparam("lr_scheduler") == "linear":
            scheduler_cls = WarmupWrapper(LinearLRScheduler)
            scheduler = scheduler_cls(
                self.context.get_hparam("warmup_epochs"),
                optimizer,
                self.context.get_hparam("lr_epochs"),
                self.context.get_hparam("warmup_epochs"),
            )
        else:
            raise NotImplementedError
        return scheduler

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

    def train_batch(
        self, batch: Any, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:

        if batch_idx == 0 or self.last_epoch_idx < epoch_idx:
            current_lr = self.lr_scheduler.get_last_lr()[0]
            print("Epoch: {} lr {}".format(epoch_idx, current_lr))
        self.last_epoch_idx = epoch_idx

        images, target = batch

        self.supernet.set_active_subnet(
            1.0, 
            self.teacher_arch['kernel_sizes'], 
            self.teacher_arch['widths'], 
            self.teacher_arch['depths']
        )
        logits = self.supernet(images)
        loss = self.criterion(logits, target)
        self.context.backward(loss)
        self.context.step_optimizer(self.optimizer)
        soft_label = F.softmax(logits, dim=1).detach()
        self.supernet.set_active_subnet(
            1.0, 
            self.student_arch['kernel_sizes'], 
            self.student_arch['widths'], 
            self.student_arch['depths']
        )
        logits = self.supernet(images)
        kd_loss = cross_entropy_loss_with_soft_target(logits, soft_label)
        self.context.backward(kd_loss)
        self.context.step_optimizer(self.optimizer)

        top1, top5 = accuracy(logits, target, topk=(1, 5))

        return {"loss": loss, "top1_accuracy": top1, "top5_accuracy": top5}

    def evaluate_batch(self, batch: Any) -> Dict[str, Any]:
        input, target = batch
        logits = self.supernet(input)
        loss = self.criterion(logits, target)
        top1, top5 = accuracy(logits, target, topk=(1, 5))

        return {
            "loss": loss,
            "top1_accuracy": top1,
            "top5_accuracy": top5,
        }
