"""
This example is to show how to use an existing PyTorch model with Determined.
The flags and configurations can be found under const.yaml. For more information
regarding the optional flas view the original script linked below.

This implementation is based on:
https://github.com/huggingface/transformers/blob/v2.2.0/examples/run_glue.py

"""
from typing import Dict, Sequence, Union

import numpy as np
import torch
from torch import nn

from determined.pytorch import DataLoader, LRScheduler, PyTorchTrial, PyTorchTrialContext
from transformers import AdamW, get_linear_schedule_with_warmup
from transformers import AutoConfig, AutoModelForSequenceClassification
from datasets import load_dataset, load_metric

import data

TorchData = Union[Dict[str, torch.Tensor], Sequence[torch.Tensor], torch.Tensor]


class BertPytorch(PyTorchTrial):
    def __init__(self, context: PyTorchTrialContext) -> None:
        self.context = context

        raw_dataset = load_dataset("glue", self.context.get_data_config().get('task').lower())
        label_list = raw_dataset["train"].features["label"].names
        num_labels = len(label_list)

        self.metric = load_metric("glue", self.context.get_data_config().get('task').lower())

        cache_dir_per_rank = f"/tmp/{self.context.distributed.get_rank()}"
        config = AutoConfig.from_pretrained(
            self.context.get_data_config().get("model_name_or_path"),
            num_labels=num_labels,
            finetuning_task=self.context.get_data_config().get("task").lower(),
            cache_dir=cache_dir_per_rank,
        )
        self.model = self.context.wrap_model(AutoModelForSequenceClassification.from_pretrained(
            self.context.get_data_config().get("model_name_or_path"),
            from_tf=(".ckpt" in self.context.get_data_config().get("model_name_or_path")),
            config=config,
            cache_dir=cache_dir_per_rank,
        ))

        no_decay = ["bias", "LayerNorm.weight"]
        optimizer_grouped_parameters = [
            {
                "params": [
                    p for n, p in self.model.named_parameters() if not any(nd in n for nd in no_decay)
                ],
                "weight_decay": self.context.get_hparam("weight_decay"),
            },
            {
                "params": [
                    p for n, p in self.model.named_parameters() if any(nd in n for nd in no_decay)
                ],
                "weight_decay": 0.0,
            },
        ]
        self.optimizer = self.context.wrap_optimizer(AdamW(
            optimizer_grouped_parameters,
            lr=self.context.get_hparam("learning_rate"),
            eps=self.context.get_hparam("adam_epsilon"),
        ))
        self.lr_scheduler = self.context.wrap_lr_scheduler(
            get_linear_schedule_with_warmup(
                self.optimizer,
                num_warmup_steps=self.context.get_hparam("num_warmup_steps"),
                num_training_steps=self.context.get_hparam("num_training_steps"),
            ),
            LRScheduler.StepMode.STEP_EVERY_BATCH,
        )

    def build_training_data_loader(self) -> DataLoader:
        train_dataset = data.load_and_cache_examples(
            config=self.context.get_data_config(),
            max_seq_length=self.context.get_hparam("max_seq_length"),
            evaluate=False,
        )
        return DataLoader(train_dataset, batch_size=self.context.get_per_slot_batch_size())

    def build_validation_data_loader(self) -> DataLoader:
        test_dataset = data.load_and_cache_examples(
            config=self.context.get_data_config(),
            max_seq_length=self.context.get_hparam("max_seq_length"),
            evaluate=True,
        )
        return DataLoader(test_dataset, batch_size=self.context.get_per_slot_batch_size())

    def get_metrics(self, outputs, inputs):
        """
        Based on outputs calculate the metrics
        """
        loss, logits = outputs[:2]

        preds = logits.detach().cpu().numpy()
        out_labels_ids = inputs["labels"].detach().cpu().numpy()
        if self.context.get_data_config()["output_mode"] == "classification":
           preds = np.argmax(preds, axis=1)
        elif self.context.get_data_config()["output_mode"] == "regression":
           preds = np.squeeze(preds)

        self.metric.add_batch(predictions=preds, references=out_labels_ids)
        results = self.metric.compute()
        results["loss"] = loss
        return results

    def train_batch(self, batch: TorchData, epoch_idx: int, batch_idx: int):
        """
        Trains the provided batch.
        Returns: Dictionary of the calculated Metrics
        """

        inputs = {"input_ids": batch[0], "attention_mask": batch[1], "labels": batch[-1]}
        inputs["token_type_ids"] = (
            batch[2] if self.context.get_hparam("model_type") in ["bert", "xlnet"] else None
        )

        outputs = self.model(**inputs)
        results = self.get_metrics(outputs, inputs)

        self.context.backward(results["loss"])
        self.context.step_optimizer(self.optimizer)

        return results

    def evaluate_batch(self, batch: TorchData):
        """
        Evaluates the provided batch.
        Returns: Dictionary of the calculated Metrics
        """
        inputs = {"input_ids": batch[0], "attention_mask": batch[1], "labels": batch[3]}
        inputs["token_type_ids"] = (
            batch[2] if self.context.get_hparam("model_type") in ["bert", "xlnet"] else None
        )
        outputs = self.model(**inputs)
        results = self.get_metrics(outputs, inputs)
        return results
