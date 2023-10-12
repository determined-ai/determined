import datetime
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
    overwrite_deepspeed_config,
)


class Net(nn.Module):
    def __init__(self, args):
        super(Net, self).__init__()
        self.args = args
        self.conv1 = nn.Conv2d(3, 6, 5)
        self.pool = nn.MaxPool2d(2, 2)
        self.conv2 = nn.Conv2d(6, 16, 5)
        self.fc1 = nn.Linear(16 * 5 * 5, 120)
        self.fc2 = nn.Linear(120, 84)
        if args.moe:
            fc3 = nn.Linear(84, 84)
            self.moe_layer_list = []
            for n_e in args.num_experts:
                # create moe layers based on the number of experts
                self.moe_layer_list.append(
                    deepspeed.moe.layer.MoE(
                        hidden_size=84,
                        expert=fc3,
                        num_experts=n_e,
                        ep_size=args.ep_world_size,
                        use_residual=args.mlp_type == "residual",
                        k=args.top_k,
                        min_capacity=args.min_capacity,
                        noisy_gate_policy=args.noisy_gate_policy,
                    )
                )
            self.moe_layer_list = nn.ModuleList(self.moe_layer_list)
            self.fc4 = nn.Linear(84, 10)
        else:
            self.fc3 = nn.Linear(84, 10)

    def forward(self, x):
        x = self.pool(F.relu(self.conv1(x)))
        x = self.pool(F.relu(self.conv2(x)))
        x = x.view(-1, 16 * 5 * 5)
        x = F.relu(self.fc1(x))
        x = F.relu(self.fc2(x))
        if self.args.moe:
            for layer in self.moe_layer_list:
                x, _, _ = layer(x)
            x = self.fc4(x)
        else:
            x = self.fc3(x)
        return x


def create_moe_param_groups(model):
    from deepspeed.moe.utils import split_params_into_different_moe_groups_for_optimizer

    parameters = {"params": [p for p in model.parameters()], "name": "parameters"}

    return split_params_into_different_moe_groups_for_optimizer(parameters)


class CIFARTrial(DeepSpeedTrial):
    def __init__(self, context: DeepSpeedTrialContext) -> None:
        self.context = context
        self.args = AttrDict(self.context.get_hparams())

        model = Net(self.args)
        parameters = filter(lambda p: p.requires_grad, model.parameters())
        if self.args.moe_param_group:
            parameters = create_moe_param_groups(model)

        ds_config = overwrite_deepspeed_config(
            self.args.deepspeed_config, self.args.get("overwrite_deepspeed_args", {})
        )

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
