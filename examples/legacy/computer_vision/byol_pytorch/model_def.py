from enum import Enum, auto
from typing import Any, Dict, Sequence, Union, cast

import torch
import torch.nn.functional as F
from attrdict import AttrDict
from backbone import BACKBONE_METADATA_BY_NAME
from byol_pytorch import BYOL
from data import (
    DATASET_METADATA_BY_NAME,
    DatasetSplit,
    JointDataset,
    build_dataset,
    build_evaluation_transform,
    build_training_transform,
)
from optim import (
    build_byol_optimizer,
    build_cls_optimizer,
    reset_model_parameters,
    reset_sgd_optimizer,
    set_ema_beta_cosine_anneal,
    set_learning_rate_warmup_cosine_anneal,
)
from reducers import ValidatedAccuracyReducer
from torch.utils.data import Dataset
from utils import LambdaModule

from determined.pytorch import (
    DataLoader,
    PyTorchCallback,
    PyTorchTrial,
    PyTorchTrialContext,
    _SimpleReducer,
)

TorchData = Union[Dict[str, torch.Tensor], Sequence[torch.Tensor], torch.Tensor]


class TrainingMode(Enum):
    """
    SELF_SUPERVISED: Primary training mode, for training the self-supervised feature network.
    CLASSIFIER: Trains a classifier on the frozen self-supervised network for validation purposes.
    """

    SELF_SUPERVISED = auto()
    CLASSIFIER_ONLY = auto()


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
        + hparams.classifier.logit_regularization_beta * (clipped_logits**2).mean()
    )


class BYOLTrial(PyTorchTrial):
    def __init__(self, context: PyTorchTrialContext) -> None:
        self.context = context
        self.hparams = AttrDict(self.context.get_hparams())
        self.data_config = AttrDict(self.context.get_data_config())
        self.hparams.total_epochs = self.context.get_experiment_config()["searcher"]["max_length"][
            "epochs"
        ]
        self.rank = self.context.distributed.get_rank()
        try:
            self.training_mode = TrainingMode[self.hparams.training_mode]
        except ValueError:
            print(
                f"Training mode {self.hparams.training_mode} not supported: use one of {TrainingMode._member_names_}"
            )
        if self.training_mode == TrainingMode.CLASSIFIER_ONLY:
            assert (
                self.hparams.validate_with_classifier
            ), "Must set validate_with_classifier==true during CLASSIFIER_ONLY training."
        print(f"Training in mode {self.training_mode}")
        self._init_transforms()
        self._init_self_supervised()
        self._init_classifiers()
        self._init_reducers()
        if self._should_train_classifier_before_validation():
            # Create a separate dataloader for training the classifier during validation.
            # With distributed training, calling get_data_loader on the Determined dataloader will automatically
            # shard the dataset.
            self.train_cls_dataloader = self._build_cls_training_data_loader().get_data_loader(
                repeat=False,
                num_replicas=self.context.distributed.get_size(),
                rank=self.rank,
            )

    def _should_train_classifier_before_validation(self) -> bool:
        """
        When in SELF_SUPERVISED training mode, can train a classifier in on_validation_epoch_start before evaluating.
        Returns true if this should happen.
        """
        return (
            self.training_mode == TrainingMode.SELF_SUPERVISED
            and self.hparams.validate_with_classifier
        )

    def _should_evaluate_classifier(self) -> bool:
        """
        Can evaluate with either classifier accuracy or self-supervised loss.
        Returns true if a classifier is available, either because TrainingMode == CLASSIFER_ONLY or
        hparams.validate_with_classifier is set.
        """
        return (
            self.training_mode == TrainingMode.CLASSIFIER_ONLY
            or self.hparams.validate_with_classifier
        )

    def _init_transforms(self) -> None:
        """
        Create training and evaluation transforms.
        """
        dataset_metadata = DATASET_METADATA_BY_NAME[self.data_config.dataset_name]
        mean = dataset_metadata.mean
        std = dataset_metadata.std
        # BYOL paper uses two different distributions for training transforms.
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
        # Trick to support passing in pair of augmented images to byol_pytorch.
        # Normally, byol_pytorch does augmentations inside its forward pass, which is slow.
        self.byol_model.augment1 = LambdaModule(lambda x: x.first)
        self.byol_model.augment2 = LambdaModule(lambda x: x.second)
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
        self.cls_models = [
            self.context.wrap_model(
                torch.nn.Linear(backbone_metadata.feature_size, dataset_metadata.num_classes)
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
        if self._should_evaluate_classifier():
            self.final_acc_reducer = cast(
                ValidatedAccuracyReducer,
                self.context.wrap_reducer(
                    ValidatedAccuracyReducer(), "test_accuracy", for_training=False
                ),
            )
        if self._should_train_classifier_before_validation():
            self.cls_loss_reducers = [
                cast(
                    _SimpleReducer,
                    self.context.wrap_reducer(
                        lambda x: sum(x) / len(x), f"cls_loss_{i}", for_training=False
                    ),
                )
                for i in range(len(self.hparams.classifier.learning_rates))
            ]

    def build_callbacks(self) -> Dict[str, PyTorchCallback]:
        return {"classifier_train": ClassifierTrainCallback(trial=self)}

    def build_training_data_loader(self) -> DataLoader:
        if self.training_mode == TrainingMode.SELF_SUPERVISED:
            split = DatasetSplit.TRAIN
        elif self.training_mode == TrainingMode.CLASSIFIER_ONLY:
            split = DatasetSplit.CLS_TRAIN
        train_dataset = build_dataset(self.data_config, split=split)
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
        """
        Builds data loader for the on_validation_epoch_start classifier training when enabled.
        """
        cls_train_dataset = build_dataset(self.data_config, DatasetSplit.CLS_TRAIN)
        rounded_length = (
            len(cls_train_dataset) // self.context.get_global_batch_size()
        ) * self.context.get_global_batch_size()
        cls_train_dataset = torch.utils.data.Subset(cls_train_dataset, range(rounded_length))
        return DataLoader(
            cls_train_dataset,
            batch_size=self.context.get_per_slot_batch_size(),
            pin_memory=True,
            num_workers=self.data_config.num_workers,
            shuffle=True,
            drop_last=True,
        )

    def build_validation_data_loader(self) -> DataLoader:
        if self._should_evaluate_classifier():
            # The BYOL paper performs validation in two steps:
            # - A subset of the training data (CLS_VALIDATION) is used to select an optimal LR for the classifier.
            # - Final results are reported on the test / validation data (TEST).
            # We combine these two datasets, and then use a custom reducer to calculate the final result.
            cls_val_dataset = build_dataset(
                self.data_config,
                DatasetSplit.CLS_VALIDATION,
            )
            test_dataset = build_dataset(self.data_config, DatasetSplit.TEST)
            dataset: Dataset = JointDataset([cls_val_dataset, test_dataset], ["lr_val", "test"])
        else:
            # When only reporting self-supervised loss, we just use CLS_VALIDATION.
            dataset = build_dataset(
                self.data_config,
                DatasetSplit.CLS_VALIDATION,
            )
        return DataLoader(
            dataset,
            batch_size=self.context.get_per_slot_batch_size(),
            pin_memory=True,
            num_workers=self.data_config.num_workers,
        )

    def train_batch(
        self, batch: TorchData, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        if self.training_mode == TrainingMode.SELF_SUPERVISED:
            return self._train_self_supervised_batch(batch, epoch_idx, batch_idx)
        elif self.training_mode == TrainingMode.CLASSIFIER_ONLY:
            return self._train_classifier_batch(batch, epoch_idx, batch_idx)
        else:
            # Should be unreachable.
            raise Exception(f"Unknown TrainingMode {self.training_mode}.")

    def _train_self_supervised_batch(
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
        # Dummy shape needed to bypass check inside byol_model.forward.
        x = AttrDict({"first": imgs1, "second": imgs2, "shape": [len(imgs1)]})
        loss = self.byol_model.forward(x)
        self.context.backward(loss)
        self.context.step_optimizer(self.byol_opt)
        set_ema_beta_cosine_anneal(
            self.hparams, self.byol_model, batch_idx, cast(int, self.context._epoch_len)
        )
        # Note: EMA requires an aggregation_frequency of 1 or online network may not be synced
        # across processes at this point.
        self.byol_model.update_moving_average()
        return {"loss": loss}

    def _train_classifier_batch(
        self, batch: TorchData, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        imgs, labels = batch
        labels = cast(torch.Tensor, labels)
        embeddings = self.byol_model.forward(
            imgs, return_embedding=True, return_projection=False
        ).detach()
        losses = {}
        for lr_idx, lr in enumerate(self.hparams.classifier.learning_rates):
            model = self.cls_models[lr_idx]
            opt = self.cls_opts[lr_idx]
            logits = cast(torch.Tensor, model(embeddings))
            cls_loss = classifier_loss(self.hparams, logits, labels)
            losses[f"cls_loss_{lr_idx}"] = cls_loss
            self.context.backward(cls_loss)
            self.context.step_optimizer(opt)
        return losses

    def evaluate_batch(self, batch: TorchData, batch_idx: int) -> Dict[str, Any]:
        if self._should_evaluate_classifier():
            return self._evaluate_classifier_batch(batch, batch_idx)
        else:
            return self._evaluate_self_supervised_batch(batch, batch_idx)

    def _evaluate_classifier_batch(self, batch: TorchData, batch_idx: int) -> Dict[str, Any]:
        # Evaluation batches run simultaneously over LR validation subset and test dataset (using JointDataset)
        # We use a custom reducer to report test performance on the best validated LR.
        imgs, labels, d_idx = batch
        embeddings = self.byol_model.forward(imgs, return_embedding=True, return_projection=False)
        correct_cts_val = {}
        correct_cts_test = {}
        test_ct = cast(int, torch.sum(cast(torch.Tensor, d_idx) == 1).item())
        for lr_idx, lr in enumerate(self.hparams.classifier.learning_rates):
            model = self.cls_models[lr_idx]
            logits = cast(torch.Tensor, model(embeddings))
            preds = torch.argmax(logits, 1)
            correct_cts_val[lr_idx] = cast(int, torch.sum((preds == labels) * (d_idx == 0)).item())
            correct_cts_test[lr_idx] = cast(int, torch.sum((preds == labels) * (d_idx == 1)).item())
        self.final_acc_reducer.update(correct_cts_val, correct_cts_test, test_ct)
        return {}

    def _evaluate_self_supervised_batch(self, batch: TorchData, batch_idx: int) -> Dict[str, Any]:
        imgs, labels = batch
        # During evaluation, we use the same transformed image for both inputs.
        x = AttrDict({"first": imgs, "second": imgs, "shape": [len(imgs)]})
        loss = self.byol_model.forward(x)
        return {"validation_loss": loss}


class ClassifierTrainCallback(PyTorchCallback):
    """
    When training self-supervised part of the network, performs classifier head training on frozen
    network before each validation epoch.  This classifier is needed to provide validation statistics.

    Number of classifier training epochs should be kept short in self-supervised training mode, since
    there is no checkpointing within this training loop.
    """

    def __init__(self, trial: BYOLTrial):
        self.trial = trial

    def on_validation_start(self) -> None:
        trial = self.trial
        # Only perform classifier training here to give validation stats
        if not (trial._should_train_classifier_before_validation()):
            return
        # Reset models and optimization before each validation epoch -- want them trained from
        # the most recent frozen model.
        for lr_idx in range(len(trial.hparams.classifier.learning_rates)):
            trial.cls_models[lr_idx].train()
            reset_model_parameters(trial.cls_models[lr_idx])
            reset_sgd_optimizer(trial.cls_opts[lr_idx])
        print(f"Training classifier heads for {trial.hparams.classifier.train_epochs} epochs...")
        with torch.enable_grad():
            for epoch_idx in range(trial.hparams.classifier.train_epochs):
                print(f"Training epoch {epoch_idx}")
                for batch_idx, batch in enumerate(trial.train_cls_dataloader):
                    imgs, labels = batch
                    imgs = imgs.cuda()
                    labels = labels.cuda()
                    losses = trial._train_classifier_batch((imgs, labels), epoch_idx, batch_idx)
                    # Record avg loss over last epoch.
                    if epoch_idx == trial.hparams.classifier.train_epochs - 1:
                        for cls_loss_key, cls_loss in losses.items():
                            idx = int(cls_loss_key.split("_")[-1])
                            trial.cls_loss_reducers[idx].update(cls_loss.item())
        # Set models back to eval mode.
        for lr_idx in range(len(trial.hparams.classifier.learning_rates)):
            trial.cls_models[lr_idx].eval()
        print("Done training classifier heads.")
