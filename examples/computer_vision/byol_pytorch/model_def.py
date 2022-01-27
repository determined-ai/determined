from attrdict import AttrDict
import math
import os
from typing import Any, cast, Dict, Sequence, Union

from byol_pytorch import BYOL
from determined.pytorch import (
    PyTorchCallback,
    PyTorchTrial,
    PyTorchTrialContext,
    DataLoader,
)
from determined.tensorboard.metric_writers.pytorch import TorchWriter
import torch
import torch.nn as nn
import torch.nn.functional as F
from torch.optim import Optimizer

from backbone import BACKBONE_METADATA_BY_NAME
from data import (
    build_dataset,
    build_evaluation_transform,
    build_training_transform,
    DATASET_METADATA_BY_NAME,
    DatasetSplit,
    JointDataset,
)
from optim import (
    build_byol_optimizer,
    build_cls_optimizer,
    reset_model_parameters,
    reset_sgd_optimizer,
)
from reducers import AvgReducer, ValidatedAccuracyReducer

TorchData = Union[Dict[str, torch.Tensor], Sequence[torch.Tensor], torch.Tensor]


def set_learning_rate_warmup_cosine_anneal(
    hparams: AttrDict,
    optimizer: Optimizer,
    global_batch_size: int,
    batch_idx: int,
    batches_per_epoch: int,
) -> None:
    """
    Cosine annealing with warmup, as described in BYOL paper.
    """
    # Learning rate scales linearly with batch size, is equal to base when global_batch_size==base_batch_size.
    p = hparams.self_supervised.learning_rate
    assert hparams.total_epochs > p.warmup_epochs
    base_lr = p.base * (global_batch_size / p.base_batch_size)
    fractional_epoch = batch_idx / batches_per_epoch
    if fractional_epoch <= p.warmup_epochs:
        # Warmup domain.
        adjusted_lr = base_lr * (fractional_epoch / p.warmup_epochs)
    else:
        # Cosine annealing domain.
        cosine_progress = (fractional_epoch - p.warmup_epochs) / (
            hparams.total_epochs - p.warmup_epochs
        )
        cosine_multiplier = 1 + math.cos(math.pi * cosine_progress)
        adjusted_lr = base_lr * cosine_multiplier
    for param_group in optimizer.param_groups:
        param_group["lr"] = adjusted_lr


def set_ema_beta_cosine_anneal(
    hparams: AttrDict, byol_model: BYOL, batch_idx: int, batches_per_epoch: int
) -> None:
    fractional_epoch = batch_idx / batches_per_epoch
    # 1 − (1 − τbase) · (cos(πk/K) + 1)/2
    progress = fractional_epoch / hparams.total_epochs
    ema_beta = 1 - (
        (1 - hparams.self_supervised.moving_average_decay_base)
        * (math.cos(math.pi * progress) + 1)
        / 2
    )
    byol_model.target_ema_updater.beta = ema_beta


def classifier_loss(
    hparams: AttrDict, input_logits: torch.Tensor, target: torch.Tensor
) -> torch.Tensor:
    """
    Classifier loss is as described in section C.1 of BYOL paper.
    """
    if hparams.classifier.logit_clipping.enabled:
        clipped_logits = hparams.classifier.logit_clipping.alpha * torch.tanh(
            input_logits / hparams.classifier.logit_clipping.alpha
        )
    else:
        clipped_logits = input_logits
    return (
        F.cross_entropy(clipped_logits, target)
        + hparams.classifier.logit_regularization_beta * (clipped_logits ** 2).mean()
    )


class BYOLTrial(PyTorchTrial):
    def __init__(self, context: PyTorchTrialContext) -> None:
        self.context = context
        self.hparams = AttrDict(self.context.get_hparams())
        self.data_config = AttrDict(self.context.get_data_config())
        self.hparams.total_epochs = self.context.get_experiment_config()["searcher"][
            "max_length"
        ]["epochs"]
        self.rank = self.context.distributed.get_rank()
        self.logger = TorchWriter()
        if self.data_config.use_rank_for_download_dir:
            # Avoid process contention when downloading data inside each process.
            self.download_dir = os.path.join(
                self.data_config.download_dir, str(self.rank)
            )
        else:
            self.download_dir = self.data_config.download_dir
        self._init_transforms()
        self._init_self_supervised()
        self._init_classifiers()
        self._init_reducers()
        # Create a separate dataloader for training the classifier.
        # With distributed training, calling get_data_loader on the Determined dataloader will automatically
        # shard the dataset.
        self.train_cls_dataloader = (
            self._build_cls_training_data_loader().get_data_loader(
                repeat=False,
                num_replicas=self.context.distributed.get_size(),
                rank=self.rank,
            )
        )

    def _init_transforms(self) -> None:
        """
        Create training and evaluation transforms.
        """
        dataset_metadata = DATASET_METADATA_BY_NAME[self.data_config.dataset_name]
        mean = dataset_metadata.mean
        std = dataset_metadata.std
        # BYOL paper uses two different distributions for training transforms.
        # For some reason, byol-pytorch wraps its transforms in nn.Sequential.
        self.data_config.train_transform_fn1 = build_training_transform(
            self.data_config.train_transform1, mean, std
        )
        self.data_config.train_transform_fn2 = build_training_transform(
            self.data_config.train_transform2, mean, std
        )
        self.data_config.eval_transform_fn = build_evaluation_transform(
            self.data_config.eval_transform, mean, std
        )

    def _init_self_supervised(self) -> None:
        """
        Create BYOL network and optimizer.
        """
        backbone_metadata = BACKBONE_METADATA_BY_NAME[self.hparams.backbone_name]
        net = backbone_metadata.build_fn()
        # Transform random_crop_size determines final image size.
        assert (
            self.data_config.train_transform1.random_crop_size
            == self.data_config.train_transform2.random_crop_size
        ), "Crop size must be the same for all transforms."
        assert (
            self.data_config.eval_transform.center_crop_size
            == self.data_config.train_transform1.random_crop_size
        ), "Crop size must be the same for all transforms."
        image_size = self.data_config.train_transform1.random_crop_size
        self.byol_model = cast(
            BYOL,
            self.context.wrap_model(BYOL(net, image_size)),
        )
        self.byol_opt = self.context.wrap_optimizer(
            build_byol_optimizer(self.hparams, self.byol_model)
        )

    def _init_classifiers(self) -> None:
        """
        Create classifier networks and optimizers, one for each candidate LR.
        """
        backbone_metadata = BACKBONE_METADATA_BY_NAME[self.hparams.backbone_name]
        dataset_metadata = DATASET_METADATA_BY_NAME[self.data_config.dataset_name]
        # Ensure learning rates are unique.
        assert len(self.hparams.classifier.learning_rates) == len(
            set(self.hparams.classifier.learning_rates)
        )
        self.cls_criterion = nn.CrossEntropyLoss()
        self.cls_models = [
            self.context.wrap_model(
                torch.nn.Linear(
                    backbone_metadata.feature_size, dataset_metadata.num_classes
                )
            )
            for lr in self.hparams.classifier.learning_rates
        ]
        self.cls_opts = [
            self.context.wrap_optimizer(
                build_cls_optimizer(self.hparams, lr, self.cls_models[lr_idx])
            )
            for lr_idx, lr in enumerate(self.hparams.classifier.learning_rates)
        ]

    def _init_reducers(self) -> None:
        self.final_acc_reducer = cast(
            ValidatedAccuracyReducer,
            self.context.wrap_reducer(
                ValidatedAccuracyReducer(), "test_accuracy", for_training=False
            ),
        )
        self.cls_loss_reducers = [
            cast(
                AvgReducer,
                self.context.wrap_reducer(
                    AvgReducer(), f"cls_loss_{i}", for_training=False
                ),
            )
            for i in range(len(self.hparams.classifier.learning_rates))
        ]

    def build_callbacks(self) -> Dict[str, PyTorchCallback]:
        return {"classifier_train": ClassifierTrainCallback(trial=self)}

    def build_training_data_loader(self) -> DataLoader:
        train_dataset = build_dataset(
            self.data_config, self.download_dir, split=DatasetSplit.TRAIN
        )
        # In order to ensure distributed training shards correctly, we round the dataset size
        # down to be a multiple of the global batch size.
        # See comment here for more details:
        #   https://github.com/determined-ai/determined/blob/b3a34baa7dcca788a090120a17f9a4f8dc1a4184/harness/determined/pytorch/samplers.py#L77
        rounded_length = (
            len(train_dataset) // self.context.get_global_batch_size()
        ) * self.context.get_global_batch_size()
        train_dataset = torch.utils.data.Subset(train_dataset, range(rounded_length))
        return DataLoader(
            train_dataset,
            batch_size=self.context.get_per_slot_batch_size(),
            pin_memory=True,
            num_workers=self.data_config.num_workers,
            shuffle=True,
            drop_last=True,
        )

    def _build_cls_training_data_loader(self) -> DataLoader:
        cls_train_dataset = build_dataset(
            self.data_config, self.download_dir, DatasetSplit.CLS_TRAIN
        )
        rounded_length = (
            len(cls_train_dataset) // self.context.get_global_batch_size()
        ) * self.context.get_global_batch_size()
        cls_train_dataset = torch.utils.data.Subset(
            cls_train_dataset, range(rounded_length)
        )
        return DataLoader(
            cls_train_dataset,
            batch_size=self.context.get_per_slot_batch_size(),
            pin_memory=True,
            num_workers=self.data_config.num_workers,
            shuffle=True,
            drop_last=True,
        )

    def build_validation_data_loader(self) -> DataLoader:
        # BYOL performs validation in two steps:
        # - A subset of the training data is used to select an optimal LR for the classifier.
        # - Final results are reported on the test / validation data.
        # We combine these two datasets, and then use a custom reducer to calculate the final result.
        lr_val_dataset = build_dataset(
            self.data_config,
            self.download_dir,
            DatasetSplit.CLS_VALIDATION,
        )
        test_dataset = build_dataset(
            self.data_config, self.download_dir, DatasetSplit.TEST
        )
        combined = JointDataset([lr_val_dataset, test_dataset], ["lr_val", "test"])
        return DataLoader(
            combined,
            batch_size=self.context.get_per_slot_batch_size(),
            pin_memory=True,
            num_workers=self.data_config.num_workers,
        )

    def train_batch(
        self, batch: TorchData, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        imgs1, imgs2, _ = batch
        set_learning_rate_warmup_cosine_anneal(
            self.hparams,
            self.byol_opt,
            self.context.get_global_batch_size(),
            batch_idx,
            cast(int, self.context._epoch_len),
        )
        loss = self.byol_model.forward(imgs1, imgs2)
        self.context.backward(loss)
        self.context.step_optimizer(self.byol_opt)
        set_ema_beta_cosine_anneal(
            self.hparams, self.byol_model, batch_idx, cast(int, self.context._epoch_len)
        )
        # Note: EMA requires an aggregation_frequency of 1 or online network may not be synced
        # across processes at this point.
        self.byol_model.update_moving_average()
        return {"loss": loss}

    def evaluate_batch(self, batch: TorchData, batch_idx: int) -> Dict[str, Any]:
        # Evaluation batches run simultaneously over LR validation subset and test dataset (using JointDataset)
        # We use a custom reducer to report test performance on the best validated LR.
        imgs, labels, d_idx = batch
        embeddings = self.byol_model.forward(
            imgs, return_embedding=True, return_projection=False
        )
        correct_cts_val = {}
        correct_cts_test = {}
        test_ct = cast(int, torch.sum(cast(torch.Tensor, d_idx) == 1).item())
        for lr_idx, lr in enumerate(self.hparams.classifier.learning_rates):
            model = self.cls_models[lr_idx]
            logits = cast(torch.Tensor, model(embeddings))
            preds = torch.argmax(logits, 1)
            correct_cts_val[lr_idx] = cast(
                int, torch.sum((preds == labels) * (d_idx == 0)).item()
            )
            correct_cts_test[lr_idx] = cast(
                int, torch.sum((preds == labels) * (d_idx == 1)).item()
            )
        self.final_acc_reducer.update(correct_cts_val, correct_cts_test, test_ct)
        return {}


class ClassifierTrainCallback(PyTorchCallback):
    """
    Performs classifier head training on frozen self-supervised network before each validation epoch.
    """

    def __init__(self, trial: BYOLTrial):
        self.trial = trial

    def on_validation_start(self) -> None:
        trial = self.trial
        # Reset models and optimization before each validation epoch -- want them trained from
        # the most recent frozen model.
        for lr_idx in range(len(trial.hparams.classifier.learning_rates)):
            trial.cls_models[lr_idx].train()
            reset_model_parameters(trial.cls_models[lr_idx])
            reset_sgd_optimizer(trial.cls_opts[lr_idx])
        print(
            f"Training classifier heads for {trial.hparams.classifier.train_epochs} epochs..."
        )
        for e in range(trial.hparams.classifier.train_epochs):
            print(f"Training epoch {e}")
            for batch in trial.train_cls_dataloader:
                imgs, labels = batch
                imgs = imgs.cuda()
                labels = labels.cuda()
                embeddings = trial.byol_model(
                    imgs, return_embedding=True, return_projection=False
                ).detach()
                for lr_idx, lr in enumerate(trial.hparams.classifier.learning_rates):
                    with torch.enable_grad():
                        model = trial.cls_models[lr_idx]
                        opt = trial.cls_opts[lr_idx]
                        logits = cast(torch.Tensor, model(embeddings))
                        cls_loss = classifier_loss(trial.hparams, logits, labels)
                        # Record avg loss over last epoch.
                        if e == trial.hparams.classifier.train_epochs - 1:
                            trial.cls_loss_reducers[lr_idx].update(cls_loss.item())
                        trial.context.backward(cls_loss)
                        trial.context.step_optimizer(opt)
        # Set models back to eval mode.
        for lr_idx in range(len(trial.hparams.classifier.learning_rates)):
            trial.cls_models[lr_idx].eval()
