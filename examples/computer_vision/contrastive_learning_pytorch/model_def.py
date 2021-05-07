from typing import Any, Dict, Sequence, Union

import math
import os

import attrdict
import torch
from torchvision import transforms, datasets

import determined.pytorch as det_torch
import resnet_big
import losses
import utils

TorchData = Union[Dict[str, torch.Tensor], Sequence[torch.Tensor], torch.Tensor]

NUM_CLASSES = {
    "cifar10": 10,
    "cifar100": 100,
}

STATS = {
    "cifar10": {
        "mean": (0.4914, 0.4822, 0.4465),
        "std": (0.2023, 0.1994, 0.2010),
    },
    "cifar100": {
        "mean": (0.5071, 0.4867, 0.4408),
        "std": (0.2675, 0.2565, 0.2761),
    },
}


class ContrastiveLearningTrial(det_torch.PyTorchTrial):
    def __init__(self, context: det_torch.PyTorchTrialContext) -> None:
        self.context = context
        self.hparams = attrdict.AttrDict(self.context.get_hparams())
        self.data_config = attrdict.AttrDict(self.context.get_data_config())
        self.experiment_config = self.context.get_experiment_config()
        self.rank = self.context.distributed.get_rank()
        self.distributed = self.context.distributed._hvd_config.use
        self.hparams.epochs = self.experiment_config["searcher"]["max_length"]["epochs"]
        self.download_dir = os.path.join(self.data_config.data_dir, "self.rank")
        os.makedirs(self.download_dir, exist_ok=True)

        # Contrastive loss head
        self.model = resnet_big.SupConResNet(self.hparams.backbone)
        self.criterion = losses.SupConLoss(
            temperature=self.hparams.temperature,
            distributed=self.distributed,
            rank=self.rank,
        ).cuda()

        if self.distributed and self.hparams.use_sync_bn:
            self.model = utils.convert_syncbn_model(self.model)

        self.model = self.context.wrap_model(self.model)

        self.optimizer = utils.set_optimizer(self.hparams, self.model)
        self.optimizer = self.context.wrap_optimizer(self.optimizer)

        if self.hparams.warm:
            if self.hparams.cosine:
                eta_min = self.hparams.learning_rate * (self.hparams.lr_decay_rate ** 3)
                self.hparams.warmup_to = (
                    eta_min
                    + (self.hparams.learning_rate - eta_min)
                    * (
                        1
                        + math.cos(
                            math.pi * self.hparams.warm_epochs / self.hparams.epochs
                        )
                    )
                    / 2
                )
            else:
                self.hparams.warmup_to = self.hparams.learning_rate

        # Classifier head
        self.classifier = self.context.wrap_model(
            resnet_big.LinearClassifier(
                self.hparams.backbone,
                NUM_CLASSES[self.data_config["dataset"]],
            )
        )
        self.cls_optimizer = self.context.wrap_optimizer(
            torch.optim.SGD(  # type: ignore
                self.classifier.parameters(),
                lr=self.hparams.cls_learning_rate,
                momentum=self.hparams.momentum,
            )
        )

        self.cls_lr_scheduler = self.context.wrap_lr_scheduler(
            torch.optim.lr_scheduler.MultiStepLR(
                self.cls_optimizer,
                milestones=[
                    int(self.hparams.epochs * 0.7),
                    int(self.hparams.epochs * 0.8),
                    int(self.hparams.epochs * 0.9),
                ],
                gamma=0.1,
            ),
            det_torch.LRScheduler.StepMode.STEP_EVERY_EPOCH,
        )

        self.cls_loss = torch.nn.CrossEntropyLoss().cuda()

    def build_training_data_loader(self) -> Any:
        assert self.data_config.dataset in STATS, "only supports cifar10 and cifar100"
        mean = STATS[self.data_config.dataset]["mean"]
        std = STATS[self.data_config.dataset]["std"]
        normalize = transforms.Normalize(mean=mean, std=std)

        train_transform = transforms.Compose(
            [
                transforms.RandomResizedCrop(
                    size=self.data_config.crop_size, scale=(0.2, 1.0)
                ),
                transforms.RandomHorizontalFlip(),
                transforms.RandomApply(
                    [transforms.ColorJitter(0.4, 0.4, 0.4, 0.1)], p=0.8
                ),
                transforms.RandomGrayscale(p=0.2),
                transforms.ToTensor(),
                normalize,
            ]
        )
        if self.data_config.dataset == "cifar10":
            train_dataset = datasets.CIFAR10(
                root=self.download_dir,
                transform=utils.TwoCropTransform(train_transform),
                download=True,
            )
        else:  # dataset is cifar100
            train_dataset = datasets.CIFAR100(
                root=self.download_dir,
                transform=utils.TwoCropTransform(train_transform),
                download=True,
            )

        # We will modify the dataset so that the automatically sharded datasets have the same len.
        rounded_length = (
            len(train_dataset) // self.context.get_global_batch_size()
        ) * self.context.get_global_batch_size()
        train_dataset = torch.utils.data.Subset(train_dataset, range(rounded_length))

        train_loader = det_torch.DataLoader(
            train_dataset,
            batch_size=self.context.get_per_slot_batch_size(),
            pin_memory=True,
            num_workers=self.data_config.num_workers,
            shuffle=True,
            drop_last=True,
        )
        return train_loader

    def build_validation_data_loader(self) -> Any:
        assert (
            self.data_config.dataset in STATS
        ), "only supports cifar10 and cifar100 datasets"
        mean = STATS[self.data_config.dataset]["mean"]
        std = STATS[self.data_config.dataset]["std"]
        normalize = transforms.Normalize(mean=mean, std=std)

        val_transform = transforms.Compose(
            [
                transforms.ToTensor(),
                normalize,
            ]
        )
        if self.data_config.dataset == "cifar10":
            val_dataset = datasets.CIFAR10(
                root=self.download_dir,
                train=False,
                transform=val_transform,
                download=True,
            )
        else:  # dataset is cifar100
            val_dataset = datasets.CIFAR100(
                root=self.download_dir,
                train=False,
                transform=val_transform,
                download=True,
            )

        val_loader = det_torch.DataLoader(
            val_dataset,
            batch_size=self.context.get_per_slot_batch_size(),
            pin_memory=True,
            num_workers=self.data_config.num_workers,
        )
        return val_loader

    def train_batch(
        self, batch: TorchData, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        if self.context.is_epoch_start():
            utils.adjust_learning_rate(self.hparams, self.optimizer, epoch_idx + 1)
            self.epoch = epoch_idx
        utils.warmup_learning_rate(
            self.hparams,
            epoch_idx + 1,
            batch_idx % self.context._epoch_len,
            self.context._epoch_len,
            self.optimizer,
        )

        images, labels = batch
        bsz = labels.shape[0]
        images = torch.cat([images[0], images[1]], dim=0)

        features = self.model(images)
        f1, f2 = torch.split(features, [bsz, bsz], dim=0)
        features = torch.cat([f1.unsqueeze(1), f2.unsqueeze(1)], dim=1)
        if self.hparams.supervised:
            loss = self.criterion(features, labels)
        else:  # simclr
            loss = self.criterion(features)

        if self.hparams.train_features:
            self.context.backward(loss)
            self.context.step_optimizer(self.optimizer)

        images = images[0:bsz]
        embeddings = self.model.encoder(images).detach()
        cls_loss = self.cls_loss(self.classifier(embeddings), labels)
        self.context.backward(cls_loss)
        self.context.step_optimizer(self.cls_optimizer)

        return {
            "loss": loss,
            "cls_loss": cls_loss,
            "lr": self.optimizer.param_groups[0]["lr"],
            "cls_lr": self.cls_lr_scheduler.get_last_lr(),
        }

    def evaluate_batch(self, batch: TorchData) -> Dict[str, Any]:
        images, labels = batch
        embeddings = self.model.encoder(images)
        logits = self.classifier(embeddings)
        preds = logits.argmax(1)
        return {
            "accuracy": float((preds == labels.to(torch.long)).sum() / preds.shape[0])
        }
