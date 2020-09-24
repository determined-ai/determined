"""
This model is from the CNN NAS search space considered in:
    https://openreview.net/forum?id=S1eYHoC5FX

We will use the adaptive searcher in Determined to find a
good architecture in this search space for CIFAR-10.  
"""

from collections import namedtuple
from typing import Any, Dict

import torch
from torch import nn
import torchvision.datasets as dset

import determined as det
from determined.pytorch import DataLoader, LRScheduler, PyTorchTrial, reset_parameters, PyTorchCallback, ClipGradsL2Norm

from model import NetworkCIFAR as Network
import utils

Genotype = namedtuple("Genotype", "normal normal_concat reduce reduce_concat")


class AttrDict(dict):
    def __init__(self, *args, **kwargs):
        super(AttrDict, self).__init__(*args, **kwargs)
        self.__dict__ = self


class DARTSCNNTrial(PyTorchTrial):
    def __init__(self, context: det.TrialContext) -> None:
        self.context = context
        self.data_config = context.get_data_config()
        self.hparams = context.get_hparams()
        self.criterion = torch.nn.functional.cross_entropy
        # The last epoch is only used for logging.
        self._last_epoch = -1

    def build_training_data_loader(self) -> DataLoader:
        train_transform, valid_transform = utils._data_transforms_cifar10(
            AttrDict(self.hparams)
        )
        train_data = dset.CIFAR10(
            root="{}/data-rank{}".format(
                self.data_config["data_download_dir"],
                self.context.distributed.get_rank(),
            ),
            train=True,
            download=True,
            transform=train_transform,
        )

        train_queue = DataLoader(
            train_data,
            batch_size=self.context.get_per_slot_batch_size(),
            shuffle=True,
            pin_memory=True,
            num_workers=2,
        )

        return train_queue

    def build_validation_data_loader(self) -> DataLoader:
        train_transform, valid_transform = utils._data_transforms_cifar10(
            AttrDict(self.hparams)
        )
        valid_data = dset.CIFAR10(
            root="{}/data-rank{}".format(
                self.data_config["data_download_dir"],
                self.context.distributed.get_rank(),
            ),
            train=False,
            download=True,
            transform=valid_transform,
        )

        valid_queue = DataLoader(
            valid_data,
            batch_size=self.context.get_per_slot_batch_size(),
            shuffle=False,
            pin_memory=True,
            num_workers=2,
        )

        return valid_queue

    def get_genotype_from_hps(self):
        # This function creates an architecture definition
        # from the hyperparameter settings.
        cell_config = {"normal": [], "reduce": []}

        for cell in ["normal", "reduce"]:
            for node in range(4):
                for edge in [1, 2]:
                    edge_ind = self.hparams[
                        "{}_node{}_edge{}".format(cell, node + 1, edge)
                    ]
                    edge_op = self.hparams[
                        "{}_node{}_edge{}_op".format(cell, node + 1, edge)
                    ]
                    cell_config[cell].append((edge_op, edge_ind))
        print(cell_config)
        return Genotype(
            normal=cell_config["normal"],
            normal_concat=range(2, 6),
            reduce=cell_config["reduce"],
            reduce_concat=range(2, 6),
        )

    def build_model(self) -> nn.Module:
        genotype = self.get_genotype_from_hps()

        model = Network(
            self.hparams["init_channels"],
            10,  # num_classes
            self.hparams["layers"],
            self.hparams["auxiliary"],
            genotype,
        )
        print("param size = {} MB".format(utils.count_parameters_in_MB(model)))
        size = 0
        for p in model.parameters():
            size += p.nelement()
        print("param count: {}".format(size))

        # If loading backbone weights, do not call reset_parameters() or
        # call before loading the backbone weights.
        reset_parameters(model)
        return model

    def optimizer(self, model: nn.Module) -> torch.optim.Optimizer:  
        return torch.optim.SGD(
            model.parameters(),
            lr=self.context.get_hparam("learning_rate"),
            momentum=self.context.get_hparam("momentum"),
            weight_decay=self.context.get_hparam("weight_decay"),
        )

    def create_lr_scheduler(self, optimizer):
        self.scheduler = torch.optim.lr_scheduler.CosineAnnealingLR(
            optimizer, self.context.get_hparam("train_epochs")
        )
        step_mode = LRScheduler.StepMode.STEP_EVERY_EPOCH
        return LRScheduler(self.scheduler, step_mode=step_mode)

    def train_batch(
        self, batch: Any, model: nn.Module, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        input, target = batch
        model.drop_path_prob = (
            self.hparams["drop_path_prob"] * (self.scheduler.last_epoch) / self.hparams["train_epochs"]
        )
        if batch_idx == 0 or epoch_idx > self._last_epoch:
            print("epoch {} lr: {}".format(epoch_idx, self.scheduler.get_last_lr()[0]))
            print("drop_path_prob: {}".format(model.drop_path_prob))
        self._last_epoch = epoch_idx

        logits, logits_aux = model(input)
        loss = self.criterion(logits, target)
        if self.context.get_hparam("auxiliary"):
            loss_aux = self.criterion(logits_aux, target)
            loss += self.context.get_hparam("auxiliary_weight") * loss_aux
        top1, top5 = utils.accuracy(logits, target, topk=(1, 5))

        return {"loss": loss, "top1_accuracy": top1, "top5_accuracy": top5}

    def evaluate_batch(self, batch: Any, model: nn.Module) -> Dict[str, Any]:
        input, target = batch
        logits, _ = model(input)
        loss = self.criterion(logits, target)
        top1, top5 = utils.accuracy(logits, target, topk=(1, 5))

        return {"loss": loss, "top1_accuracy": top1, "top5_accuracy": top5}

    def build_callbacks(self) -> Dict[str, PyTorchCallback]:
        return {
            "clip_grads": ClipGradsL2Norm(
                self.context.get_hparam("clip_gradients_l2_norm")
            )
        }

