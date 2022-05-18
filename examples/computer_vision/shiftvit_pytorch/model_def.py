"""
ShiftViT code and paper:s
https://github.com/microsoft/SPACH/blob/main/models/shiftvit.py
https://arxiv.org/abs/2201.10801
"""
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
        self.dataset_metadata = data.DATASET_METADATA_BY_NAME[
            self.data_config.dataset_name
        ]

        self.model = self.context.wrap_model(
            shiftvit.ShiftViT(
                **self.dataset_metadata.to_dict(),
                **self.model_config,
            )
        )

        # Use timm's create_xxx factories for the optimizer and scheduler.
        optimizer = create_optimizer(self.optimizer_config, self.model)
        self.optimizer = self.context.wrap_optimizer(optimizer)

        scheduler, _ = create_scheduler(self.scheduler_config, self.optimizer)
        self.scheduler = self.context.wrap_lr_scheduler(
            scheduler, step_mode=LRScheduler.StepMode.MANUAL_STEP
        )

        # timm's scheduler expects to be stepped at the end of each epoch and to be passed an epoch_idx arg.
        # The internal epoch_idx arg used by Determined is computed based on the len of the un-sharded training
        # dataloader and its count is stepped when the number of globally processed batches surpasses an integer
        # multiple of this length.  Consequently, during distributed training workers which are simultaneously
        # processing the same batch_idx can have differing epoch_idx values near the end of an epoch.  Keying off of
        # epoch_idx while stepping the scheduler can therefore lead to un-synchronized stepping, so we instead perform
        # the epoch-end bookkeeping manually by tracking the appropriate batches-per-epoch. We default to using the
        # records_per_epoch field to define epoch lengths, if provided, and otherwise rely on the training
        # dataloader's len.
        self._records_per_epoch = self.context.get_experiment_config().get(
            "records_per_epoch", None
        )
        self._global_batch_size = self.context.get_global_batch_size()
        if self._records_per_epoch is None:
            self._batches_per_epoch = None
        else:
            self._batches_per_epoch = int(
                math.ceil(self._records_per_epoch / self._global_batch_size)
            )

        self._training_loader_len = None
        self._curr_epoch_idx = 0

    def build_callbacks(self) -> Dict[str, PyTorchCallback]:
        """Builds the callback for stepping the ShiftViT learning-rate scheduler used when training on ImageNet."""
        callbacks = {}
        callbacks["timing_callback"] = TimingCallback()
        return callbacks

    def build_training_data_loader(self) -> DataLoader:
        training_data_loader = self._get_data_loader(train=True)
        self._training_loader_len = len(training_data_loader)
        num_workers = self.context.distributed.size
        batches_per_epoch_from_dataloader = int(
            math.ceil(self._training_loader_len / num_workers)
        )
        # If records_per_epoch was not specified in the config, compute batches per epoch based on the dataloader len
        if self._records_per_epoch is None:
            self._batches_per_epoch = batches_per_epoch_from_dataloader
        elif batches_per_epoch_from_dataloader != self._batches_per_epoch:
            warning_msg = (
                f"The 'records_per_epoch' configuration yields {self._batches_per_epoch} batches-per-epoch, "
                f"while the len of the training DataLoader yields {batches_per_epoch_from_dataloader}. Epoch "
                "lengths are internally computed based on the latter number."
            )
            warnings.warn(warning_msg)
        return training_data_loader

    def build_validation_data_loader(self) -> DataLoader:
        validation_data_loader = self._get_data_loader(train=False)
        return validation_data_loader

    def train_batch(
        self, batch: TorchData, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        batch = cast(Tuple[torch.Tensor, torch.Tensor], batch)
        images, labels = batch

        output = self.model(images)
        loss = torch.nn.functional.cross_entropy(output, labels)

        self.context.backward(loss)
        self.context.step_optimizer(self.optimizer)

        # Step timm lr scheduler after completion of epoch.
        is_last_batch_of_epoch = (batch_idx + 1) % self._batches_per_epoch == 0
        if is_last_batch_of_epoch:
            self._curr_epoch_idx += 1
            print(
                f"LR SCHEDULER STEPPED AT EPOCH {epoch_idx}/{self._curr_epoch_idx} BATCH {batch_idx}"
            )
            print(f"LEN TRAIN LOADER: {self._training_loader_len}")
            print(f"BATCHES PER EPOCH {self._batches_per_epoch}")
            print(f"BATCH SHAPE {images.shape}")
            self.scheduler.step(epoch=self._curr_epoch_idx)

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
        """Constructs the appropriate datasets and relevant transforms using utilities from data.py."""
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
        )


class TimingCallback(PyTorchCallback):
    """Callback for computing and printing total duration of entire trial and of the training portion of the trial."""

    def __init__(self) -> None:
        self._trial_start_time = None
        self._train_start_time = None

    def on_trial_startup(
        self, first_batch_idx: int, checkpoint_uuid: Optional[str]
    ) -> None:
        self._trial_start_time = time.perf_counter()

    def on_training_start(self) -> None:
        self._train_start_time = time.perf_counter()
        print(
            f"Trial-start to training-start duration: {self._train_start_time - self._trial_start_time:.4f} seconds"
        )

    def on_trial_shutdown(self) -> None:
        trial_end_time = time.perf_counter()
        print(f"Trial duration: {trial_end_time - self._trial_start_time:.4f} seconds")
        print(
            f"Training-start to trial-end duration: {trial_end_time - self._train_start_time:.4f} seconds"
        )
