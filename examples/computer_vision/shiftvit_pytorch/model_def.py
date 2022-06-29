"""
ShiftViT code and paper:s
https://github.com/microsoft/SPACH/blob/main/models/shiftvit.py
https://arxiv.org/abs/2201.10801
"""
import logging
import math
import time
import warnings

import attrdict
from determined.pytorch import (
    DataLoader,
    LRScheduler,
    PyTorchCallback,
    PyTorchTrial,
    PyTorchTrialContext,
)
from timm.optim import create_optimizer
from timm.scheduler import create_scheduler
import torch
from typing import Any, Dict, List, Optional, Sequence, Tuple, Union, cast

import data
import shiftvit  # shiftfit.py cloned via startup-hook.sh during container initialization.


TorchData = Union[Dict[str, torch.Tensor], Sequence[torch.Tensor], torch.Tensor]


class ShiftViTTrial(PyTorchTrial):
    def __init__(self, context: PyTorchTrialContext) -> None:
        self.context = context

        self.hparams = attrdict.AttrDict(self.context.get_hparams())
        self.model_config = self.hparams.model
        self.optimizer_config = self.hparams.optimizer
        self.scheduler_config = self.hparams.scheduler
        self.transform_config = self.hparams.transform
        self.data_config = attrdict.AttrDict(self.context.get_data_config())
        self.dataset_metadata = data.DATASET_METADATA_BY_NAME[self.data_config.dataset_name]

        self.model = self.context.wrap_model(
            shiftvit.ShiftViT(
                **self.dataset_metadata.to_dict(),
                **self.model_config,
            )
        )

        # Use timm's create_xxx factories for the optimizer and scheduler.
        optimizer = create_optimizer(self.optimizer_config, self.model)
        self.optimizer = self.context.wrap_optimizer(optimizer)

        # timm's scheduler expects to be stepped at the end of each epoch and to be passed the
        # epoch_idx as an arg. Passing epoch_idx requires using MANUAL_STEP mode.
        scheduler, _ = create_scheduler(self.scheduler_config, self.optimizer)
        self.scheduler = self.context.wrap_lr_scheduler(
            scheduler, step_mode=LRScheduler.StepMode.MANUAL_STEP
        )
        self._curr_epoch_idx = 0

    def build_callbacks(self) -> Dict[str, PyTorchCallback]:
        """Builds the callback for stepping the ShiftViT learning-rate scheduler used when training
        on ImageNet."""
        callbacks = {}
        callbacks["timing_callback"] = TimingCallback()
        return callbacks

    def build_training_data_loader(self) -> DataLoader:
        training_data_loader = self._get_data_loader(train=True)
        return training_data_loader

    def build_validation_data_loader(self) -> DataLoader:
        validation_data_loader = self._get_data_loader(train=False)
        return validation_data_loader

    def train_batch(
        self, batch: TorchData, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        batch = cast(Tuple[torch.Tensor, torch.Tensor], batch)
        # Step timm lr scheduler at the start of each epoch.
        if epoch_idx != self._curr_epoch_idx:
            self.scheduler.step(epoch_idx)
            self._curr_epoch_idx = epoch_idx

        images, labels = batch
        output = self.model(images)
        loss = torch.nn.functional.cross_entropy(output, labels)
        self.context.backward(loss)
        self.context.step_optimizer(self.optimizer)
        return {"loss": loss}

    def evaluate_batch(self, batch: TorchData) -> Dict[str, Any]:
        batch = cast(Tuple[torch.Tensor, torch.Tensor], batch)
        images, labels = batch
        output = self.model(images)
        validation_loss = torch.nn.functional.cross_entropy(output, labels).item()
        pred = output.argmax(dim=1, keepdim=True)
        accuracy = pred.eq(labels.view_as(pred)).sum().item() / len(images)
        return {"validation_loss": validation_loss, "accuracy": accuracy}

    def _get_data_loader(self, train: bool) -> DataLoader:
        """Constructs the appropriate datasets and relevant transforms using utilities from
        data.py.
        """
        transform = data.build_transform(
            dataset_metadata=self.dataset_metadata,
            transform_config=self.transform_config,
            train=train,
        )
        dataset = data.get_dataset(
            data_config=self.data_config,
            train=train,
            transform=transform,
        )
        return DataLoader(
            dataset,
            batch_size=self.context.get_per_slot_batch_size(),
            num_workers=self.data_config.num_workers,
            pin_memory=self.data_config.pin_memory,
            persistent_workers=self.data_config.persistent_workers,
            shuffle=train,
            drop_last=train,
        )
