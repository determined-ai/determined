import logging
from typing import Any, Dict

import torch
import torch.nn as nn
from attrdict import AttrDict
from torch.utils.data import Dataset
from torchvision import models

import deepspeed
from determined.pytorch import DataLoader, dsat
from determined.pytorch.deepspeed import DeepSpeedTrial, DeepSpeedTrialContext


class RandImageNetDataset(Dataset):
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


class MinimalModel(nn.Module):
    def __init__(self, dim: int, layers: int) -> None:
        super().__init__()
        self.dim = dim
        layers = [nn.Linear(dim, dim, bias=False) for _ in range(layers)]

    def forward(self, inputs: torch.Tensor) -> torch.Tensor:
        outputs = inputs
        for layer in self.model:
            outputs = layer(outputs)
        return outputs


class TorchvisionTrial(DeepSpeedTrial):
    def __init__(self, context: DeepSpeedTrialContext) -> None:
        self.context = context
        self.args = AttrDict(self.context.get_hparams())
        model_dict = {
            "resnet152": models.resnet152,
            "wide_resnet101_2": models.wide_resnet101_2,
            "vgg19": models.vgg19,
            "regnet_x_32gf": models.regnet_x_32gf,
        }

        model = model_dict[self.args.model_name]()
        parameters = filter(lambda p: p.requires_grad, model.parameters())
        logging.info(f"Seeing args:{self.args}")

        ds_config = dsat.get_ds_config_from_hparams(self.args)
        logging.info(f"Using ds_config: {ds_config}")
        model_engine, optimizer, __, __ = deepspeed.initialize(
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
        """
        Calculate validation metrics for a batch and return them as a dictionary.
        This method is not necessary if the user defines evaluate_full_dataset().
        """
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
