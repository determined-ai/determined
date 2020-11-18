"""
This example is largely based on the XNLI text-classification example in the huggingface
transformers library. The license for the example in the transformer's library is reproduced below.

==================================================================================================

Copyright 2018 The Google AI Language Team Authors and The HuggingFace Inc. team.
Copyright (c) 2018, NVIDIA CORPORATION.  All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
"""

import functools
import logging
from typing import Dict, Union

import attrdict
import datasets
import numpy as np
import transformers

import determined.pytorch as det_torch
import model_hub.huggingface as hf
import model_hub.utils as utils


class XNLITrial(hf.BaseTransformerTrial):
    def __init__(self, context: det_torch.PyTorchTrialContext) -> None:
        self.logger = logging.getLogger(__name__)
        self.hparams = attrdict.AttrDict(context.get_hparams())
        self.data_config = attrdict.AttrDict(context.get_data_config())
        self.context = context

        # Load dataset and get metadata.
        # This needs to be done before we initialize the HF config, tokenizer, and model
        # because we need to know num_labels before doing so.
        if self.data_config.train_language is None:
            train_dataset = datasets.load_dataset("xnli", self.data_config.language, split="train")
        else:
            train_dataset = datasets.load_dataset(
                "xnli", self.data_config.train_language, split="train"
            )
        eval_dataset = datasets.load_dataset("xnli", self.data_config.language, split="validation")

        self.raw_datasets = {"train": train_dataset, "validation": eval_dataset}
        label_list = train_dataset.features["label"].names
        self.hparams.num_labels = len(label_list)

        super(XNLITrial, self).__init__(context)
        self.logger.info(self.config)

        # We need to create the tokenized dataset after init because we need to model and
        # tokenizer to be available.
        self.tokenized_datasets = self.build_datasets()
        train_length = len(self.tokenized_datasets["train"])
        self.logger.info("training records: {}".format(train_length))
        if (
            "records_per_epoch" in self.exp_config
            and train_length != self.exp_config["records_per_epoch"]
        ):
            self.logger.warning(
                "number of train records {} does not match records_per_epoch of {}".format(
                    train_length, self.exp_config["records_per_epoch"]
                )
            )

        # Create metric reducer
        metric = datasets.load_metric("xnli", timeout=200)

        def compute_metrics(pred_labels) -> Dict:
            preds, labels = zip(*pred_labels)
            preds = utils.expand_like(preds)
            labels = utils.expand_like(labels)
            preds = np.argmax(preds, axis=1)
            return metric.compute(predictions=preds, references=labels)

        self.reducer = context.experimental.wrap_reducer(compute_metrics, for_training=False)

    def build_datasets(self) -> Dict[str, Union[datasets.Dataset, datasets.DatasetDict]]:
        if self.data_config.pad_to_max_length:
            padding = "max_length"
        else:
            # We will pad later, dynamically at batch creation to the max_seq_length in each batch.
            padding = False

        # We cannot use self.tokenizer as a non-local variable in the preprocess_function if we
        # want map to be able to cache the output of the tokenizer.  Hence, the preprocess_function
        # takes a tokenizer explicitly as an input and we create a closure using functools.partial.
        def preprocess_function(tokenizer, padding, max_length, examples):
            # Tokenize the texts
            return tokenizer(
                examples["premise"],
                examples["hypothesis"],
                padding=padding,
                max_length=max_length,
                truncation=True,
            )

        train_dataset = self.raw_datasets["train"].map(
            functools.partial(
                preprocess_function, self.tokenizer, padding, self.data_config.max_seq_length
            ),
            batched=True,
            load_from_cache_file=not self.data_config.overwrite_cache,
        )
        eval_dataset = self.raw_datasets["validation"].map(
            functools.partial(
                preprocess_function, self.tokenizer, padding, self.data_config.max_seq_length
            ),
            batched=True,
            load_from_cache_file=not self.data_config.overwrite_cache,
        )

        if self.data_config.pad_to_max_length:
            self.collator = transformers.default_data_collator
        else:
            collator = transformers.DataCollatorWithPadding(
                self.tokenizer, pad_to_multiple_of=8 if self.hparams.use_apex_amp else None
            )
            self.collator = lambda x: collator(x).data

        return {"train": train_dataset, "validation": eval_dataset}

    def build_training_data_loader(self) -> det_torch.DataLoader:
        return det_torch.DataLoader(
            self.tokenized_datasets["train"],
            batch_size=self.context.get_per_slot_batch_size(),
            collate_fn=self.collator,
        )

    def build_validation_data_loader(self) -> det_torch.DataLoader:
        eval_dataset = self.tokenized_datasets["validation"]
        return det_torch.DataLoader(
            eval_dataset,
            batch_size=self.context.get_per_slot_batch_size(),
            collate_fn=self.collator,
        )

    def evaluate_batch(self, batch: det_torch.TorchData, batch_idx: int) -> Dict:
        outputs = self.model(**batch)
        tmp_eval_loss, logits = outputs[:2]
        preds = logits.detach().cpu().numpy()
        out_label_ids = batch["labels"].detach().cpu().numpy()
        self.reducer.update((preds, out_label_ids))
        # We will return just the metrics outputed by the reducer.
        return {}
