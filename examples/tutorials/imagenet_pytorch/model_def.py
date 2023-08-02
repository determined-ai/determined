"""
This example shows how to interact with the Determined PyTorch interface.
The original PyTorch script can be found here:
https://github.com/pytorch/examples/tree/master/imagenet

In the `__init__` method, the model and optimizer are wrapped with `wrap_model`
and `wrap_optimizer`. This model is single-input and single-output.

The methods `train_batch` and `evaluate_batch` define the forward pass
for training and evaluation respectively.

"""

import os
import sys
from typing import Any, Dict, Sequence, Tuple, Union, cast

import torch
import torch.nn as nn
import torchvision.datasets as datasets
import torchvision.models as models
import torchvision.transforms as transforms

from determined.pytorch import DataLoader, LRScheduler, PyTorchTrial, PyTorchTrialContext

TorchData = Union[Dict[str, torch.Tensor], Sequence[torch.Tensor], torch.Tensor]


class ImageNetTrial(PyTorchTrial):
    def __init__(self, context: PyTorchTrialContext):
        self.context = context

        arch = self.context.get_hparam("arch")
        if self.context.get_hparam("pretrained"):
            print("=> using pre-trained model '{}'".format(arch))
            model = models.__dict__[arch](pretrained=True)
        else:
            print("=> creating model '{}'".format(arch))
            model = models.__dict__[arch]()

        self.model = self.context.wrap_model(model)

        # Create a unique download directory for each rank so they don't overwrite each
        # other when doing distributed training.
        self.download_directory = f"/tmp/data-rank{self.context.distributed.get_rank()}"
        self.data_directory = self.context.get_hparam("data_location")

        optimizer = torch.optim.SGD(
            self.model.parameters(),
            self.context.get_hparam("lr"),
            momentum=self.context.get_hparam("momentum"),
            weight_decay=self.context.get_hparam("weight_decay"),
        )
        self.optimizer = self.context.wrap_optimizer(optimizer)

        self.criterion = nn.CrossEntropyLoss()

        self.lr_sch = self.context.wrap_lr_scheduler(
            torch.optim.lr_scheduler.StepLR(self.optimizer, gamma=0.1, step_size=2),
            step_mode=LRScheduler.StepMode.STEP_EVERY_EPOCH,
        )

    def build_training_data_loader(self):
        if self.context.get_hparam("dataset") == "imagenet":
            traindir = os.path.join(self.data_directory, "train")
            self.normalize = transforms.Normalize(
                mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225]
            )

            train_dataset = datasets.ImageFolder(
                traindir,
                transforms.Compose(
                    [
                        transforms.RandomResizedCrop(224),
                        transforms.RandomHorizontalFlip(),
                        transforms.ToTensor(),
                        self.normalize,
                    ]
                ),
            )

            return DataLoader(
                train_dataset,
                batch_size=self.context.get_per_slot_batch_size(),
                shuffle=True,
                num_workers=self.context.get_hparam("workers", pin_memory=True),
            )

        elif self.context.get_hparam("dataset") == "cifar":
            transform = transforms.Compose(
                [transforms.ToTensor(), transforms.Normalize((0.5, 0.5, 0.5), (0.5, 0.5, 0.5))]
            )
            trainset = datasets.CIFAR10(
                root=self.download_directory,
                train=True,
                download=self.context.get_hparam("download"),
                transform=transform,
            )
            return DataLoader(trainset, batch_size=self.context.get_per_slot_batch_size())

    def build_validation_data_loader(self):
        if self.context.get_hparam("dataset") == "imagenet":
            valdir = os.path.join(self.download_directory, "val")
            self.normalize = transforms.Normalize(
                mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225]
            )

            val_dataset = datasets.ImageFolder(
                traindir,
                transforms.Compose(
                    [
                        transforms.RandomResizedCrop(224),
                        transforms.RandomHorizontalFlip(),
                        transforms.ToTensor(),
                        self.normalize,
                    ]
                ),
            )

            return DataLoader(
                val_dataset,
                batch_size=self.context.get_per_slot_batch_size(),
                shuffle=False,
                num_workers=self.context.get_hparam("workers", pin_memory=True),
            )
        else:
            transform = transforms.Compose(
                [transforms.ToTensor(), transforms.Normalize((0.5, 0.5, 0.5), (0.5, 0.5, 0.5))]
            )
            val_dataset = datasets.CIFAR10(
                root=self.download_directory, train=False, download=True, transform=transform
            )
            return DataLoader(val_dataset, batch_size=self.context.get_per_slot_batch_size())

    def train_batch(self, batch: TorchData, epoch_idx: int, batch_idx: int):
        images, target = batch
        output = self.model(images)
        loss = self.criterion(output, target)
        acc1, acc5 = self.accuracy(output, target, topk=(1, 5))

        self.context.backward(loss)
        self.context.step_optimizer(self.optimizer)

        return {"loss": loss.item(), "top1": acc1[0], "top5": acc5[0]}

    def evaluate_batch(self, batch: TorchData):
        images, target = batch

        output = self.model(images)
        loss = self.criterion(output, target)

        # measure accuracy and record loss
        acc1, acc5 = self.accuracy(output, target, topk=(1, 5))

        return {"val_loss": loss.item(), "top1": acc1[0], "top5": acc5[0]}

    def accuracy(self, output, target, topk=(1,)):
        """Computes the accuracy over the k top predictions for the specified values of k"""
        with torch.no_grad():
            maxk = max(topk)
            batch_size = target.size(0)

            _, pred = output.topk(maxk, 1, True, True)
            pred = pred.t()
            correct = pred.eq(target.view(1, -1).expand_as(pred))

            res = []
            for k in topk:
                correct_k = correct[:k].reshape(-1).float().sum(0, keepdim=True)
                res.append(correct_k.mul_(100.0 / batch_size))
            return res
