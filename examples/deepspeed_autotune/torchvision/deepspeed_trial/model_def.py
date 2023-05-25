from typing import Any, Dict

import deepspeed
import torch
import torch.nn as nn
from attrdict import AttrDict
from torch.utils.data import Dataset
from torchvision import models

from determined.pytorch import DataLoader, dsat
from determined.pytorch.deepspeed import DeepSpeedTrial, DeepSpeedTrialContext


class RandImageNetDataset(Dataset):
    """
    A fake, ImageNet-like dataset which only actually contains `num_actual_datapoints` independent
    datapoints, but pretends to have the number reported in `__len__`. Used for speed and
    simplicity. Replace with your own ImageNet-like dataset as desired.
    """

    def __init__(self, num_actual_datapoints: int = 128) -> None:
        self.num_actual_datapoints = num_actual_datapoints
        self.imgs = torch.randn(self.num_actual_datapoints, 3, 224, 224)
        self.labels = torch.randint(1000, size=(self.num_actual_datapoints,))

    def __len__(self) -> int:
        return 10**6

    def __getitem__(self, idx: int) -> torch.Tensor:
        img = self.imgs[idx % self.num_actual_datapoints]
        label = self.labels[idx % self.num_actual_datapoints]
        return img, label


class TorchvisionTrial(DeepSpeedTrial):
    def __init__(self, context: DeepSpeedTrialContext) -> None:
        self.context = context
        self.hparams = AttrDict(self.context.get_hparams())

        model = getattr(models, self.hparams.model_name)()
        parameters = filter(lambda p: p.requires_grad, model.parameters())

        ds_config = dsat.get_ds_config_from_hparams(self.hparams)
        model_engine, _, _, _ = deepspeed.initialize(
            model=model, model_parameters=parameters, config=ds_config
        )

        self.fp16 = model_engine.fp16_enabled()
        self.model_engine = self.context.wrap_model_engine(model_engine)

        self.criterion = nn.CrossEntropyLoss().to(self.context.device)

    def train_batch(self, iter_dataloader, epoch_idx, batch_idx) -> Dict[str, torch.Tensor]:
        inputs, labels = self.context.to_device(next(iter_dataloader))
        if self.fp16:
            inputs = inputs.half()
        outputs = self.model_engine(inputs)
        loss = self.criterion(outputs, labels)

        self.model_engine.backward(loss)
        self.model_engine.step()
        return {"train_loss": loss.item()}

    def evaluate_batch(self, iter_dataloader, batch_idx) -> Dict[str, Any]:
        inputs, labels = self.context.to_device(next(iter_dataloader))
        if self.fp16:
            inputs = inputs.half()
        outputs = self.model_engine(inputs)
        loss = self.criterion(outputs, labels)
        return {"val_loss": loss.item()}

    def build_training_data_loader(self) -> Any:
        trainset = RandImageNetDataset()
        train_loader = DataLoader(
            trainset,
            batch_size=self.context.train_micro_batch_size_per_gpu,
            shuffle=True,
            num_workers=2,
        )
        return train_loader

    def build_validation_data_loader(self) -> Any:
        testset = RandImageNetDataset()
        return DataLoader(
            testset,
            batch_size=self.context.train_micro_batch_size_per_gpu,
            shuffle=False,
            num_workers=2,
        )
