from typing import Any, Dict, Union, Sequence
import os

import torch
import torch.nn as nn
import torch.nn.functional as F
import torchvision.datasets as dset
from torch.optim.lr_scheduler import CosineAnnealingLR

from determined.pytorch import (
    PyTorchTrial,
    PyTorchTrialContext,
    DataLoader,
    LRScheduler,
    PyTorchCallback
)

from data import BilevelDataset
from model_search import Network
from optimizer import EG
from utils import AttrDict, data_transforms_cifar10, accuracy

TorchData = Union[Dict[str, torch.Tensor], Sequence[torch.Tensor], torch.Tensor]


class GenotypeCallback(PyTorchCallback):
    def __init__(self, context):
        self.model = context.models[0]

    def on_validation_end(self, metrics):
        print(self.model.genotype())


class GAEASearchTrial(PyTorchTrial):
    def __init__(self, trial_context: PyTorchTrialContext) -> None:
        self.context = trial_context
        self.data_config = trial_context.get_data_config()
        self.hparams = AttrDict(trial_context.get_hparams())
        self.last_epoch = 0

        self.data_dir = os.path.join(
            self.data_config["download_dir"],
            f"data-rank{self.context.distributed.get_rank()}",
        )

        # Initialize the models.
        criterion = nn.CrossEntropyLoss()
        self.model = self.context.wrap_model(
            Network(
                self.hparams.init_channels,
                self.hparams.n_classes,
                self.hparams.layers,
                criterion,
                self.hparams.nodes,
                k=self.hparams.shuffle_factor,
            )
        )

        # Initialize the optimizers and learning rate scheduler.
        self.ws_opt = self.context.wrap_optimizer(
            torch.optim.SGD(
                self.model.ws_parameters(),
                self.hparams.learning_rate,
                momentum=self.hparams.momentum,
                weight_decay=self.hparams.weight_decay,
            )
        )
        self.arch_opt = self.context.wrap_optimizer(
            EG(
                self.model.arch_parameters(),
                self.hparams.arch_learning_rate,
                lambda p: p / p.sum(dim=-1, keepdim=True),
            )
        )

        self.lr_scheduler = self.context.wrap_lr_scheduler(
            lr_scheduler=CosineAnnealingLR(
                self.ws_opt,
                self.hparams.scheduler_epochs,
                self.hparams.min_learning_rate,
            ),
            step_mode=LRScheduler.StepMode.STEP_EVERY_EPOCH,
        )

    def build_training_data_loader(self) -> DataLoader:
        """
        For bi-level NAS, we'll need each instance from the dataloader to have one image
        for training shared-weights and another for updating architecture parameters.
        """
        train_transform, _ = data_transforms_cifar10()
        train_data = dset.CIFAR10(
            root=self.data_dir, train=True, download=True, transform=train_transform
        )
        bilevel_data = BilevelDataset(train_data)

        self.train_data = bilevel_data

        train_queue = DataLoader(
            bilevel_data,
            batch_size=self.context.get_per_slot_batch_size(),
            shuffle=True,
            num_workers=2,
        )
        return train_queue

    def build_validation_data_loader(self) -> DataLoader:
        _, valid_transform = data_transforms_cifar10()
        valid_data = dset.CIFAR10(
            root=self.data_dir, train=False, download=True, transform=valid_transform
        )
        valid_queue = DataLoader(
            valid_data,
            batch_size=self.context.get_per_slot_batch_size(),
            shuffle=False,
            num_workers=2,
        )
        return valid_queue

    def train_batch(
        self, batch: TorchData, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        if epoch_idx != self.last_epoch:
            self.train_data.shuffle_val_inds()
        self.last_epoch = epoch_idx
        x_train, y_train, x_val, y_val = batch

        # Train shared-weights
        for a in self.model.arch_parameters():
            a.requires_grad = False
        for w in self.model.ws_parameters():
            w.requires_grad = True
        loss = self.model._loss(x_train, y_train)
        self.context.backward(loss)
        self.context.step_optimizer(self.ws_opt)

        # Train arch parameters
        for a in self.model.arch_parameters():
            a.requires_grad = True
        for w in self.model.ws_parameters():
            w.requires_grad = False
        arch_loss = self.model._loss(x_val, y_val)
        self.context.backward(arch_loss)
        self.context.step_optimizer(self.arch_opt)

        return {
            "loss": loss,
            "arch_loss": arch_loss,
        }

    def evaluate_batch(self, batch: TorchData) -> Dict[str, Any]:
        input, target = batch
        logits = self.model(input)
        loss = self.model._loss(input, target)
        top1, top5 = accuracy(logits, target, topk=(1, 5))

        return {"loss": loss, "top1_accuracy": top1, "top5_accuracy": top5}

    def build_callbacks(self):
        return {"genotype": GenotypeCallback(self.context)}
