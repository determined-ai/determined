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
from determined.pytorch import DataLoader, LRScheduler, PyTorchTrial, PyTorchTrialContext, reset_parameters
from model import NetworkImageNet
from utils import AutoAugment, CrossEntropyLabelSmooth, Cutout, HSwish, Swish, accuracy

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

        self.model = self.context.Model(NetworkImageNet(
            genotype,
            activation_function,
            self.context.get_hparam("init_channels"),
            self.context.get_hparam("num_classes"),
            self.context.get_hparam("layers"),
            auxiliary=self.context.get_hparam("auxiliary"),
            do_SE=self.context.get_hparam("do_SE"),
        ))

        # If loading backbone weights, do not call reset_parameters() or
        # call before loading the backbone weights.
        reset_parameters(self.model)

        self.optimizer = self.context.Optimizer(torch.optim.SGD(
            self.model.parameters(),
            lr=self.context.get_hparam("learning_rate"),
            momentum=self.context.get_hparam("momentum"),
            weight_decay=self.context.get_hparam("weight_decay"),
        ))

        self.lr_scheduler = self.context.LRScheduler(
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
        if self.context.get_hparam("autoaugment"):
            train_transforms.transforms.insert(0, AutoAugment)

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
        self, batch: Any, model: nn.Module, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:

        if batch_idx == 0 or self.last_epoch_idx < epoch_idx:
            self.lr_scheduler.step()
            current_lr = self.lr_scheduler.get_last_lr()[0]

            if epoch_idx < 5:
                lr = self.context.get_hparam("learning_rate")
                for param_group in self.optimizer.param_groups:
                    param_group["lr"] = lr * (epoch_idx + 1) / 5.0
                print(
                    "Warming-up Epoch: {}, LR: {}".format(
                        epoch_idx, lr * (epoch_idx + 1) / 5.0
                    )
                )
            else:
                print("Epoch: {} lr {}".format(epoch_idx, current_lr))

        input, target = batch
        self.model.drop_path_prob = 0

        logits, logits_aux = self.model(input)
        loss = self.criterion(logits, target)
        if self.context.get_hparam("auxiliary"):
            loss_aux = self.criterion(logits_aux, target)
            loss += self.context.get_hparam("auxiliary_weight") * loss_aux
        top1, top5 = accuracy(logits, target, topk=(1, 5))
        self.last_epoch_idx = epoch_idx

        return {"loss": loss, "top1_accuracy": top1, "top5_accuracy": top5}

    def evaluate_batch(self, batch: Any, model: nn.Module) -> Dict[str, Any]:
        input, target = batch
        logits, _ = self.model(input)
        loss = self.criterion(logits, target)
        top1, top5 = accuracy(logits, target, topk=(1, 5))

        return {"loss": loss, "top1_accuracy": top1, "top5_accuracy": top5}
