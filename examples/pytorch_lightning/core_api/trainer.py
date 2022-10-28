# Adapted from
# https://pytorch-lightning.readthedocs.io/en/stable/notebooks/lightning_examples/cifar10-baseline.html

from attrdict import AttrDict
import json
import logging
from typing import Any, cast, Dict, Optional, Tuple

import determined as det
import numpy as np
import torch
import torch.nn as nn
import torch.nn.functional as F
import torchvision
from pl_bolts.datamodules import CIFAR10DataModule
from pl_bolts.transforms.dataset_normalizations import cifar10_normalization
import pytorch_lightning as pl
from torch.optim.lr_scheduler import LambdaLR
from torchmetrics.functional import accuracy

from integration import build_determined_trainer, DeterminedDeepSpeedStrategy


def get_hyperparameters() -> AttrDict:
    info = det.get_cluster_info()
    assert info is not None, "This example only runs on-cluster"
    return cast(Dict, AttrDict(info.trial.hparams))


class LitResnet(pl.LightningModule):  # type: ignore
    def __init__(self) -> None:
        super().__init__()
        # We rely on passing hyperparameters in from Determined, rather than accepting arguments
        # in the constructor.  This is necessary to work properly with build_determined_trainer.
        self.hparams.update(get_hyperparameters())
        self.model = torchvision.models.resnet18(pretrained=False, num_classes=10)
        self.model.conv1 = nn.Conv2d(
            3, 64, kernel_size=(3, 3), stride=(1, 1), padding=(1, 1), bias=False
        )
        self.model.maxpool = nn.Identity()  # type: ignore

    def forward(self, x: torch.Tensor) -> torch.Tensor:
        out = self.model(x)
        return F.log_softmax(out, dim=1)

    def training_step(
        self, batch: Tuple[torch.Tensor, torch.Tensor], batch_idx: int
    ) -> pl.utilities.types.STEP_OUTPUT:
        x, y = batch
        logits = self(x)
        loss = F.nll_loss(logits, y)
        result = {"loss": loss}
        self.log_dict(result)
        return result

    def evaluate(
        self, batch: Tuple[torch.Tensor, torch.Tensor], stage: Optional[str] = None
    ) -> pl.utilities.types.STEP_OUTPUT:
        x, y = batch
        logits = self(x)
        loss = F.nll_loss(logits, y)
        preds = torch.argmax(logits, dim=1)
        acc = accuracy(preds, y)

        if stage:
            result = {f"{stage}_loss": loss, f"{stage}_acc": acc}
            self.log_dict(result, prog_bar=True, sync_dist=True)
            return result

    def validation_step(
        self, batch: Tuple[torch.Tensor, torch.Tensor], batch_idx: int
    ) -> pl.utilities.types.STEP_OUTPUT:
        return self.evaluate(batch, "val")

    def test_step(
        self, batch: Tuple[torch.Tensor, torch.Tensor], batch_idx: int
    ) -> pl.utilities.types.STEP_OUTPUT:
        return self.evaluate(batch, "test")

    def configure_optimizers(self) -> Any:
        optimizer = torch.optim.Adam(
            self.parameters(),
            lr=self.hparams["lr"],
            weight_decay=5e-4,
        )
        scheduler_dict = {
            "scheduler": LambdaLR(
                optimizer, lr_lambda=lambda epoch: lr_schedule(epoch, self.trainer.max_epochs)
            ),
            "interval": "epoch",
        }
        return {"optimizer": optimizer, "lr_scheduler": scheduler_dict}


def lr_schedule(epoch: int, max_epochs: int) -> float:
    return cast(float, np.interp(epoch / max_epochs, [0.0, 0.45, 0.9, 1.0], [0.1, 1.0, 0.1, 0.0]))


logging.basicConfig(level=logging.INFO, handlers=[logging.StreamHandler()])
pl.seed_everything(7)
hparams = get_hyperparameters()
train_transforms = torchvision.transforms.Compose(
    [
        torchvision.transforms.RandomCrop(32, padding=4),
        torchvision.transforms.RandomHorizontalFlip(),
        torchvision.transforms.ToTensor(),
        cifar10_normalization(),
    ]
)
test_transforms = torchvision.transforms.Compose(
    [
        torchvision.transforms.ToTensor(),
        cifar10_normalization(),
    ]
)
# Note that CIFAR10DataModule does not save/load any state, so pausing and resuming experiments
# will give different results than running to completion.
# It would be better to write a custom data module that tracked where it left off.
cifar10_dm = CIFAR10DataModule(
    data_dir="/datasets", batch_size=hparams["batch_size"], num_workers=hparams["num_workers"]
)
cifar10_dm.train_transforms = train_transforms
cifar10_dm.val_transforms = test_transforms
cifar10_dm.test_transforms = test_transforms

hparams = get_hyperparameters()
use_deepspeed = "ds_config" in hparams
strategy = None
if use_deepspeed:
    distributed_context = det.core.DistributedContext.from_deepspeed()
    with open(hparams["ds_config"], "r") as f:
        strategy = DeterminedDeepSpeedStrategy(
            config=json.load(f), logging_batch_size_per_gpu=hparams["batch_size"]
        )
else:
    distributed_context = det.core.DistributedContext.from_torch_distributed()

with det.core.init(distributed=distributed_context) as core_context:
    trainer, model = build_determined_trainer(
        core_context,
        LitResnet,
        num_nodes=distributed_context.cross_size,
        gpus=distributed_context.local_size,
        accelerator="auto",
        devices=1 if torch.cuda.is_available() else None,
        logger=pl.loggers.CSVLogger(save_dir="logs/"),
        callbacks=[
            pl.callbacks.LearningRateMonitor(logging_interval="step"),
            pl.callbacks.progress.TQDMProgressBar(refresh_rate=10),
        ],
        strategy=strategy,
    )
    trainer.fit(model, cifar10_dm)
    # Note that PTL advises running testing on a single GPU to ensure that distributed sampling
    # doesn't result in repeated data points.  If we cared about exactness (e.g. for publishing),
    # we'd set up a separate 1 GPU experiment to do so.
    trainer.test(model, datamodule=cifar10_dm)
