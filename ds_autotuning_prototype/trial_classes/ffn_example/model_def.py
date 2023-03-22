import logging
from typing import Any, Dict

import deepspeed
import torch
import torch.nn as nn
import torch.nn.functional as F
from attrdict import AttrDict
from determined.pytorch import DataLoader
from determined.pytorch.deepspeed import (
    DeepSpeedTrial,
    DeepSpeedTrialContext,
    get_ds_config_from_hparams,
)
from torch.utils.data import Dataset


class RandDataset(Dataset):
    def __init__(self, dim: int, num_actual_datapoints: int = 128) -> None:
        self.num_actual_datapoints = num_actual_datapoints
        self.dim = dim
        self.data = torch.randn(self.num_actual_datapoints, self.dim)

    def __len__(self) -> int:
        return 10 ** 6

    def __getitem__(self, idx: int) -> torch.Tensor:
        data = self.data[idx % self.num_actual_datapoints]
        return data


class MinimalModel(nn.Module):
    def __init__(self, dim: int, layers: int) -> None:
        super().__init__()
        self.dim = dim
        layers = [nn.Linear(dim, dim, bias=False) for _ in range(layers)]
        self.model = nn.ModuleList(layers)

    def forward(self, inputs: torch.Tensor) -> torch.Tensor:
        outputs = inputs
        for layer in self.model:
            outputs = layer(outputs)
        return outputs


class FNNTrial(DeepSpeedTrial):
    def __init__(self, context: DeepSpeedTrialContext) -> None:
        self.context = context
        self.args = AttrDict(self.context.get_hparams())
        model = MinimalModel(self.args.dim, self.args.layers)
        parameters = filter(lambda p: p.requires_grad, model.parameters())
        logging.info(f"Seeing args:{self.args}")

        ds_config = get_ds_config_from_hparams(self.args)
        logging.info(f"Using ds_config: {ds_config}")
        model_engine, optimizer, __, __ = deepspeed.initialize(
            model=model, model_parameters=parameters, config=ds_config
        )

        self.fp16 = model_engine.fp16_enabled()
        self.model_engine = self.context.wrap_model_engine(model_engine)

        self.criterion = nn.MSELoss().to(self.context.device)
        self.reducer = self.context.wrap_reducer(
            lambda x: sum([m[0] for m in x]) / sum([m[1] for m in x]),
            "accuracy",
            for_training=False,
        )

    def train_batch(self, iter_dataloader, epoch_idx, batch_idx) -> Dict[str, torch.Tensor]:
        batch = self.context.to_device(next(iter_dataloader))
        if self.fp16:
            batch = batch.half()
        outputs = self.model_engine(batch)
        loss = self.criterion(outputs, batch)

        self.model_engine.backward(loss)
        self.model_engine.step()
        return {"loss": loss.item()}

    def evaluate_batch(self, iter_dataloader, batch_idx) -> Dict[str, Any]:
        """
        Calculate validation metrics for a batch and return them as a dictionary.
        This method is not necessary if the user defines evaluate_full_dataset().
        """
        return {}

    def build_training_data_loader(self) -> Any:
        trainset = RandDataset(self.args.dim)
        train_loader = DataLoader(
            trainset,
            batch_size=self.context.train_micro_batch_size_per_gpu,
            shuffle=True,
            num_workers=2,
        )
        return train_loader

    def build_validation_data_loader(self) -> Any:
        testset = RandDataset(self.args.dim)
        return DataLoader(
            testset,
            batch_size=self.context.train_micro_batch_size_per_gpu,
            shuffle=False,
            num_workers=2,
        )
