import os
import filelock
import datetime
from typing import Any, Dict
from attrdict import AttrDict
import torch
import torchvision
import torchvision.transforms as transforms
import deepspeed
from deepspeed.moe.utils import is_moe_param
import torch.nn as nn
import torch.nn.functional as F

from determined.pytorch import DataLoader
from determined.pytorch.deepspeed import DeepSpeedTrial, DeepSpeedTrialContext, overwrite_deepspeed_config


class Net(nn.Module):
    def __init__(self, args):
        super(Net, self).__init__()
        self.moe = args.moe
        self.conv1 = nn.Conv2d(3, 6, 5)
        self.pool = nn.MaxPool2d(2, 2)
        self.conv2 = nn.Conv2d(6, 16, 5)
        self.fc1 = nn.Linear(16 * 5 * 5, 120)
        self.fc2 = nn.Linear(120, 84)
        if args.moe:
            self.fc3 = nn.Linear(84, 84)
            self.fc3 = deepspeed.moe.layer.MoE(
                hidden_size=84,
                expert=self.fc3,
                num_experts=args.num_experts,
                k=args.top_k,
                min_capacity=args.min_capacity,
                noisy_gate_policy=args.noisy_gate_policy,
            )
            self.fc4 = nn.Linear(84, 10)
        else:
            self.fc3 = nn.Linear(84, 10)

    def forward(self, x):
        x = self.pool(F.relu(self.conv1(x)))
        x = self.pool(F.relu(self.conv2(x)))
        x = x.view(-1, 16 * 5 * 5)
        x = F.relu(self.fc1(x))
        x = F.relu(self.fc2(x))
        if self.moe:
            x, _, _ = self.fc3(x)
            x = self.fc4(x)
        else:
            x = self.fc3(x)
        return x


def create_moe_param_groups(model):
    params_with_weight_decay = {"params": [], "name": "weight_decay_params"}
    moe_params_with_weight_decay = {
        "params": [],
        "moe": True,
        "name": "weight_decay_moe_params",
    }

    for module_ in model.modules():
        moe_params_with_weight_decay["params"].extend(
            [
                p
                for n, p in list(module_._parameters.items())
                if p is not None and is_moe_param(p)
            ]
        )
        params_with_weight_decay["params"].extend(
            [
                p
                for n, p in list(module_._parameters.items())
                if p is not None and not is_moe_param(p)
            ]
        )

    return params_with_weight_decay, moe_params_with_weight_decay


class CIFARTrial(DeepSpeedTrial):
    def __init__(self, context: DeepSpeedTrialContext) -> None:
        self.context = context
        self.args = AttrDict(self.context.get_hparams())
        if self.args.moe:
            deepspeed.utils.groups.initialize(ep_size=self.args.ep_world_size)
        model = Net(self.args)

        parameters = filter(lambda p: p.requires_grad, model.parameters())
        if self.args.moe_param_group:
            parameters = create_moe_param_groups(model)

        ds_config = overwrite_deepspeed_config(
            self.args.deepspeed_config, 
            self.args.get("overwrite_deepspeed_args", {})
        )

        model_engine, optimizer, __, __ = deepspeed.initialize(
            model=model, model_parameters=parameters, config=ds_config
        )

        self.fp16 = model_engine.fp16_enabled()
        self.model_engine = model_engine
        self.model_engine = self.context.wrap_model_engine(model_engine)

        self.criterion = nn.CrossEntropyLoss().to(self.context.device)

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
        return {"loss": float(loss)}

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

        return {"accuracy": float(correct / total)}

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
            batch_size=self.context.train_micro_batch_size_per_gpu,
            shuffle=False,
            num_workers=2,
        )
