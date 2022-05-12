"""
ShiftViT code and paper:s
https://github.com/microsoft/SPACH/blob/main/models/shiftvit.py
https://arxiv.org/abs/2201.10801
"""
import time

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

        self.data_config = attrdict.AttrDict(self.context.get_data_config())
        self.dataset_metadata = data.DATASET_METADATA_BY_NAME[
            self.data_config.dataset_name
        ]

        self.model = self.context.wrap_model(
            shiftvit.ShiftViT(
                n_div=self.hparams.n_div,
                img_size=self.dataset_metadata.img_size,
                patch_size=self.hparams.patch_size,
                in_chans=self.dataset_metadata.in_chans,
                num_classes=self.dataset_metadata.num_classes,
                embed_dim=self.hparams.embed_dim,
                depths=self.hparams.depths,
                mlp_ratio=self.hparams.mlp_ratio,
                drop_rate=self.hparams.drop_rate,
                drop_path_rate=self.hparams.drop_path_rate,
                norm_layer=self.hparams.norm_layer,
                act_layer=self.hparams.act_layer,
                patch_norm=self.hparams.patch_norm,
                use_checkpoint=False,
            )
        )

        # When training on ImageNet, follow the ShiftViT training steps which use timm's create_xxx factories.
        self._is_using_imagenet = self.data_config.dataset_name == "imagenet"
        if self._is_using_imagenet:
            # Using timm's create_xxx factories.
            optimizer = create_optimizer(self.hparams.optimizer, self.model)
        else:
            optimizer = torch.optim.Adam(
                self.model.parameters(), lr=self.context.get_hparam("lr")
            )
        self.optimizer = self.context.wrap_optimizer(optimizer)
        # ImageNet scheduler must use the wrapped optimizer.
        if self._is_using_imagenet:
            scheduler, _ = create_scheduler(self.hparams.scheduler, self.optimizer)
            self.scheduler = self.context.wrap_lr_scheduler(
                scheduler, step_mode=LRScheduler.StepMode.MANUAL_STEP
            )

    def build_callbacks(self) -> Dict[str, PyTorchCallback]:
        """Builds the callback for stepping the ShiftViT learning-rate scheduler used when training on ImageNet."""
        callbacks = {}
        if self._is_using_imagenet:
            callbacks["imagenet_scheduler"] = ImageNetLRStepper(self.scheduler)
        return callbacks

    def build_training_data_loader(self) -> DataLoader:
        return self._get_data_loader(train=True)

    def build_validation_data_loader(self) -> DataLoader:
        return self._get_data_loader(train=False)

    def train_batch(
        self, batch: TorchData, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        batch = cast(Tuple[torch.Tensor, torch.Tensor], batch)
        data, labels = batch

        output = self.model(data)
        loss = torch.nn.functional.cross_entropy(output, labels)

        self.context.backward(loss)
        self.context.step_optimizer(self.optimizer)

        return {"loss": loss}

    def evaluate_batch(self, batch: TorchData) -> Dict[str, Any]:
        batch = cast(Tuple[torch.Tensor, torch.Tensor], batch)
        data, labels = batch

        output = self.model(data)
        validation_loss = torch.nn.functional.cross_entropy(output, labels).item()

        pred = output.argmax(dim=1, keepdim=True)
        accuracy = pred.eq(labels.view_as(pred)).sum().item() / len(data)

        return {"validation_loss": validation_loss, "accuracy": accuracy}

    def _get_data_loader(self, train: bool) -> DataLoader:
        """Constructs the appropriate datasets and relevant transforms using utilities from data.py."""
        transform = data.build_transform(
            dataset_metadata=self.dataset_metadata,
            data_config=self.data_config,
            train=train,
        )
        dataset = data.download_dataset(
            data_config=self.data_config,
            train=train,
            transform=transform,
        )
        return DataLoader(
            dataset,
            batch_size=self.context.get_per_slot_batch_size(),
            num_workers=self.data_config.num_workers,
            pin_memory=self.data_config.pin_memory,
            shuffle=train,
            drop_last=True,
        )


class ImageNetLRStepper(PyTorchCallback):
    """
    Callback for use when training on ImageNet. Steps the ShiftViT learning-rate scheduler at the end of each epoch,
    following the ShiftViT training procedures, and prints epoch timing statistics.
    """

    def __init__(self, wrapped_lr_scheduler) -> None:
        self.wrapped_lr_scheduler = wrapped_lr_scheduler
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
        self.wrapped_lr_scheduler.step(epoch=epoch_idx)

    def on_validation_epoch_start(self) -> None:
        print("val epoch start")
        self._validation_epoch_start_time = time.perf_counter()

    def on_validation_epoch_end(self, outputs: List[Any]) -> None:
        validation_epoch_end_time = time.perf_counter()
        print(
            f"Validation epoch duration: {validation_epoch_end_time - self._validation_epoch_start_time:.4f} seconds"
        )
