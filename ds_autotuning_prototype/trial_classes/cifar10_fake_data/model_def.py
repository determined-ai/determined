import logging
import os
from typing import Any, Dict

import deepspeed
import filelock
import torch
import torch.nn as nn
import torch.nn.functional as F
import torchvision
import torchvision.transforms as transforms
from attrdict import AttrDict
from determined.pytorch import DataLoader
from determined.pytorch.deepspeed import (
    DeepSpeedTrial,
    DeepSpeedTrialContext,
    get_ds_config_from_hparams,
)


class RandCIFAR10Dataset(torch.utils.data.Dataset):
    def __init__(self, num_actual_datapoints: int = 128) -> None:
        self.num_actual_datapoints = num_actual_datapoints
        self.imgs = torch.randn(self.num_actual_datapoints, 3, 32, 32)
        self.labels = torch.randint(10, size=(self.num_actual_datapoints,))

    def __len__(self) -> int:
        return 10 ** 8

    def __getitem__(self, idx: int) -> torch.Tensor:
        img = self.imgs[idx % self.num_actual_datapoints]
        label = self.labels[idx % self.num_actual_datapoints]
        return img, label


class Net(nn.Module):
    def __init__(self, args):
        super(Net, self).__init__()
        self.args = args
        self.conv1 = nn.Conv2d(3, 6, 5)
        self.pool = nn.MaxPool2d(2, 2)
        self.conv2 = nn.Conv2d(6, 16, 5)
        # Adding some additional layers to make the model larger and avoid shm size errors.
        hidden_dim = 2 ** 15
        self.fc0 = nn.Linear(16 * 5 * 5, hidden_dim)
        self.more_fcls = nn.ModuleList([nn.Linear(hidden_dim, hidden_dim) for _ in range(8)])
        self.fc_last = nn.Linear(hidden_dim, 10)

    def forward(self, x):
        x = self.pool(F.relu(self.conv1(x)))
        x = self.pool(F.relu(self.conv2(x)))
        x = x.view(-1, 16 * 5 * 5)
        x = F.relu(self.fc0(x))
        for layer in self.more_fcls:
            x = F.relu(layer(x))
        x = self.fc_last(x)
        return x


class CIFARTrial(DeepSpeedTrial):
    def __init__(self, context: DeepSpeedTrialContext) -> None:
        self.context = context
        self.args = AttrDict(self.context.get_hparams())
        model = Net(self.args)
        parameters = filter(lambda p: p.requires_grad, model.parameters())
        logging.info(f"Seeing args:{self.args}")

        ds_config = get_ds_config_from_hparams(self.args)
        logging.info(f"Using ds_config: {ds_config}")
        model_engine, optimizer, __, __ = deepspeed.initialize(
            model=model, model_parameters=parameters, config=ds_config
        )

        self.fp16 = model_engine.fp16_enabled()
        self.model_engine = self.context.wrap_model_engine(model_engine)

        self.criterion = nn.CrossEntropyLoss().to(self.context.device)
        self.reducer = self.context.wrap_reducer(
            lambda x: sum([m[0] for m in x]) / sum([m[1] for m in x]),
            "accuracy",
            for_training=False,
        )

    def train_batch(self, iter_dataloader, epoch_idx, batch_idx) -> Dict[str, torch.Tensor]:
        batch = self.context.to_device(next(iter_dataloader))
        inputs, labels = batch[0], batch[1]
        if self.fp16:
            inputs = inputs.half()
        outputs = self.model_engine(inputs)
        loss = self.criterion(outputs, labels)

        self.model_engine.backward(loss)
        self.model_engine.step()
        return {"loss": loss.item()}

    def evaluate_batch(self, iter_dataloader, batch_idx) -> Dict[str, Any]:
        """
        Calculate validation metrics for a batch and return them as a dictionary.
        This method is not necessary if the user defines evaluate_full_dataset().
        """
        batch = self.context.to_device(next(iter_dataloader))
        images, labels = batch[0], batch[1]
        if self.fp16:
            images = images.half()
        outputs = self.model_engine(images)
        _, predicted = torch.max(outputs.data, 1)
        total = labels.size(0)
        correct = (predicted == labels).sum().item()
        self.reducer.update((correct, total))
        return {}

    def build_training_data_loader(self) -> Any:
        trainset = RandCIFAR10Dataset()
        train_loader = DataLoader(
            trainset,
            batch_size=self.context.train_micro_batch_size_per_gpu,
            shuffle=True,
            num_workers=2,
        )
        return train_loader

    def build_validation_data_loader(self) -> Any:
        testset = RandCIFAR10Dataset()
        return DataLoader(
            testset,
            batch_size=self.context.train_micro_batch_size_per_gpu,
            shuffle=False,
            num_workers=2,
        )
