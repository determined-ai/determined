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

# shiftfit.py cloned into directory via startup-hook.sh during container initialization.
import shiftvit


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
        # timm's scheduler expects an epoch_idx arg. Track when the epoch changes.
        self._last_epoch_idx = None

    def build_callbacks(self) -> Dict[str, PyTorchCallback]:
        """Builds the callback for stepping the ShiftViT learning-rate scheduler used when training on ImageNet."""
        callbacks = {}
        callbacks["timing_callback"] = TimingCallback()
        return callbacks

    def build_training_data_loader(self) -> DataLoader:
        training_data_loader = self._get_data_loader(train=True)

        # epoch_idx is internally computed based on the len of the training dataloader.  If records_per_epoch is set
        # in the config file, and records_per_epoch is not equal to the length of the training dataset, this can create
        # two different notions of epoch lengths.
        records_per_epoch = self.context.get_experiment_config().get(
            "records_per_epoch", None
        )
        if records_per_epoch is not None:
            global_batch_size = self.context.get_global_batch_size()
            records_per_epoch_batches = int(
                math.ceil(records_per_epoch / global_batch_size)
            )
            if records_per_epoch_batches != len(training_data_loader):
                warning_msg = (
                    f"The 'records_per_epoch' configuration yields {records_per_epoch_batches} batches, "
                    f"while the len of the training DataLoader yields {len(training_data_loader)} batches. Epoch "
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

        if epoch_idx != self._last_epoch_idx:
            self._last_epoch_idx = epoch_idx
            self.scheduler.step(epoch=epoch_idx)

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
    """
    Callback for use when training on ImageNet. Steps the ShiftViT learning-rate scheduler at the end of each epoch,
    following the ShiftViT training procedures, and prints epoch timing statistics.
    """

    def __init__(self) -> None:
        self._trial_start_time = None
        self._training_epoch_start_time = None
        self._validation_epoch_start_time = None

    def on_trial_startup(
        self, first_batch_idx: int, checkpoint_uuid: Optional[str]
    ) -> None:
        self._trial_start_time = time.perf_counter()

    def on_trial_shutdown(self) -> None:
        trial_end_time = time.perf_counter()
        print(f"Trial duration: {trial_end_time - self._trial_start_time:.4f} seconds")

    def on_training_epoch_start(self, epoch_idx: int) -> None:
        print(f"Starting training epoch {epoch_idx}")
        self._training_epoch_start_time = time.perf_counter()

    def on_training_epoch_end(self, epoch_idx: int) -> None:
        print(f"Ending training epoch {epoch_idx}")
        training_epoch_end_time = time.perf_counter()
        print(
            f"Training epoch {epoch_idx} duration: {training_epoch_end_time - self._training_epoch_start_time:.4f} seconds"
        )

    def on_validation_epoch_start(self) -> None:
        print("val epoch start")
        self._validation_epoch_start_time = time.perf_counter()

    def on_validation_epoch_end(self, outputs: List[Any]) -> None:
        validation_epoch_end_time = time.perf_counter()
        print(
            f"Validation epoch duration: {validation_epoch_end_time - self._validation_epoch_start_time:.4f} seconds"
        )
