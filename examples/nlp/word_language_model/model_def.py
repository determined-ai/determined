"""
This example is to show how to use an the PyTorch Word Language Modeling example with Determined.
The flags and configurations can be found under const.yaml for single GPU training, and distributed.yaml
for distributed training across 8 GPUs. For more information
regarding the optional flags view the original script linked below.
This implementation is based on:
https://github.com/pytorch/examples/tree/master/word_language_model
"""
from pathlib import Path
from typing import Dict, Sequence, Union

import torch
import torch.nn as nn
from determined.pytorch import (
    DataLoader,
    LRScheduler,
    PyTorchTrial,
    PyTorchTrialContext,
)

import data
from model import RNNModel, TransformerModel

TorchData = Union[Dict[str, torch.Tensor], Sequence[torch.Tensor], torch.Tensor]


class WordLanguageModelTrial(PyTorchTrial):
    def __init__(self, context: PyTorchTrialContext):
        self.context = context
        data_config = self.context.get_data_config()
        hparams = self.context.get_hparams()
        using_bind_mount = data_config["use_bind_mount"]
        use_cache = data_config["use_cache"]
        self.eval_batch_size = data_config["eval_batch_size"]

        download_directory = (
            Path(data_config["bind_mount_path"]) if using_bind_mount else Path("/data")
        ) / f"data-rank{self.context.distributed.get_rank()}"

        self.corpus = data.load_and_cache_dataset(download_directory, use_cache)
        self.model_cls = hparams["model_cls"]
        emsize = hparams["word_embeddings_size"]
        num_hidden = hparams["num_hidden"]
        num_layers = hparams["num_layers"]
        dropout = hparams["dropout"]
        self.bptt = hparams["bptt"]

        if self.model_cls.lower() == "transformer":
            num_heads = hparams["num_heads"]
            self.model = TransformerModel(
                self.corpus.ntokens, emsize, num_heads, num_hidden, num_layers, dropout
            )
        else:
            tied = hparams["tied"]
            self.model = RNNModel(
                self.model_cls,
                self.corpus.ntokens,
                emsize,
                num_hidden,
                num_layers,
                dropout,
                tied,
            )

        self.model = self.context.wrap_model(self.model)
        self.criterion = nn.NLLLoss()

        lr = hparams["lr"]
        optimizer = torch.optim.SGD(self.model.parameters(), lr=lr)
        self.optimizer = self.context.wrap_optimizer(optimizer)

        self.lr_scheduler = self.context.wrap_lr_scheduler(
            torch.optim.lr_scheduler.ReduceLROnPlateau(
                self.optimizer,
                factor=0.25,
                patience=0,
                threshold=0.001,
                threshold_mode="abs",
                verbose=True,
            ),
            LRScheduler.StepMode.MANUAL_STEP,
        )

    def build_training_data_loader(self) -> DataLoader:
        train_dataset = data.WikiTextDataset(
            self.corpus,
            batch_size=self.context.get_per_slot_batch_size(),
        )
        batch_samp = data.BatchSamp(train_dataset, self.bptt)
        return DataLoader(train_dataset, batch_sampler=batch_samp)

    def build_validation_data_loader(self) -> DataLoader:
        val_dataset = data.WikiTextDataset(
            self.corpus,
            batch_size=self.eval_batch_size,
            valid=True,
        )
        self.val_data_len = len(val_dataset) - 1
        batch_samp = data.BatchSamp(val_dataset, self.bptt)
        return DataLoader(val_dataset, batch_sampler=batch_samp)

    def train_batch(
        self, batch: TorchData, epoch_idx: int, batch_idx: int
    ) -> Dict[str, Union[torch.Tensor, float]]:
        if batch_idx == 0 and self.model_cls.lower() != "transformer":
            self.hidden = self.model.init_hidden(self.context.get_per_slot_batch_size())
        inputs = batch[:-1]
        labels = batch[1:].view(-1)
        if self.model_cls.lower() == "transformer":
            output = self.model(inputs)
            output = output.view(-1, self.corpus.ntokens)
        else:
            self.hidden = self.model.repackage_hidden(self.hidden)
            output, self.hidden = self.model(inputs, self.hidden)
        loss = self.criterion(output, labels)

        self.context.backward(loss)
        self.context.step_optimizer(
            self.optimizer,
            clip_grads=lambda params: torch.nn.utils.clip_grad_norm_(
                params, self.context.get_hparam("max_grad_norm")
            ),
        )
        return {"loss": loss, "lr": float(self.optimizer.param_groups[0]["lr"])}

    def evaluate_full_dataset(self, data_loader: DataLoader) -> Dict[str, torch.Tensor]:
        validation_loss = 0.0
        if self.model_cls.lower() != "transformer":
            self.hidden = self.model.init_hidden(self.eval_batch_size)
        for batch in data_loader:
            batch = self.context.to_device(batch)
            if self.model_cls.lower() == "transformer":
                output = self.model(batch[:-1])
                output = output.view(-1, self.corpus.ntokens)
            else:
                output, self.hidden = self.model(batch[:-1], self.hidden)
                self.hidden = self.model.repackage_hidden(self.hidden)
            validation_loss += (
                len(batch[:-1]) * self.criterion(output, batch[1:].view(-1)).item()
            )

        validation_loss /= len(data_loader.dataset) - 1
        self.lr_scheduler.step(validation_loss)
        if self.model_cls.lower() != "transformer":
            self.hidden = self.model.init_hidden(self.context.get_per_slot_batch_size())
        return {"validation_loss": validation_loss}
