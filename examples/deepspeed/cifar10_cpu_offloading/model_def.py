import os
import filelock
from typing import Any, Dict
from attrdict import AttrDict
import torch
import torchvision
import torchvision.transforms as transforms
import deepspeed

from torch.nn import MaxPool2d, Conv2d, Linear, Module, CrossEntropyLoss
import torch.nn.functional as F

from determined.pytorch import DataLoader
from determined.pytorch.deepspeed import (
    DeepSpeedTrial,
    DeepSpeedTrialContext,
    overwrite_deepspeed_config,
)


class Net(Module):
    def __init__(self, args):
        super(Net, self).__init__()
        self.args = args
        self.conv1_1 = Conv2d(3, 1024, kernel_size=3, padding=1)
        self.conv1_2 = Conv2d(1024, 2048, kernel_size=3, stride=1, padding=1)
        self.pool = MaxPool2d(2, 2)
        self.conv2_1 = Conv2d(2048, 3072, kernel_size=3, stride=1, padding=1)
        self.conv2_2 = Conv2d(3072, 4096, kernel_size=3, stride=1, padding=1)
        self.fc1 = Linear(4096 * 8 * 8, 1000)
        self.fc2 = Linear(1000, 10000)
        self.fc3 = Linear(10000, 10000)
        self.fc4 = Linear(10000, 10)

    def forward(self, x):
        x = self.pool(F.relu(self.conv1_2(F.relu(self.conv1_1(x)))))
        x = self.pool(F.relu(self.conv2_2(F.relu(self.conv2_1(x)))))
        x = x.view(-1, 4096 * 8 * 8)
        x = F.relu(self.fc1(x))
        x = F.relu(self.fc2(x))
        x = F.relu(self.fc3(x))
        x = self.fc4(x)
        return x


class CIFARTrial(DeepSpeedTrial):
    def __init__(self, context: DeepSpeedTrialContext) -> None:
        self.context = context
        self.args = AttrDict(self.context.get_hparams())

        if self.args.get("deepspeed_offload") is True:
            with deepspeed.zero.Init():
                model = Net(self.args)
        else:
            model = Net(self.args)

        parameters = filter(lambda p: p.requires_grad, model.parameters())

        ds_config = overwrite_deepspeed_config(
            self.args.deepspeed_config, self.args.get("overwrite_deepspeed_args", {})
        )

        model_engine, optimizer, __, __ = deepspeed.initialize(
            model=model, model_parameters=parameters, config=ds_config
        )

        self.fp16 = model_engine.fp16_enabled()
        self.model_engine = self.context.wrap_model_engine(model_engine)

        self.criterion = CrossEntropyLoss().to(self.context.device)
        self.reducer = self.context.wrap_reducer(
            lambda x: sum([m[0] for m in x]) / sum([m[1] for m in x]),
            "accuracy",
            for_training=False,
        )

    def train_batch(
        self, iter_dataloader, epoch_idx, batch_idx
    ) -> Dict[str, torch.Tensor]:
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
        transform = transforms.Compose(
            [
                transforms.ToTensor(),
                transforms.Normalize((0.5, 0.5, 0.5), (0.5, 0.5, 0.5)),
            ]
        )

        with filelock.FileLock(os.path.join("/tmp", "train.lock")):
            trainset = torchvision.datasets.CIFAR10(
                root="/data", train=True, download=True, transform=transform
            )

        return DataLoader(
            trainset,
            batch_size=self.context.train_micro_batch_size_per_gpu,
            shuffle=True,
            num_workers=2,
        )

    def build_validation_data_loader(self) -> Any:
        transform = transforms.Compose(
            [
                transforms.ToTensor(),
                transforms.Normalize((0.5, 0.5, 0.5), (0.5, 0.5, 0.5)),
            ]
        )

        with filelock.FileLock(os.path.join("/tmp", "val.lock")):
            testset = torchvision.datasets.CIFAR10(
                root="/data", train=False, download=True, transform=transform
            )

        return DataLoader(
            testset,
            batch_size=4,
            shuffle=False,
            num_workers=2,
        )
