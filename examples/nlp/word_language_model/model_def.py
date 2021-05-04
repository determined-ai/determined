from typing import Dict, Sequence, Union
import torch
import torch.nn as nn
import time
from pathlib import Path

from determined.pytorch import (
    DataLoader,
    PyTorchTrial,
    PyTorchTrialContext,
    LRScheduler,
)
import data
from model import TransformerModel, RNNModel

TorchData = Union[Dict[str, torch.Tensor], Sequence[torch.Tensor], torch.Tensor]


class WordLanguageModelPyTorch(PyTorchTrial):
    def __init__(self, context: PyTorchTrialContext):
        self.context = context
        data_config = self.context.get_data_config()
        hparams = self.context.get_hparams()
        self.using_bind_mount = data_config.get("use_bind_mount", False)
        self.use_cache = data_config.get("use_cache", True)
        self.bind_mount_path = (
            Path(data_config.get("bind_mount_path")) if self.using_bind_mount else None
        )

        download_directory = (
            self.bind_mount_path if self.using_bind_mount else Path("/tmp")
        )
        download_directory = (
            self.download_directory / f"data-rank{self.context.distributed.get_rank()}"
        )
        self.corpus = data.load_and_cache_dataset(download_directory, use_cache)
        emsize = hparams.get("word_embeddings_size", 200)
        self.model_cls = hparams.get("model_type", "transformer")
        num_heads = hparams.get("num_heads", 2)
        num_hidden = hparams.get("num_hidden", 200)
        num_layers = hparams.get("num_layers", 2)
        dropout = hparams.get("dropout", 0.2)
        tied = hparams.get("tied", False)
        self.bptt = hparams.get("bptt", 35)

        if self.model_cls.lower() == "transformer":
            self.model = TransformerModel(
                self.corpus.ntokens, emsize, num_heads, num_hidden, num_layers, dropout
            )
        else:
            self.model = RNNModel(
                model_cls,
                self.corpus.ntokens,
                emsize,
                num_hidden,
                num_layers,
                dropout,
                tied,
            )

        self.model = self.context.wrap_model(self.model)
        self.criterion = nn.NLLLoss()

        lr = hparams.get("lr", 20)
        optimizer = torch.optim.SGD(self.model.named_parameters(), lr=lr)
        self.optimizer = self.context.wrap_optimizer(optimizer)

        self.lr_scheduler = self.context.wrap_lr_scheduler(
            torch.optim.lr_scheduler.ExponentialLR(self.optimizer, 4.0)
        )

    def build_training_data_loader(self):
        train_dataset = data.WikiTextDataset(
            self.corpus,
            batch_size=self.context.get_per_slot_batch_size(),
            use_cache=self.use_cache,
        )
        batch_samp = data.BatchSamp(train_dataset, self.bptt)
        return DataLoader(train_dataset, batch_sampler=batch_samp)

    def build_validation_data_loader(self):
        val_dataset = data.WikiTextDataset(
            self.corpus,
            batch_size=self.context.get_per_slot_batch_size(),
            use_cache=self.use_cache,
            valid=True,
        )
        batch_samp = data.BatchSamp(val_dataset, self.bptt)
        return DataLoader(val_dataset, batch_sampler=batch_samp)

    def train_batch(self, batch: TorchData, epoch_idx: int, batch_idx: int):
        if batch_idx == 0 and self.model_cls.lower != "transformer":
            self.hidden = self.model.init_hidden(self.context.get_per_slot_batch_size())
        inputs = batch[:-1]
        labels = batch[1:].view(-1)
        if self.model_cls.lower == "transformer":
            output = self.model(inputs)
            output = output.view(-1, self.corpus.ntokens)
        else:
            self.hidden = model.repackage_hidden(self.hidden)
            output, self.hidden = self.model(inputs, self.hidden)
        loss = self.criterion(output, labels)

        self.context.backward(loss)
        self.context.step_optimizer(
            self.optimizer,
            clip_grads=lambda params: torch.nn.utils.clip_grad_norm_(
                params, self.context.get_hparam("max_grad_norm")
            ),
        )
        return {"loss": loss, "lr": float(self.lr_scheduler.get_last_lr()[0])}

    def evaluate_full_dataset(self, data_loader: DataLoader) -> float:
        total_loss = 0.0
        if self.model_cls.lower() != "transformer":
            self.hidden = self.model.init_hidden(self.context.get_per_slot_batch_size())
        for batch in data_loader:
            if args.model.lower() == "transformer":
                output = model(batch[:-1])
                output = output.view(-1, self.corpus.ntokens)
            else:
                output, self.hidden = model(batch[:-1], self.hidden)
                self.hidden = model.repackage_hidden(self.hidden)
            total_loss += (
                len(batch[:-1]) * self.criterion(output, batch[1:].view(-1)).item()
            )
        return total_loss / len(data_loader.dataset)
